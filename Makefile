ifndef GOBINARY
  GOBINARY:="go"
endif

ifndef GOPATH
  GOPATH:=$(shell $(GOBINARY) env GOPATH)
endif

ifndef GOBIN
  GOBIN:=$(GOPATH)/bin
endif

ifndef GOARCH
  GOARCH:=$(shell $(GOBINARY) env GOARCH)
endif

ifndef GOOS
  GOOS:=$(shell $(GOBINARY) env GOOS)
endif

ifndef GOARM
  GOARM:=$(shell $(GOBINARY) env GOARM)
endif

ifndef TARGET_FILE
  TARGET_FILE:=bin/mqtt2prometheus.$(GOOS)_$(GOARCH)$(GOARM)
endif

all: build

GO111MODULE=on


lint:
	golangci-lint run

test:
	$(GOBINARY) test ./...
	$(GOBINARY) vet ./...

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBINARY) build -o $(TARGET_FILE) ./cmd

static_build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBINARY) build -o $(TARGET_FILE) -a -tags netgo -ldflags '-w -extldflags "-static"' ./cmd

container:
	docker build -t mqtt2prometheus:latest .

test_release:
	goreleaser --rm-dist --skip-validate --skip-publish
