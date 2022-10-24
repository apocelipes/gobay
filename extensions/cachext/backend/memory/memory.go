package memory

import (
	"context"
	"time"

	"github.com/spf13/viper"

	"github.com/apocelipes/gobay/extensions/cachext"
)

func init() {
	if err := cachext.RegisteBackend("memory", func() cachext.CacheBackend { return &memoryBackend{} }); err != nil {
		panic("MemoryBackend Init error")
	}
}

type memoryBackendNode struct {
	Value     []byte
	ExpiredAt time.Time
}

type memoryBackend struct {
	client map[string]*memoryBackendNode
}

func (m *memoryBackend) Init(*viper.Viper) error {
	m.client = make(map[string]*memoryBackendNode)
	return nil
}

func (m *memoryBackend) CheckHealth(ctx context.Context) error {
	return nil
}

func (m *memoryBackend) Get(ctx context.Context, key string) ([]byte, error) {
	res, exists := m.client[key]
	if !exists {
		return nil, nil
	}
	if res.ExpiredAt.Before(time.Now()) {
		m.Delete(ctx, key)
		return nil, nil
	} else {
		return res.Value, nil
	}
}

func (m *memoryBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	node := &memoryBackendNode{Value: value, ExpiredAt: time.Now().Add(ttl)}
	m.client[key] = node
	return nil
}

func (m *memoryBackend) SetMany(ctx context.Context, keyValues map[string][]byte, ttl time.Duration) error {
	for key, value := range keyValues {
		if err := m.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

func (m *memoryBackend) GetMany(ctx context.Context, keys []string) [][]byte {
	resBytes := make([][]byte, len(keys))
	for i, key := range keys {
		resBytes[i], _ = m.Get(ctx, key)
	}
	return resBytes
}

func (m *memoryBackend) Delete(ctx context.Context, key string) bool {
	exists := m.Exists(ctx, key)
	delete(m.client, key)
	return exists
}

func (m *memoryBackend) DeleteMany(ctx context.Context, keys []string) bool {
	var res bool
	for _, key := range keys {
		if m.Delete(ctx, key) {
			res = true
		}
	}
	return res
}

func (m *memoryBackend) Expire(ctx context.Context, key string, ttl time.Duration) bool {
	val, _ := m.Get(ctx, key)
	if val == nil {
		return false
	}
	m.client[key].ExpiredAt = time.Now().Add(ttl)
	return true
}

func (m *memoryBackend) TTL(ctx context.Context, key string) time.Duration {
	_, _ = m.Get(ctx, key)
	val := m.client[key]
	if val == nil {
		return 0
	}
	return time.Until(val.ExpiredAt)
}

func (m *memoryBackend) Exists(ctx context.Context, key string) bool {
	val, _ := m.Get(ctx, key)
	if val == nil {
		return false
	} else {
		return true
	}
}

func (m *memoryBackend) Close() error {
	return nil
}
