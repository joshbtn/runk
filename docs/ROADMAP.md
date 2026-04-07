# runk Roadmap

This document tracks phased delivery for runk, a daemonless and rootless OCI container CLI.

## Vision

- Single standalone Go binary.
- No background daemon.
- Pull, unpack, and run OCI images locally.
- Rootless-first operation with fallback when subuid/subgid mappings are unavailable.

## Current status

Implemented baseline:

- CLI commands: pull, run.
- Rootless preflight checks and mixed ID-map fallback logic.
- Automatic apt compatibility in single-ID fallback mode.
- Registry pull with local blob store and rootfs unpack.
- OCI bundle generation and direct runc invocation.
- Dev container workflow and Docker-backed Make targets.

## Phase 1: Foundation (done)

- Initialize module and package layout.
- Add config and data-root handling.
- Add initial Makefile and docs.

## Phase 2: Rootless core (in progress)

- Keep mixed mode behavior:
  - Use subuid/subgid ranges if available.
  - Fallback to single UID/GID mapping by default.
  - Fail in strict mode.
- Improve preflight output and diagnostics.
- Add explicit rootlesskit integration path.

## Phase 3: Image and storage pipeline (in progress)

- Keep digest-addressed blob storage under local data root.
- Improve unpack correctness:
  - Better metadata handling and whiteout compatibility.
  - Compression format detection beyond gzip.
- Add deterministic snapshot strategy behavior:
  - overlayfs rootless where supported.
  - fuse-overlayfs fallback.
  - copy/bind fallback.

## Phase 4: Runtime lifecycle

- Expand lifecycle commands:
  - create
  - start
  - delete
  - optional exec
- Harden bundle cleanup and failure recovery.
- Add richer OCI spec controls for env, mounts, user, and annotations.

## Phase 5: Networking and isolation

- Keep host networking as default for PoC compatibility.
- Add functional slirp4netns mode.
- Add network mode validation and clear compatibility matrix.

## Phase 6: Tests and reliability

- Add integration smoke tests:
  - pull alpine
  - run id checks
  - strict-rootless failure behavior
  - fallback path selection
- Add regression tests for unpack and ID map resolution.

## Phase 7: UX and distribution

- Improve CLI help and structured errors.
- Add release build workflow.
- Publish architecture and limitations docs.

## Target feature set (full)

Core runtime:

- Daemonless process model.
- Rootless operation by default.
- OCI-compliant config generation and runc execution.

Image pipeline:

- Pull from Docker Hub and GHCR.
- Local OCI-style content storage.
- Reusable unpacked rootfs/snapshots.

Rootless:

- No hard dependency on host subuid/subgid configuration.
- Mixed fallback mode plus strict mode toggle.
- Optional rootless networking via slirp4netns.

Developer experience:

- Native and containerized development workflows.
- Deterministic Make targets for build/test/container build.
- Living docs for roadmap and agent context.
