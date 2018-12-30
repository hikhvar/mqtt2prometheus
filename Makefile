ifndef GOPATH
  GOPATH:=$(shell go env GOPATH)
endif

ifndef GOBIN
  GOBIN:=$(GOPATH)/bin
endif

ifndef GOARCH
  GOARCH:=$(shell go env GOARCH)
endif

ifndef GOOS
  GOOS:=$(shell go env GOOS)
endif

ifndef GOARM
  GOARM:=$(shell go env GOARM)
endif

ifndef TARGET_FILE
  TARGET_FILE:=bin/mqtt2prometheus.$(GOOS)_$(GOARCH)$(GOARM)
endif

all: build

install-dep:
	@which $(GOBIN)/dep || go get -u github.com/golang/dep/cmd/dep

Gopkg.lock: | install-dep
	$(GOBIN)/dep ensure --no-vendor

Gopkg.toml: | install-dep
	$(GOBIN)/dep init

prepare-vendor: Gopkg.toml Gopkg.lock
	$(GOBIN)/dep ensure -update --no-vendor
	$(GOBIN)/dep status
	@echo "You can apply these locks via 'make vendor' or rollback via 'git checkout -- Gopkg.lock'"

vendor: install-dep Gopkg.toml Gopkg.lock
	$(GOBIN)/dep ensure -vendor-only
	$(GOBIN)/dep status

test:
	go test ./...

build: vendor
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(TARGET_FILE) ./cmd

static_build: vendor
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(TARGET_FILE) -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd

container:
	docker build -t mqtt2prometheus:latest .
