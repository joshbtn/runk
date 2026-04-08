# runk

runk is a daemonless, rootless-oriented OCI container CLI PoC.

## PoC scope

- Standalone Go binary.
- Pull OCI images directly from registries (no dockerd).
- Unpack layers into a local rootfs under `~/.local/share/runk`.
- Generate OCI `config.json` bundle data.
- Execute via `runc` directly from the CLI process (no background daemon).
- Rootless fallback: if subuid/subgid are missing, use `proot` by default (or use `--single-user-fallback` for legacy single-user mapping, unless `--strict-rootless`).

## Build

```bash
go mod tidy
make build
```

`make build` provisions sidecar binaries at `bin/runc` and `bin/proot`, then builds `bin/runk`. Versioned download caches are stored under `.tmp/runc/<version>/linux-<arch>/` and `.tmp/proot/<version>/linux-<arch>/`.

Build configuration knobs:

- `RUNC_VERSION` pinned upstream `runc` release tag (default in `Makefile`)
- `RUNC_ARCH` release asset architecture (default `amd64`)
- `RUNC_SHA256` checksum used for artifact verification (required; default is pinned for `amd64` and `arm64`)
- `PROOT_VERSION` pinned upstream `proot` release tag used for static sidecar download
- `PROOT_SHA256` checksum used for proot artifact verification (required; default is pinned for `amd64` and `arm64`)

Useful build targets:

- `make smoke` validates local sidecar/runtime wiring (`bin/runc` plus `runk --help` with sidecar override)
- `make package` creates `dist/runk-<version>-<os>-<arch>.tar.gz` containing `bin/runk`, `bin/runc`, and `bin/proot`

## Usage

```bash
runk pull alpine:latest
runk run alpine:latest -- /bin/sh -c "id -u && id -g"
```

Global flags:

- `--data-root` data directory (default `~/.local/share/runk`)
- `--runtime` OCI runtime binary path
- `--network` `host|none|slirp4netns` (default `host`)
- `--strict-rootless` disable fallback behavior when subuid/subgid mapping is unavailable
- `--single-user-fallback` use legacy single-user fallback behavior instead of default `proot` fallback

Runtime path resolution order:

1. `--runtime` value
2. `RUNK_RUNTIME` environment variable
3. sidecar `runc` next to `runk` binary (when present)
4. `runc` on PATH

## Notes

- Linux-only PoC.
- Overlay/fuse detection is currently advisory and used for snapshot strategy reporting.
- `--network=slirp4netns` currently validates binary presence; host networking remains default.

## Planning and context docs

- Roadmap and full target feature set: `docs/ROADMAP.md`
- Agent context and repository conventions: `AGENTS.md`

## Release artifacts

GitHub Actions builds release artifacts on tags/releases and uploads `tar.gz` bundles that include `runk` plus sidecar runtime/fallback binaries (`runc` and `proot`).
