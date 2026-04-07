# AGENTS Context for runk

This file captures durable context for AI coding agents working in this repository.

## Project summary

runk is a Go-based daemonless, rootless-oriented OCI container CLI PoC.

Current CLI surface:

- pull
- run

## Goals

- Pull and execute OCI images without dockerd.
- Use runc directly from CLI process.
- Operate unprivileged and support fallback when subuid/subgid are missing.

## Current architecture

- cmd/runk: command entrypoint and flag parsing.
- internal/config: global options and defaults.
- internal/rootless: preflight checks and ID mapping logic.
- internal/image: pull, store, unpack pipeline.
- internal/snapshot: snapshot driver selection.
- internal/oci: OCI runtime spec generation.
- internal/runtime: bundle creation and runc execution.

## Important defaults

- Data root: ~/.local/share/runk
- Runtime: runc
- Network mode: host
- Rootless mode: mixed fallback unless strict-rootless enabled

## Environment assumptions

- Runtime path is Linux-first for actual container execution.
- Development may happen on Windows via devcontainer/docker workflow.

## Non-goals for current PoC

- Full Docker-compatible UX.
- Orchestration or long-running daemon.
- Full cgroup delegation management.

## Priority backlog

1. Add rootlesskit execution path.
2. Add slirp4netns functional mode.
3. Improve unpack/compression support.
4. Expand lifecycle commands beyond run.
5. Add integration test harness.

## Agent guardrails

- Prefer small, focused changes.
- Keep docs aligned with implemented behavior.
- Do not claim implemented features that are only planned.
- Preserve host-default networking unless explicitly changed.
