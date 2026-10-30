# Changelog

All notable changes to the LabTether CLI.

## Unreleased

Changes since v1.7.0.

### Commands

- List commands print a table instead of raw JSON: `agents list` and
  `agents pending`, `services list`, `web-services list`, `search`, and the
  alerts, checks, collectors, connectors, credentials, groups, failover,
  incidents, updates, notifications, discovery, Home Assistant, audit,
  `hub agents`, Proxmox, packages, users, disks, net, cron, PBS snapshots and
  Portainer lists. `--json` output is unchanged. Single-item commands such as
  `get` and `status` still print JSON.
- Tables print whole numbers as integers (a PID of 1942945 no longer prints as
  `1.942945e+06`), show `-` for a missing value instead of `<nil>`, and align
  their columns.
- `docker ps` shows the 12-character container ID. With a Hub that includes
  labtether/labtether#234, `docker logs`, `start`, `stop` and `restart` accept
  it.
- `search` prints "No results." when nothing matches, instead of `{}`.
- `pbs datastores` works again. It called a Hub route that does not exist and
  always failed; it now reads the datastore list from the PBS details and
  prints STORE, STATUS, SIZE, USED, AVAIL, USE%, GROUPS, SNAPSHOTS and
  LAST BACKUP. `--json` prints that list.
- Usage text follows only command-line mistakes, such as a missing argument or
  an unknown flag. Hub and network errors print the error alone.

## v1.7.0

Changes since v1.6.5.

### Commands

- Commands use the current Hub v2 API responses, including Docker, files,
  Proxmox, topology, discovery, connectors and alerts.
- Long Hub actions, such as package updates and Proxmox tasks, are allowed up
  to 12 minutes before the CLI gives up waiting.
- `files cat` streams the original file bytes and stops at 512 MiB.
- `--json` is now a stable machine-readable contract: action commands print the
  Hub's `data` payload, configuration changes print a small status object, and
  failures print one `error` object with a nonzero exit code. A failed remote
  `exec` keeps its result and exit code. `files cat` and `docker logs` still
  print raw output.
- Built with Go 1.27.3 (was 1.26) and current stable dependencies.

### Release assets

- Added Windows ARM64 (`labtether-cli-windows-arm64.exe` and its zip).
- Each release has one `SHA256SUMS` file instead of a `.sha256` file per asset.
  See the README for the verification steps.
- Windows archives are now named `labtether-cli-windows-<arch>-vX.Y.Z.zip`
  (previously `labtether-cli-windows-amd64.exe-vX.Y.Z.zip`).
- Archives also contain `LICENSE` and `README.md`. Binaries and archives carry
  GitHub build provenance attestations.
- The release is created once, after every platform builds, and only for tags
  of the form `vX.Y.Z`.
- macOS binaries are still not signed with an Apple Developer ID or notarized.

### Project

- Licensed under the Apache License 2.0 (the README already said so; the
  license file is now included). Added a security policy.
- CI enforces a 500-code-line limit per handwritten file and checks the CLI
  configuration file permissions on Unix and Windows.
