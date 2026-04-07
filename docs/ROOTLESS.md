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

## Caveats

- This is a PoC and does not implement full runtime parity with Docker or Podman.
- Rootless cgroup delegation is not fully managed in this milestone.
