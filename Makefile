
ifndef GOARCH
  GOARCH:=$(shell go env GOARCH)
endif

ifndef GOOS
  GOOS:=$(shell go env GOOS)
endif

ifndef TARGET_FILE
  TARGET_FILE:=bin/mqtt2prometheus.$(GOOS)_$(GOARCH)
endif

install-dep:
	@which dep || go get -u github.com/golang/dep/cmd/dep

Gopkg.lock: | install-dep
	dep ensure --no-vendor

Gopkg.toml: | install-dep
	dep init

prepare-vendor: Gopkg.toml Gopkg.lock
	dep ensure -update --no-vendor
	dep status
	@echo "You can apply these locks via 'make vendor' or rollback via 'git checkout -- Gopkg.lock'"

vendor: Gopkg.toml Gopkg.lock
	dep ensure -vendor-only
	dep status

test:
	go test ./...

build: vendor
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(TARGET_FILE) ./cmd

static_build: vendor
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(TARGET_FILE) -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd

container:
	docker build -t mqtt2prometheus:latest .
