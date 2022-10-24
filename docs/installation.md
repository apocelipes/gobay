# Installation

```sh
go get github.com/apocelipes/gobay/cmd/gobay
```

## Tools for gobay project

Here are the tools we recommend installing for developing a project with gobay:
`And if it's too much for you, checkout the docker development box section below`

- [golang (programming language)](https://golang.org/doc/install) [Download and Install](https://golang.org/dl/)

- [grpc tool](https://github.com/grpc/grpc-go)

```sh
go get -u google.golang.org/grpc
```

- [protobuf tool](https://github.com/golang/protobuf) [(probably official)installation guide](http://google.github.io/proto-lens/installing-protoc.html)

```sh
# install protobuf (or at least protobuf-compiler)
# mac:
brew install protobuf

go get -u github.com/gogo/protobuf/protoc-gen-gofast
go get -u github.com/golang/mock/mockgen
```

- [lint tool: golangci-lint](https://golangci-lint.run/usage/install/#local-installation)

```sh
# mac
brew install golangci/tap/golangci-lint
brew upgrade golangci/tap/golangci-lint

# windows
# install git bash and：
# binary will be $(go env GOPATH)/bin/golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.27.0
golangci-lint --version
```

- [orm tool: ent, make sure to install version v0.4.0](https://github.com/shanbay/ent)

```sh
go get github.com/apocelipes/ent/cmd/entc@v0.4.0
```

- [test tool: gotests](https://github.com/cweill/gotests)

```sh
go get -u github.com/cweill/gotests/
```

- [test coverage tool: gocovmerge](https://github.com/wadey/gocovmerge)

```sh
go get github.com/wadey/gocovmerge
```

- [OpenAPI generator for echo (oapi-codegen)](https://github.com/shanbay/oapi-codegen/releases)

```sh
# Or download binary from Github release
git clone git@github.com:shanbay/oapi-codegen.git
cd oapi-codegen
go install ./cmd/oapi-codegen/
```

## Docker development box

If any of the installations above is too annoying in your OS, you may try this option:

After creating your new project (checkout the quickstart.md page), there will be a `dockerfiles` directory in your project.

open it, and run `sh run.sh`, it will build a development ready docker image, and run a docker container with the project directory mounted.
