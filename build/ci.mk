.PHONY : ci/pull-builder-image
ci/pull-builder-image:
	@docker pull $(BUILDER_IMAGE)

.PHONY : ci/deps
ci/deps: ci/pull-builder-image

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
			$(BUILDER_IMAGE) bash

.PHONY : ci/validate
ci/validate: ci/deps
	@docker run --rm -t \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			$(BUILDER_IMAGE) make validate

.PHONY : ci/test
ci/test: ci/deps
	@docker run --rm -t \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			$(BUILDER_IMAGE) make test

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
			$(BUILDER_IMAGE) make release/build
else
	@echo "===> $(INTEGRATION) ===  [ci/build] TAG env variable expected to be set"
	exit 1
endif

.PHONY : ci/prerelease-fips
ci/prerelease-fips: ci/deps
ifdef TAG
	@docker run --rm -t \
			--name "nri-$(INTEGRATION)-prerelease" \
			-v $(CURDIR):/go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-w /go/src/github.com/newrelic/nri-$(INTEGRATION) \
			-e INTEGRATION \
			-e PRERELEASE=true \
			-e GITHUB_TOKEN \
			-e REPO_FULL_NAME \
			-e TAG \
			-e TAG_SUFFIX \
			-e GENERATE_PACKAGES \
			-e PRERELEASE \
			$(BUILDER_IMAGE) make release-fips
else
	@echo "===> $(INTEGRATION) ===  [ci/prerelease] TAG env variable expected to be set"
	exit 1
endif
