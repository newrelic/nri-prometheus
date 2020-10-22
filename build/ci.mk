BUILDER_TAG ?= nri-$(INTEGRATION)-builder

.PHONY : ci/deps
ci/deps:
	@docker build -t $(BUILDER_TAG) -f $(CURDIR)/build/Dockerfile $(CURDIR)

.PHONY : ci/debug-container
ci/debug-container: ci/deps
	@docker run --rm -it \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-e PRERELEASE=true \
			-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
			-e TAG \
			-e GPG_MAIL \
			-e GPG_PASSPHRASE \
			-e GPG_PRIVATE_KEY_BASE64 \
			$(BUILDER_TAG) bash

.PHONY : ci/validate
ci/validate: ci/deps
	@docker run --rm -t \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			$(BUILDER_TAG) make validate

.PHONY : ci/test
ci/test: ci/deps
	@docker run --rm -t \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			$(BUILDER_TAG) make test

.PHONY : ci/snyk-test
ci/snyk-test:
	@docker run --rm -t \
			--name "nri-$(INTEGRATION)-snyk-test" \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-e SNYK_TOKEN \
			snyk/snyk:golang snyk test --severity-threshold=high
			
.PHONY : ci/build
ci/build: ci/deps
ifdef TAG
	@docker run --rm -t \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-e INTEGRATION=$(INTEGRATION) \
			-e TAG \
			$(BUILDER_TAG) make release/build
else
	@echo "===> $(INTEGRATION) ===  [ci/build] TAG env variable expected to be set"
	exit 1
endif

.PHONY : ci/prerelease
ci/prerelease: ci/deps
ifdef TAG
	# REPO_FULL_NAME here is only necessary for forks. It can be removed when this is merged into the original repo
	@docker run --rm -t \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-e INTEGRATION \
			-e PRERELEASE=true \
			-e REPO_FULL_NAME \
			-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
			-e TAG \
			-e GPG_MAIL \
			-e GPG_PASSPHRASE \
			-e GPG_PRIVATE_KEY_BASE64 \
			$(BUILDER_TAG) make release
else
	@echo "===> $(INTEGRATION) ===  [ci/prerelease] TAG env variable expected to be set"
	exit 1
endif
