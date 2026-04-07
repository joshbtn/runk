# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog.

## [Unreleased]

### Fixed

- Multi-arch image pull resolution now recurses through nested image indexes and falls back to image config platform metadata when descriptor platform metadata is incomplete, improving Docker-style ARM64 selection.

## [0.2.0] - 2026-04-07

### Added

- Image config metadata (Env, Entrypoint, Cmd, WorkingDir) is now extracted during `pull` and stored in `record.json`.
- `runk run` now starts containers using the image's Entrypoint and Cmd with Docker-style composition (`ENTRYPOINT + CMD`).
- `runk run` now inherits the image's environment variables as the base env for the container process.
- `runk run` now honours the image's WorkingDir as the container process `cwd`.
- `--env KEY=VALUE` flag on `runk run` to add or override individual environment variables (repeatable).
- `--entrypoint <cmd>` flag on `runk run` to override the image entrypoint.
- CLI trailing args (after `--`) override the image Cmd; `--entrypoint` overrides the image Entrypoint.
- `BundleInput` struct in `internal/runtime` to carry container spec inputs without a long parameter list.
- `ContainerInput` struct in `internal/runtime` as the single input type for `runtime.Run()`.

## [0.1.1] - 2026-04-06

### Added

- ARM64-focused Docker build and package workflows in `Makefile`:
  - `docker-build-arm64`
  - `docker-package-arm64`
- Improved rootless ID-map resolution behavior when subid ranges exist but helper binaries are not available.

### Changed

- Rootless mapping selection now verifies `newuidmap` and `newgidmap` availability before choosing subuid/subgid mapping mode.
- If subid ranges are present but helper binaries are missing, runk now falls back to single-ID mode (unless strict mode is enabled).

### Fixed

- Avoids unusable subid mapping attempts on constrained hosts without `uidmap` helpers.
- Improves cross-device testing flow by making it easier to produce Linux ARM64 artifacts for SCP-based deployment.

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
