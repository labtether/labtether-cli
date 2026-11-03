# Changelog

All notable changes to the LabTether CLI.

## v2.0.0

Changes since v1.7.1. The CLI takes the LabTether 2.0 version with the Hub and
agents. Apart from the changes below, commands, flags, output and exit codes
are the same as in v1.7.1. The CLI works with Hub 2.0, whose API has the same
routes as Hub v1.7.1.

### Command names

Commands use the console's 2.0 page names. The old names still work as hidden
aliases: they run the same command and print the same output, with no warning
on stdout or stderr, so existing scripts keep running. Help and shell
completion show only the new names.

| v1.7.1 | v2.0.0 |
|:-------|:-------|
| `assets` | `devices` |
| `incidents` | `issues` |
| `incidents ack` | `issues investigate` (`ack` already ran it) |
| `topology` | `map` |
| `topology blast-radius` | `map downstream` |
| `topology edges` | `map connections` |
| `metrics asset` | `metrics device` |

### Output

- `--json` output is unchanged. JSON field names are the Hub's own, such as
  `asset_id`, `assets` and `incidents`, and scripts that read them keep
  working.
- Tables use the console's status words, such as `Online`, `Offline`,
  `Not responding`, `Firing` and `Resolved`, instead of the Hub's lowercase
  values. A device that is offline or not responding shows when it was last
  seen: `Offline since 21:12`, or the date when that was more than a day ago.
- Some table headers use the console's words: alerts and issues show STATE
  (was STATUS), `devices list` shows ADDRESS (was IP), and `collectors list`
  and `hub agents` show DEVICE (was ASSET). `search` names a device result
  `device`, not `asset`. Column order is unchanged.
- `whoami` lists "Devices" with `Online` or `Offline` for each.
- `devices list` shows the same address as the console. It read only the
  `ip` field, so a TrueNAS, PBS or Portainer device, whose address is its
  connection's host, showed `-`.
- `alerts list` names the rule that fired instead of showing its ID, and adds
  a DEVICE column with the devices the alert fired on.
- `files ls` shows the modified time in local time, as other tables do, and
  drops terminal control characters from file names.
- `whoami` keeps its device columns aligned when a device ID is long.
- Help text uses plain words and the new names, and its examples use neutral
  hosts such as `nas01` and `pve01`.

### Commands

- `connectors test` works again. It sent no connection settings, so the Hub
  answered "invalid test payload" for every connector except Docker. It now
  tests the saved connection for the connector, such as
  `connectors test proxmox`, using the connection's address and stored
  credential; no secret passes through the CLI. When a connector has more
  than one saved connection, the error lists their collector IDs and
  `connectors test <collector-id>` tests one.
- `ha entities` shows each entity's ID first. `ha entity` and `ha call` take
  that ID, not the Home Assistant entity name such as `switch.office`, and the
  table did not show it.
- `exec --group` help says it takes a group ID, as shown by `groups list`. The
  Hub matches the group by ID, never by name.
- `exec` says how to pass a remote command's own flags. `exec nas01 df -h`
  printed help, ran nothing and exited 0; `exec nas01 free -m` failed with only
  "unknown shorthand flag". Both now exit 1 and say to quote the command or
  put it after `--`, as in `exec nas01 -- df -h`.

- `metrics overview` printed `null`. The Hub answers that route without its
  usual data wrapper; the CLI now prints the devices and their metrics.
- An `exec` flag with a bad value, such as `--timeout 0`, no longer adds the
  hint about remote command flags.
- `exec --targets` reports each device that has no result and exits 1. The
  hub leaves out devices the API key cannot reach, so a key limited to one
  device printed only that device's output and exited 0.

### Exit codes

- Configuration errors exit 4, as the CLI guide says. Only a missing hub URL
  or API key did; a plain-HTTP or malformed hub URL, an API key file with
  open permissions, a missing or invalid CA file and a broken config file
  exited 1.
- `config set-host` checks the URL before saving it. It saved any text, and
  a plain-HTTP or malformed URL failed only on the next command.
- `connectors get` with an unknown connector exits 3, like other not-found
  errors, instead of 1.

### Install

- `go install github.com/labtether/labtether-cli@...` does not install v2
  releases, because the module path has no `/v2` suffix: `@latest` stays on
  v1.7.1 and `@v2.0.0` fails. The README and `docs/install.md` now show how to
  build a release tag from source instead. Release downloads are unchanged.

## v1.7.1

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
- `exec` reads the Hub's `status` to decide success and shows the exit code
  only when the Hub reports one. It no longer stops with "missing exit_code"
  when a command times out; it prints the Hub's output or reason instead. A
  failed single-asset `exec` exits with the remote exit code when it is known
  and between 1 and 255, otherwise 1. Older Hubs that send only `exit_code`
  still work.
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
