# Copyright 2019 New Relic Corporation. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
NATIVEOS	 := $(shell go version | awk -F '[ /]' '{print $$4}')
NATIVEARCH	 := $(shell go version | awk -F '[ /]' '{print $$5}')
INTEGRATION  := nri-prometheus
BINARY_NAME   = $(INTEGRATION)
IMAGE_NAME   ?= newrelic/nri-prometheus
GOPATH := $(shell go env GOPATH)
GOLANGCI_LINT_VERSION := v1.29.0
GOLANGCI_LINT_BIN = $(GOPATH)/bin/golangci-lint
GORELEASER_VERSION := v0.138.0
GORELEASER_SHA256 := 60cd594e1413483e5728398f861e34834530e0fb1de842312d62ba9ccd57e5f8
GORELEASER_BIN ?= $(GOPATH)/bin/goreleaser
GO_PKGS      := $(shell go list ./... | grep -v "/vendor/")
GOTOOLS       = github.com/stretchr/testify/assert

all: build

build: check-version clean validate test compile docker-release

docker-release:
	@echo "=== $(INTEGRATION) === [ docker-release ]: Building Docker image for release..."
	@cp Dockerfile.release bin/Dockerfile
	@docker build -t $(IMAGE_NAME):release bin/

docker-build:
	@echo "=== $(INTEGRATION) === [ docker-build ]: Building Docker image..."
	@docker build -t $(IMAGE_NAME) .

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: Removing binaries and coverage file..."
	@rm -rfv bin
	@rm -rfv target

tools: check-version tools-golangci-lint
	@echo "=== $(INTEGRATION) === [ tools ]: Installing tools required by the project..."
	@go get $(GOTOOLS)

tools-update: check-version
	@echo "=== $(INTEGRATION) === [ tools-update ]: Updating tools required by the project..."
	@go get -u $(GOTOOLS)

$(GOLANGCI_LINT_BIN):
	@echo "installing GolangCI version $(GOLANGCI_LINT_VERSION)"
	@(curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION))

tools-golangci-lint: $(GOLANGCI_LINT_BIN)

lint: tools-golangci-lint
	@echo "=== $(INTEGRATION) === [ lint ]: Running golangci-lint version $(GOLANGCI_LINT_VERSION)..."
	@$(GOLANGCI_LINT_BIN) run --verbose --timeout 90s

deps: tools deps-only

deps-only:
	@echo "=== $(INTEGRATION) === [ deps ]: Installing package dependencies required by the project..."
	@go mod download

validate: deps
	@echo "=== $(INTEGRATION) === [ validate ]: Validating source code running golangci-lint..."
	@golangci-lint --version
	@golangci-lint run --timeout=30m

compile: deps
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/nri-prometheus/

compile-only: deps-only
	@echo "=== $(INTEGRATION) === [ compile ]: Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) ./cmd/nri-prometheus/

test: deps
	@echo "=== $(INTEGRATION) === [ test ]: Running unit tests..."
	@go test -race $(GO_PKGS)

check-version:
ifdef GOOS
ifneq "$(GOOS)" "$(NATIVEOS)"
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif
ifdef GOARCH
ifneq "$(GOARCH)" "$(NATIVEARCH)"
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif

$(GORELEASER_BIN):
	@echo "=== $(INTEGRATION) === [ release/deps ]: Installing goreleaser"
	@(curl -Ls https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)/goreleaser_Linux_x86_64.tar.gz --output /tmp/goreleaser.tar.gz)
	@(echo "$(GORELEASER_SHA256) /tmp/goreleaser.tar.gz" | sha256sum --check)
	@(tar -xf  /tmp/goreleaser.tar.gz -C $(GOPATH)/bin/)
	@(rm -f /tmp/goreleaser.tar.gz)

release/deps: $(GORELEASER_BIN)

release: release/deps
	@echo "=== $(INTEGRATION) === [ release ]: Releasing new version..."
	@$(GORELEASER_BIN) release
	@(aws s3 sync ./target/deploy/ ${S3_BUCKET})
	@$(MAKE) snyk/monitor

release/test: release/deps
	@echo "=== $(INTEGRATION) === [ release/test ]: Testing releasing new version..."
	@$(GORELEASER_BIN) release --snapshot --skip-publish --rm-dist

snyk: deps-only
	@echo "=== $(INTEGRATION) === [ snyk ]: Running snyk..."
	# @snyk test # issue with Govendor causing snyk to fail
	@snyk test --docker $(IMAGE_NAME):release --file=Dockerfile.release

snyk/monitor: deps-only
	@echo "=== $(INTEGRATION) === [ snyk/monitor ]: Running snyk..."
	# @snyk monitor # issue with Govendor causing snyk to fail
	@snyk monitor --docker $(IMAGE_NAME):release --file=Dockerfile.release

.PHONY: all build clean tools tools-update deps deps-only validate compile compile-only test check-version tools-golangci-lint docker-build release release/deps release/test snyk snyk/monitor docker-release
