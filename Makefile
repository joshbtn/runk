BINARY := runk
DEV_IMAGE := runk-dev:latest
DOCKER_SHELL_FLAGS := --privileged --security-opt seccomp=unconfined

.PHONY: build tidy test docker-build docker-test docker-shell

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
	docker run --rm -it $(DOCKER_SHELL_FLAGS) -v "$(CURDIR):/workspace" -w /workspace $(DEV_IMAGE) bash -c "export PATH=/workspace/bin:/usr/local/go/bin:$$PATH; make build; exec bash -i"
