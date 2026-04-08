BINARY := runk
DEV_IMAGE := runk-dev:latest
DOCKER_SHELL_FLAGS := --privileged --security-opt seccomp=unconfined
GOOS ?= linux
GOARCH ?= amd64
RUNC_VERSION ?= v1.2.5
RUNC_OS ?= $(GOOS)
RUNC_ARCH ?= $(GOARCH)
RUNC_BASE_URL ?= https://github.com/opencontainers/runc/releases/download
RUNC_ASSET := runc.$(RUNC_ARCH)
RUNC_CACHE_DIR := .tmp/runc/$(RUNC_VERSION)/$(RUNC_OS)-$(RUNC_ARCH)
RUNC_CACHE_PATH := $(RUNC_CACHE_DIR)/$(RUNC_ASSET)
RUNC_PATH := bin/runc
RUNC_SHA256_AMD64 := fbd851fce6a8e0d67a9d184ea544c2abf67c9fd29b80fcc1adf67dfe9eb036a1
RUNC_SHA256_ARM64 := bfc6575f4c601740539553b639ad6f635c23f76695ed484171bd864df6a23f76
RUNC_SHA256_DEFAULT := $(if $(filter amd64,$(RUNC_ARCH)),$(RUNC_SHA256_AMD64),$(if $(filter arm64,$(RUNC_ARCH)),$(RUNC_SHA256_ARM64),))
RUNC_SHA256 ?= $(RUNC_SHA256_DEFAULT)
PROOT_VERSION ?= v5.3.0
PROOT_OS ?= $(GOOS)
PROOT_ARCH_RAW ?= $(GOARCH)
PROOT_ARCH := $(if $(filter amd64,$(PROOT_ARCH_RAW)),x86_64,$(if $(filter arm64,$(PROOT_ARCH_RAW)),aarch64,$(PROOT_ARCH_RAW)))
PROOT_BASE_URL ?= https://github.com/proot-me/proot/releases/download
PROOT_ASSET := proot-$(PROOT_VERSION)-$(PROOT_ARCH)-static
PROOT_CACHE_DIR := .tmp/proot/$(PROOT_VERSION)/$(PROOT_OS)-$(PROOT_ARCH)
PROOT_CACHE_PATH := $(PROOT_CACHE_DIR)/$(PROOT_ASSET)
PROOT_PATH := bin/proot
PROOT_SHA256_AMD64 := d1eb20cb201e6df08d707023efb000623ff7c10d6574839d7bb42d0adba6b4da
PROOT_SHA256_ARM64 := fa10b1a7818c2f5b1dcb5834450570c368c9ecf66d31521509621b95c4538a45
PROOT_SHA256_DEFAULT := $(if $(filter x86_64,$(PROOT_ARCH)),$(PROOT_SHA256_AMD64),$(if $(filter aarch64,$(PROOT_ARCH)),$(PROOT_SHA256_ARM64),))
PROOT_SHA256 ?= $(PROOT_SHA256_DEFAULT)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST_DIR := dist
PKG_BASENAME := runk-$(VERSION)-$(GOOS)-$(GOARCH)
PKG_DIR := $(DIST_DIR)/$(PKG_BASENAME)
PKG_TAR := $(DIST_DIR)/$(PKG_BASENAME).tar.gz

.PHONY: build tidy test smoke package docker-build docker-test docker-shell docker-build-arm64 docker-package-arm64 runc-install runc-download runc-verify runc-clean proot-install proot-download proot-verify proot-clean

build: runc-install proot-install
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(BINARY) ./cmd/runk

runc-install: runc-download runc-verify
	mkdir -p bin
	rm -rf $(RUNC_PATH)
	install -m 0755 $(RUNC_CACHE_PATH) $(RUNC_PATH)

runc-download:
	mkdir -p $(RUNC_CACHE_DIR)
	curl -fsSL $(RUNC_BASE_URL)/$(RUNC_VERSION)/$(RUNC_ASSET) -o $(RUNC_CACHE_PATH)

runc-verify:
	@if [ -z "$(RUNC_SHA256)" ]; then \
		echo "No checksum configured for $(RUNC_ASSET). Set RUNC_SHA256 explicitly."; \
		exit 1; \
	fi
	echo "$(RUNC_SHA256)  $(RUNC_CACHE_PATH)" | sha256sum -c -

runc-clean:
	rm -rf .tmp/runc bin/runc

proot-install: proot-download proot-verify
	mkdir -p bin
	rm -rf $(PROOT_PATH)
	install -m 0755 $(PROOT_CACHE_PATH) $(PROOT_PATH)

proot-download:
	mkdir -p $(PROOT_CACHE_DIR)
	curl -fsSL $(PROOT_BASE_URL)/$(PROOT_VERSION)/$(PROOT_ASSET) -o $(PROOT_CACHE_PATH)

proot-verify:
	@if [ -z "$(PROOT_SHA256)" ]; then \
		echo "No checksum configured for $(PROOT_ASSET). Set PROOT_SHA256 explicitly."; \
		exit 1; \
	fi
	echo "$(PROOT_SHA256)  $(PROOT_CACHE_PATH)" | sha256sum -c -

proot-clean:
	rm -rf .tmp/proot bin/proot

tidy:
	go mod tidy

test:
	go test ./...

smoke: build
	./bin/runc --version >/dev/null
	./bin/proot --version >/dev/null
	RUNK_RUNTIME=./bin/runc ./bin/runk --help > .tmp/runk-help.txt 2>&1 || true
	grep -q "Usage of runk:" .tmp/runk-help.txt

package: build
	rm -rf $(PKG_DIR) $(PKG_TAR)
	mkdir -p $(PKG_DIR)/bin $(DIST_DIR)
	cp bin/$(BINARY) $(PKG_DIR)/bin/runk
	cp $(RUNC_PATH) $(PKG_DIR)/bin/runc
	cp $(PROOT_PATH) $(PKG_DIR)/bin/proot
	tar -C $(DIST_DIR) -czf $(PKG_TAR) $(PKG_BASENAME)

docker-build:
	docker build -f docker/dev.Dockerfile -t $(DEV_IMAGE) .

docker-test: docker-build
	docker run --rm -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) make test

docker-shell: docker-build
	docker run --rm -it $(DOCKER_SHELL_FLAGS) -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) bash -c "export PATH=/workspace/bin:/usr/local/go/bin:$$PATH; make build; exec bash -i"

docker-build-arm64: docker-build
	docker run --rm -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) bash -c "export PATH=/usr/local/go/bin:$$PATH; make GOOS=linux GOARCH=arm64 build"

docker-package-arm64: docker-build
	docker run --rm -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) bash -c "export PATH=/usr/local/go/bin:$$PATH; make GOOS=linux GOARCH=arm64 package"
