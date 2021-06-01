INTEGRATION     := prometheus
BINARY_NAME      = nri-$(INTEGRATION)
SRC_DIR          = .
TEST_DEPS        = github.com/axw/gocov/gocov github.com/AlekSi/gocov-xml
INTEGRATIONS_DIR = /var/db/newrelic-infra/newrelic-integrations/
CONFIG_DIR       = /etc/newrelic-infra/integrations.d
GO_FILES        := ./
BIN_FILES       := ./cmd/nri-prometheus/
TARGET          := target
GOFLAGS          = -mod=readonly
GOLANGCI_LINT    = github.com/golangci/golangci-lint/cmd/golangci-lint

all: build

build: clean validate compile test

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: removing binaries and coverage file..."
	@rm -rfv bin coverage.xml $(TARGET)


$(GOLANGCI_LINT_BIN):
	@echo "installing GolangCI version $(GOLANGCI_LINT_VERSION)"
	@(curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION))

validate:
	@printf "=== $(INTEGRATION) === [ validate ]: running golangci-lint & semgrep... "
	@go run  $(GOFLAGS) $(GOLANGCI_LINT) run --verbose
	@[ -f .semgrep.yml ] && semgrep_config=".semgrep.yml" || semgrep_config="p/golang" ; \
	docker run --rm -v "${PWD}:/src:ro" --workdir /src returntocorp/semgrep -c "$$semgrep_config"

compile-deps:
	@echo "=== $(INTEGRATION) === [ compile-deps ]: installing build dependencies..."
	@go get -v -d -t ./...

bin/$(BINARY_NAME):
	@echo "=== $(INTEGRATION) === [ compile ]: building $(BINARY_NAME)..."
	@go build -v -o bin/$(BINARY_NAME) $(BIN_FILES)

compile: compile-deps bin/$(BINARY_NAME)

clean-test:
	@echo "=== $(INTEGRATION) === [ compile ]: cleanup test dependencies..."
	@go mod tidy

test-deps: compile-deps
	@echo "=== $(INTEGRATION) === [ test-deps ]: installing testing dependencies..."
	@go get -v $(TEST_DEPS)

test-only:
	@echo "=== $(INTEGRATION) === [ test ]: running unit tests..."
	@gocov test ./... | gocov-xml > coverage.xml

test: test-deps test-only clean-test


integration-test: test-deps
	@echo "=== $(INTEGRATION) === [ test ]: running integration tests...[TODO]"
	#@docker-compose -f tests/integration/docker-compose.yml up -d --build
	#@go test -tags=integration ./tests/integration/. || (ret=$$?; docker-compose -f tests/integration/docker-compose.yml down && exit $$ret)
	#@docker-compose -f tests/integration/docker-compose.yml down

install: bin/$(BINARY_NAME)
	@echo "=== $(INTEGRATION) === [ install ]: installing bin/$(BINARY_NAME)..."
	@sudo install -D --mode=755 --owner=root --strip $(ROOT)bin/$(BINARY_NAME) $(INTEGRATIONS_DIR)/bin/$(BINARY_NAME)
	@sudo install -D --mode=644 --owner=root $(ROOT)$(INTEGRATION)-definition.yml $(INTEGRATIONS_DIR)/$(INTEGRATION)-definition.yml
	@sudo install -D --mode=644 --owner=root $(ROOT)$(INTEGRATION)-config.yml.sample $(CONFIG_DIR)/$(INTEGRATION)-config.yml.sample

# Include thematic Makefiles
include $(CURDIR)/build/ci.mk
include $(CURDIR)/build/release.mk

.PHONY: all build clean validate compile-deps compile test-deps test-only test integration-test install
