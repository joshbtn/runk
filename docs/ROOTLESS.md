# Rootless behavior in runk PoC

## ID mapping

runk resolves ID mapping in this order:

1. Use `/etc/subuid` and `/etc/subgid` ranges for the current user when present, if `newuidmap` and `newgidmap` are available.
2. If unavailable and strict mode is off:
   - use `proot` fallback by default
   - or use legacy single-user mapping when `--single-user-fallback` is set
3. If missing and `--strict-rootless` is on, fail startup.

If subuid/subgid entries exist but `newuidmap`/`newgidmap` are not installed, runk treats this as unusable subid mode and applies the same fallback logic unless strict mode is enabled.

If default `proot` fallback is selected but `proot` is not installed, runk fails with a remediation hint to install `proot` or use `--single-user-fallback`.

## Kernel preflight

runk checks:

- `/proc/sys/kernel/unprivileged_userns_clone` when present.
- `/proc/sys/user/max_user_namespaces` when present.

## Networking

- Default mode: `host`
- `slirp4netns` mode currently checks for `slirp4netns` binary and reserves integration hooks.

## Apt compatibility in fallback mode

When runk falls back (default `proot` mode or legacy single-user mapping), apt may fail to switch to its sandbox user (for example `_apt`) and can return setgroups/seteuid errors.

This compatibility behavior is applied only when apt is detected in the rootfs (for example Debian/Ubuntu images). Alpine `apk` images do not use this path.

To keep the PoC usable, runk automatically writes this file inside the container rootfs before `runc run`:

- `/etc/apt/apt.conf.d/99-runk-rootless`

With content:

- `APT::Sandbox::User "root";`
- `Acquire::Sandbox::User "root";` (compatibility fallback)

This keeps package signature verification in place but disables apt's privilege-drop sandbox user inside the container.

## DNS file mounts

To improve package-manager networking reliability in rootless runs, runk bind-mounts host resolver files into the container when present:

- `/etc/resolv.conf`
- `/etc/hosts`
- `/etc/hostname`

## Caveats

- This is a PoC and does not implement full runtime parity with Docker or Podman.
- Rootless cgroup delegation is not fully managed in this milestone.

## Runtime binary sourcing

- `runk` executes an external OCI runtime binary (`runc`) and does not link it as a library.
- Development and local builds can provision a sidecar runtime via `make runc-install`.
- Default runtime resolution uses this order:
   1. `--runtime`
   2. `RUNK_RUNTIME`
   3. sidecar `runc` next to the `runk` executable
   4. `runc` on `PATH`
