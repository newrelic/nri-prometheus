BUILD_DIR    := ./bin/
GORELEASER_VERSION ?= v0.168.0
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
	@$(GORELEASER_BIN) release --config $(CURDIR)/.goreleaser.yml --skip-validate --rm-dist
else
	@echo "===> $(INTEGRATION) === [release/build] build compiling all binaries"
	# release/build with PRERELEASE unset is actually called only from push/pr pipeline to check everything builds correctly
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser.yml --skip-validate --snapshot --rm-dist
endif

.PHONY : release/fix-archive
release/fix-archive:
	@echo "===> $(INTEGRATION) === [release/fix-archive] fixing tar.gz archives internal structure"
	@bash $(CURDIR)/build/nix/fix_archives.sh $(CURDIR)
	@echo "===> $(INTEGRATION) === [release/fix-archive] fixing zip archives internal structure"
	@bash $(CURDIR)/build/windows/fix_archives.sh $(CURDIR)

.PHONY : release/publish
release/publish:
ifeq ($(UPLOAD_PACKAGES), true)
	@echo "===> $(INTEGRATION) === [release/publish] publishing packages"
	# REPO_FULL_NAME here is only necessary for forks. It can be removed when this is merged into the original repo
	@bash $(CURDIR)/build/upload_artifacts_gh.sh $(REPO_FULL_NAME)
endif
	@echo "===> $(INTEGRATION) === [release/publish] publishing manifests"
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser.yml --skip-validate --snapshot --rm-dist


.PHONY : release
release: release/build release/fix-archive release/publish release/clean
	@echo "===> $(INTEGRATION) === [release/publish] full pre-release cycle complete for nix"

OS := $(shell uname -s)
ifeq ($(OS), Darwin)
	OS_DOWNLOAD := "darwin"
	TAR := gtar
else
	OS_DOWNLOAD := "linux"
endif
