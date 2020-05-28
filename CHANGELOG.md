# `active` Changelog

## 1.0.2 (2020-05-28)

#### Fixed

- Fixed a deadlock and a segfault.

## 1.0.1 (2020-05-25)

#### Changed

- When using `--pull`, the `master` branch is now pulled from the detected
  remote before analysis occurs.
- Performance improvements.

## 1.0.0 (2020-05-20)

This is the initial release of `active`.

#### Added

- Auto-detection of outdated Action versions.
- Config file support. Expected at `$HOME/.config/active.yaml`, but can be
  overridden with `--config`.
- OAuth support for querying the Github API without a rate limit.
- Automatic PR opening with the `--push` flag.
