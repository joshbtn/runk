# Rootless behavior in runk PoC

## ID mapping

runk resolves ID mapping in this order:

1. Use `/etc/subuid` and `/etc/subgid` ranges for the current user when present.
2. If missing and strict mode is off, fallback to single mapping:
   - container UID 0 -> host effective UID
   - container GID 0 -> host effective GID
   - mapping size = 1
3. If missing and `--strict-rootless` is on, fail startup.

## Kernel preflight

runk checks:

- `/proc/sys/kernel/unprivileged_userns_clone` when present.
- `/proc/sys/user/max_user_namespaces` when present.

## Networking

- Default mode: `host`
- `slirp4netns` mode currently checks for `slirp4netns` binary and reserves integration hooks.

## Apt compatibility in single-ID mode

When runk falls back to a single UID/GID mapping (size = 1), apt may fail to switch to its sandbox user (for example `_apt`) and can return setgroups/seteuid errors.

This compatibility behavior is applied only when apt is detected in the rootfs (for example Debian/Ubuntu images). Alpine `apk` images do not use this path.

To keep the PoC usable, runk automatically writes this file inside the container rootfs before `runc run`:

- `/etc/apt/apt.conf.d/99-runk-rootless`

With content:

- `Acquire::Sandbox::User "root";`

This keeps package signature verification in place but disables apt's privilege-drop sandbox user inside the container.

## DNS file mounts

To improve package-manager networking reliability in rootless runs, runk bind-mounts host resolver files into the container when present:

- `/etc/resolv.conf`
- `/etc/hosts`
- `/etc/hostname`

## Caveats

- This is a PoC and does not implement full runtime parity with Docker or Podman.
- Rootless cgroup delegation is not fully managed in this milestone.
