VERSION ?= v0.0.1
# Image URL to use all building/pushing image targets
IMG_REG ?= registry.cn-hangzhou.aliyuncs.com/r2d2
IMG := $(IMG_REG)/eventexporter

GO_VERSION := 1.17.2
ARCH     ?= $(shell go env GOARCH)
BUILD_DATE = $(shell date +'%Y-%m-%dT%H:%M:%SZ')
COMMIT    = $(shell git rev-parse --short HEAD)
GOENV    := CGO_ENABLED=0 GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) GOPROXY=https://goproxy.io,direct
GO       := $(GOENV) go build -tags=jsoniter

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

build:
	$(GO) -v -o bin/eventexporter cmd/main.go

build-push: build
	docker build -t ${IMG}:${VERSION} -f ./Dockerfile .
	docker push ${IMG}:${VERSION}

# Build the docker image
docker-build:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -v "$$PWD"/mod:/go/pkg/mod -w /go/src/${ROOT} golang:${GO_VERSION} make build

docker-build-push: docker-build
	docker build -t ${IMG}:${VERSION} -f ./Dockerfile .
	docker push ${IMG}:${VERSION}
