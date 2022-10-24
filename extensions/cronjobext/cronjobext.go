package cronjobext

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/go-co-op/gocron"
	"github.com/mitchellh/mapstructure"
	"github.com/apocelipes/gobay"
	"github.com/apocelipes/gobay/extensions/asynctaskext"
)

// Config configuration of cronjobext
type Config struct {
	AsyncTaskConfig *config.Config `yaml:"-" ignored:"true"`
	BindTo          string         `yaml:"bind_to"`
	TimeZone        string         `yaml:"tz"`
	HealthCheckPort int            `yaml:"health_check_port"` // default is 5000
}

// TZ converts TimeZone to a pointer of time.Location
func (c *Config) TZ() (*time.Location, error) {
	tz, err := time.LoadLocation(c.TimeZone)
	if err != nil {
		return nil, err
	}
	return tz, nil
}

type CronJobExt struct {
	NS string
	// TimeZone scheduler's timezone
	TimeZone *time.Location

	app    *gobay.Application
	config *Config
	// for sending tasks to async task queues
	server            *asynctaskext.AsyncTaskExt
	scheduler         *gocron.Scheduler
	registeredTasks   *sync.Map
	healthCheckServer *http.Server
}

func (t *CronJobExt) Object() interface{} {
	return t
}

func (t *CronJobExt) Init(app *gobay.Application) error {
	if t.NS == "" {
		return errors.New("lack of NS")
	}
	t.app = app
	extCfg := app.Config()
	extCfg = gobay.GetConfigByPrefix(extCfg, t.NS, true)
	t.config = &Config{AsyncTaskConfig: &config.Config{}, TimeZone: "UTC", HealthCheckPort: 5000}
	if err := extCfg.Unmarshal(t.config, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
		config.Squash = true
	}); err != nil {
		return err
	}

	// bind to other AsyncTaskExt configurations
	asyncTaskNS := t.config.BindTo
	asyncExtCfg := app.Config()
	asyncExtCfg = gobay.GetConfigByPrefix(asyncExtCfg, asyncTaskNS, true)
	asyncConf := &config.Config{}
	if err := asyncExtCfg.Unmarshal(asyncConf, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	}); err != nil {
		return err
	}
	t.config.AsyncTaskConfig = asyncConf
	t.server = &asynctaskext.AsyncTaskExt{
		NS: asyncTaskNS,
	}
	if err := t.server.Init(app); err != nil {
		return err
	}

	tz, err := t.config.TZ()
	if err != nil {
		return err
	}
	t.scheduler = gocron.NewScheduler(tz)
	t.TimeZone = tz
	t.registeredTasks = &sync.Map{}
	return nil
}

func (t *CronJobExt) Application() *gobay.Application {
	return t.app
}

func (t *CronJobExt) Close() error {
	t.scheduler.Stop()
	if t.healthCheckServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return t.healthCheckServer.Shutdown(ctx)
	}
	return nil
}

type CronJobSchedulerType string

const (
	DurationScheduler = "duration" // time.ParseDuration style expressions
	CronScheduler     = "crontab"  // crontab style expressions
)

type CronJobTask struct {
	Type CronJobSchedulerType
	// Spec Scheduled interval or cron expression
	Spec string
	// TaskFunc the function be registered into AsyncTaskExt.
	// This function must also be registered on the async task worker side.
	TaskFunc interface{}
	// TaskSignature the args of the TaskFunc,
	// The value of tasks.Signature.Name must be the same as the Name with which the TaskFunc was registered.
	TaskSignature *tasks.Signature
}

func (t *CronJobExt) jobWrapper(task *CronJobTask) func() error {
	return func() error {
		signature := tasks.CopySignature(task.TaskSignature)
		_, err := t.server.SendTask(signature)
		if err != nil {
			log.ERROR.Printf("send task failed: %v", err)
		}
		return err
	}
}

// RemoveAllJobs stops and deletes all scheduled tasks, but the scheduler itself does not stop running
func (t *CronJobExt) RemoveAllJobs() {
	t.scheduler.Clear()
}

// RegisterTasks this function can only be used before the scheduler starting.
func (t *CronJobExt) RegisterTasks(ts map[string]*CronJobTask) error {
	if t.scheduler.IsRunning() {
		return errors.New("scheduler is running")
	}
	for _, ct := range ts {
		// TaskFunc with same task names will only register once
		_, loaded := t.registeredTasks.LoadOrStore(ct.TaskSignature.Name, struct{}{})
		if !loaded {
			if err := t.server.RegisterWorkerHandler(ct.TaskSignature.Name, ct.TaskFunc); err != nil {
				return err
			}
		}
		switch ct.Type {
		case DurationScheduler:
			_, err := t.scheduler.Every(ct.Spec).Do(t.jobWrapper(ct))
			if err != nil {
				t.RemoveAllJobs()
				return err
			}
		case CronScheduler:
			_, err := t.scheduler.Cron(ct.Spec).Do(t.jobWrapper(ct))
			if err != nil {
				t.RemoveAllJobs()
				return err
			}
		}
	}
	return nil
}

// StartCronJob this function will be blocked until the scheduler exits
func (t *CronJobExt) StartCronJob(enableHealthCheck bool) {
	if t.scheduler.IsRunning() {
		return
	}
	if enableHealthCheck {
		t.healthCheckServer = &http.Server{Addr: fmt.Sprintf(":%v", t.config.HealthCheckPort)}
		http.Handle("/health", http.HandlerFunc(t.healthHttpHandler))
		go func() {
			if err := t.healthCheckServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.FATAL.Printf("error when starting health check server: %v\n", err)
			}
		}()
	}
	t.scheduler.StartBlocking()
}

func (t *CronJobExt) healthHttpHandler(w http.ResponseWriter, _ *http.Request) {
	if !t.scheduler.IsRunning() {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte("scheduler down!")); err != nil {
			panic(err)
		}
		return
	}

	for _, j := range t.scheduler.Jobs() {
		if j.Error() != nil {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(j.Error().Error())); err != nil {
				panic(err)
			}
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		panic(err)
	}
}
