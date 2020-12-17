BUILD_DIR    := ./bin/
GORELEASER_VERSION := v0.146.0
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
ifeq ($(PRERELEASE), true)
	@echo "===> $(INTEGRATION) === [release/build] PRE-RELEASE compiling all binaries, creating packages, archives"
	# Pre-release pipeline in GHA runs `goreleaser release`, which will upload docker images, s3 manifests, and publish a release with changelog and artifacts
	@$(GORELEASER_BIN) release --config $(CURDIR)/.goreleaser.yml --skip-validate --rm-dist
	# Upload manifests to S3
	aws s3 cp $(CURDIR)/target/deploy/* $$S3_PATH/integrations/kubernetes/
else
	@echo "===> $(INTEGRATION) === [release/build] build compiling all binaries"
	# Just build packages. This is done as a double-check, and also called from push/pr pipeline as a check
	@$(GORELEASER_BIN) build --config $(CURDIR)/.goreleaser.yml --skip-validate --snapshot --rm-dist
endif

.PHONY : release/fix-archive
release/fix-archive:
	@echo "===> $(INTEGRATION) === [release/fix-archive] fixing tar.gz archives internal structure"
	@bash $(CURDIR)/build/nix/fix_archives.sh $(CURDIR)
	@echo "===> $(INTEGRATION) === [release/fix-archive] fixing zip archives internal structure"
	@bash $(CURDIR)/build/windows/fix_archives.sh $(CURDIR)

.PHONY : release/sign/nix
release/sign/nix:
	@echo "===> $(INTEGRATION) === [release/sign] signing packages"
	@bash $(CURDIR)/build/nix/sign.sh


.PHONY : release/publish
release/publish:
	@echo "===> $(INTEGRATION) === [release/publish] publishing artifacts"
	# REPO_FULL_NAME here is only necessary for forks. It can be removed when this is merged into the original repo
	@bash $(CURDIR)/build/upload_artifacts_gh.sh $(REPO_FULL_NAME)

.PHONY : release
release: release/build release/fix-archive release/publish release/clean
	# release/sign/nix 
	@echo "===> $(INTEGRATION) === [release/publish] full pre-release cycle complete for nix"

OS := $(shell uname -s)
ifeq ($(OS), Darwin)
	OS_DOWNLOAD := "darwin"
	TAR := gtar
else
	OS_DOWNLOAD := "linux"
endif
