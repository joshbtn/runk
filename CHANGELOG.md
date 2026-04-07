# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog.

## [0.1.0] - 2026-04-06

### Added

- Initial `runk` CLI PoC with `pull` and `run` commands.
- Daemonless runtime execution path using direct `runc` invocation.
- OCI spec and bundle generation flow for container startup.
- Local image content storage and rootfs unpack pipeline.
- Whiteout-aware layer application during unpack.
- Rootless preflight checks for runtime and user namespace readiness.
- Mixed rootless ID mapping behavior:
  - Subuid/subgid range usage when available.
  - Single-UID/GID fallback when unavailable (unless strict mode).
- Devcontainer support using a Dockerfile-based development image.
- Docker-oriented Make targets for dev image build, test, and shell workflows.
- Project documentation for roadmap and rootless behavior.
- Agent project context in `AGENTS.md`.

### Changed

- `pull` command no longer requires `runc` preflight checks.
- `run` flow now scopes rootless/runtime preflight to execution path only.
- Runtime invocation now uses explicit rootless mode and writable runtime state root.
- Interactive shell behavior improved for default `run` experience.
- Default interactive command selection now prefers `bash` when present, with `sh` fallback.
- Added resolver-related bind mounts (`/etc/resolv.conf`, `/etc/hosts`, `/etc/hostname`) when available to improve in-container DNS behavior.
- Apt compatibility handling is distro-aware and applied only when apt is detected in rootfs under single-ID fallback mode.

### Fixed

- Rootless-in-container execution path issues related to cgroup permission handling.
- Prompt rendering issue that showed literal shell escape sequences.
- Debian/apt package update failures in single-ID fallback mode by injecting apt sandbox compatibility config:
  - `APT::Sandbox::User "root";`
  - `Acquire::Sandbox::User "root";`

### Known Limitations

- Linux-first runtime behavior for actual container execution.
- Rootless cgroup delegation remains best-effort in this PoC.
- Host networking remains the default mode; full slirp4netns functional path is not complete.
- Nested runtime usage in some dev environments may require relaxed outer container security settings.
