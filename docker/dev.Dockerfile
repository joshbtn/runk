FROM golang:1.24-bookworm

# Small but useful toolchain for Go development.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        git \
        make \
        runc \
        bash \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# Keep module and build caches in standard Go locations.
ENV GOPATH=/go
ENV GOCACHE=/root/.cache/go-build
ENV PATH=/usr/local/go/bin:/go/bin:${PATH}
