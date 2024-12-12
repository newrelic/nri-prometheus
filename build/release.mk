BUILD_DIR    := ./bin/
GORELEASER_VERSION ?= v2.4.4
GORELEASER_BIN ?= bin/goreleaser

bin:
	@mkdir -p $(BUILD_DIR)

$(GORELEASER_BIN): bin
	@echo "===> $(INTEGRATION) === [$(GORELEASER_BIN)] Installing goreleaser $(GORELEASER_VERSION)"
	@(wget -qO /tmp/goreleaser.tar.gz https://github.com/goreleaser/goreleaser/releases/download/$(GORELEASER_VERSION)/goreleaser_$(OS_DOWNLOAD)_x86_64.tar.gz)
	@(tar -xf  /tmp/goreleaser.tar.gz -C bin/)
	@(rm -f /tmp/goreleaser.tar.gz)
	@echo "===> $(INTEGRATION) === [$(GORELEASER_BIN)] goreleaser downloaded"

.PHONY : release/clean
release/clean:
	@echo "===> $(INTEGRATION) === [release/clean] remove build metadata files"
	rm -fv $(CURDIR)/cmd/nri-prometheus/versioninfo.json
	rm -fv $(CURDIR)/cmd/nri-prometheus/resource.syso

.PHONY : release/deps
release/deps: $(GORELEASER_BIN)
	@echo "===> $(INTEGRATION) === [release/deps] installing deps"
	@go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
	@go mod tidy

.PHONY : release/build
release/build: release/deps release/clean
ifeq ($(GENERATE_PACKAGES), true)
	@echo "===> $(INTEGRATION) === [release/build] PRERELEASE/RELEASE compiling all binaries, creating packages, archives"
	# Pre-release/release actually builds and uploads images
	# goreleaser will compile binaries, generate manifests, and push multi-arch docker images
	# TAG_SUFFIX should be set as "-pre" during prereleases
	@$(GORELEASER_BIN) release --config $(CURDIR)/.goreleaser.yml --skip=validate --clean
else
	@echo "===> $(INTEGRATION) === [release/build] build compiling all binaries"
	# release/build with PRERELEASE unset is actually called only from push/pr pipeline to check everything builds correctly
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser.yml --skip=validate --snapshot --clean
endif

.PHONY : release/build-fips
release/build-fips: release/deps release/clean
ifeq ($(GENERATE_PACKAGES), true)
	@echo "===> $(INTEGRATION) === [release/build] PRERELEASE/RELEASE compiling fips binaries, creating packages, archives"
	# TAG_SUFFIX should be set as "-pre" during prereleases
	@$(GORELEASER_BIN) release --config $(CURDIR)/.goreleaser-fips.yml --skip=validate --clean
else
	@echo "===> $(INTEGRATION) === [release/build-fips] build compiling fips binaries"
	# release/build with PRERELEASE unset is actually called only from push/pr pipeline to check everything builds correctly
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser-fips.yml --skip=validate --snapshot --clean
endif

.PHONY : release/fix-archive
release/fix-archive:
	@echo "===> $(INTEGRATION) === [release/fix-archive] fixing tar.gz archives internal structure"
	@bash $(CURDIR)/build/nix/fix_archives.sh $(CURDIR)
	@echo "===> $(INTEGRATION) === [release/fix-archive] fixing zip archives internal structure"
	@bash $(CURDIR)/build/windows/fix_archives.sh $(CURDIR)

.PHONY : release/publish
release/publish:
ifeq ($(PRERELEASE), true)
	@echo "===> $(INTEGRATION) === [release/publish] publishing packages"
	@bash $(CURDIR)/build/upload_artifacts_gh.sh
endif
	# TODO: This seems like a leftover, should consider removing
	@echo "===> $(INTEGRATION) === [release/publish] compiling binaries"
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser.yml --skip=validate --snapshot --clean

.PHONY : release/publish-fips
release/publish-fips:
ifeq ($(PRERELEASE), true)
	@echo "===> $(INTEGRATION) === [release/publish-fips] publishing fips packages"
	@bash $(CURDIR)/build/upload_artifacts_gh.sh
endif
	# TODO: This seems like a leftover, should consider removing
	@echo "===> $(INTEGRATION) === [release/publish-fips] compiling fips binaries"
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser-fips.yml --skip=validate --snapshot --clean

.PHONY : release
release: release/build release/fix-archive release/publish release/clean
	@echo "===> $(INTEGRATION) === [release] full pre-release cycle complete for nix"

.PHONY : release-fips
release-fips: release/build-fips release/fix-archive release/publish-fips release/clean
	@echo "===> $(INTEGRATION) === [release-fips] fips pre-release cycle complete for nix"

OS := $(shell uname -s)
ifeq ($(OS), Darwin)
	OS_DOWNLOAD := "darwin"
	TAR := gtar
else
	OS_DOWNLOAD := "linux"
endif
