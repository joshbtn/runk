# runk

runk is a daemonless, rootless-oriented OCI container CLI PoC.

## PoC scope

- Standalone Go binary.
- Pull OCI images directly from registries (no dockerd).
- Unpack layers into a local rootfs under `~/.local/share/runk`.
- Generate OCI `config.json` bundle data.
- Execute via `runc` directly from the CLI process (no background daemon).
- Rootless fallback: if subuid/subgid are missing, map container root to current host user (unless `--strict-rootless`).

## Build

```bash
go mod tidy
go build ./cmd/runk
```

## Usage

```bash
runk pull alpine:latest
runk run alpine:latest -- /bin/sh -c "id -u && id -g"
```

Global flags:

- `--data-root` data directory (default `~/.local/share/runk`)
- `--runtime` OCI runtime binary path (default `runc`)
- `--network` `host|none|slirp4netns` (default `host`)
- `--strict-rootless` disable single-user fallback behavior

## Notes

- Linux-only PoC.
- Overlay/fuse detection is currently advisory and used for snapshot strategy reporting.
- `--network=slirp4netns` currently validates binary presence; host networking remains default.

## Planning and context docs

- Roadmap and full target feature set: `docs/ROADMAP.md`
- Agent context and repository conventions: `AGENTS.md`
