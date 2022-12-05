INTEGRATION     := prometheus
BINARY_NAME      = nri-$(INTEGRATION)
SRC_DIR          = .
INTEGRATIONS_DIR = /var/db/newrelic-infra/newrelic-integrations/
CONFIG_DIR       = /etc/newrelic-infra/integrations.d
GO_FILES        := ./
BIN_FILES       := ./cmd/nri-prometheus/
TARGET          := target
GOFLAGS          = -mod=readonly

all: build

build: clean compile test

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: removing binaries..."
	@rm -rfv bin $(TARGET)

compile-deps:
	@echo "=== $(INTEGRATION) === [ compile-deps ]: installing build dependencies..."
	@go get -v -d -t ./...

bin/$(BINARY_NAME):
	@echo "=== $(INTEGRATION) === [ compile ]: building $(BINARY_NAME)..."
	@go build -v -o bin/$(BINARY_NAME) $(BIN_FILES)

compile: compile-deps bin/$(BINARY_NAME)

test:
	@echo "=== $(INTEGRATION) === [ test ]: running unit tests..."
	@go test ./...

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean compile-deps compile test install
