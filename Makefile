BINARY := runk
DEV_IMAGE := runk-dev:latest
DOCKER_SHELL_FLAGS := --privileged --security-opt seccomp=unconfined

.PHONY: build tidy test docker-build docker-test docker-shell docker-run

build:
	go build -o bin/$(BINARY) ./cmd/runk

tidy:
	go mod tidy

test:
	go test ./...

docker-build:
	docker build -f docker/dev.Dockerfile -t $(DEV_IMAGE) .

docker-test: docker-build
	docker run --rm -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) make test

docker-shell: docker-build
	docker run --rm -it $(DOCKER_SHELL_FLAGS) -e RUNK_PATH_PREFIX="/workspace/bin:/usr/local/go/bin" -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) bash -lc "export PATH=$$RUNK_PATH_PREFIX:$$PATH; make build; export PS1='(runk-dev) \u@\h:\w\\$ '; exec bash -il"

docker-run: docker-build
	docker run --rm $(DOCKER_SHELL_FLAGS) -e PATH="/workspace/bin:/usr/local/go/bin:$$PATH" -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) bash -lc "make build && runk --help"
