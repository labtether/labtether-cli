<div align="center">

<img src=".github/logo.svg" alt="LabTether" width="120" />

# LabTether CLI

**Run your [LabTether](https://github.com/labtether/labtether) homelab from the terminal.**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat-square)](LICENSE)

</div>

Do your day-to-day hub jobs from a shell: list your devices, run a command on
a machine, restart a container, read logs or acknowledge an alert. Every
command can print JSON, so it fits in scripts, cron jobs and AI agents too.

```bash
labtether-cli devices list
labtether-cli exec nas01 uptime
labtether-cli exec nas01 -- df -h
labtether-cli docker restart <container>
labtether-cli alerts list --json
```

## Install

Download the file for your platform from the
[latest release](https://github.com/labtether/labtether-cli/releases/latest)
and put it on your `PATH` as `labtether-cli`. Builds are there for Linux,
macOS and Windows on x86-64 and ARM64.

To check the download's checksum and GitHub attestation first, and for the extra step
macOS needs for files downloaded in a browser, see
[docs/install.md](docs/install.md).

To build from source with Go instead, see
[Build from source](docs/install.md#build-from-source). `go install` does not
install v2 releases.

## Quick start

```bash
labtether-cli config set-host https://your-hub:8443
labtether-cli config set-ca /path/to/labtether-ca.crt   # if your hub uses its own CA
labtether-cli config set-key                            # asks for your API key
labtether-cli whoami                                    # what this key can reach
labtether-cli devices list
```

Create the API key in the hub. Give it only the scopes you need.

## Commands

| Command | Description |
|:--------|:------------|
| `devices` | List devices and show one device's details |
| `agents` | List agents and approve new ones |
| `exec` | Run a command on one device or several |
| `services` | Start, stop and restart system services on a device |
| `docker` | List Docker hosts and containers; start, stop and restart containers and read their logs |
| `files` | List directories and read files on a device |
| `ps` | List processes on a device and send them signals |
| `alerts` | List, acknowledge and resolve alerts |
| `issues` | List, investigate and resolve issues |
| `updates` | List and run update plans and their runs |
| `connectors` | List and test hub connectors, such as Proxmox and TrueNAS |
| `proxmox` | Proxmox cluster status, VMs, containers and Ceph |
| `truenas` | TrueNAS pools, datasets and shares |
| `pbs` | Proxmox Backup Server datastores and snapshots |
| `map` | A device's connections, upstream and downstream |
| `discovery` | Start discovery scans and review suggested relationships |
| `search` | Search devices and groups |
| `audit` | Show the audit log |
| `config` | Save the hub URL, CA and API key |
| `whoami` | Show this API key's role, scopes and devices |

Run `labtether-cli --help` for everything, or `labtether-cli <command> --help`
for one command.

In 2.0 three commands took the console's page names: `assets` is now
`devices`, `incidents` is now `issues` and `topology` is now `map`. The old
names still run the same commands, so existing scripts keep working.

## Shell completion

`labtether-cli completion <shell>` prints a completion script for bash, zsh,
fish or PowerShell. For zsh:

```bash
labtether-cli completion zsh > "${fpath[1]}/_labtether-cli"
```

Run `labtether-cli completion <shell> --help` for the steps for each shell.

## JSON output

Add `--json` to get machine-readable results. Hub commands print the hub's
`data` payload, and config changes print a small status object. A failure
exits nonzero and prints an `error` object, unless a result was already
printed. A failed remote `exec` keeps its result and its nonzero exit code.
`files cat` always streams the original bytes and `docker logs` always prints
log text, even with `--json`; their errors go to standard error. JSON field
names are the hub's own, such as `asset_id` and `incidents`, and did not
change with the 2.0 command names.

## Configuration

Settings are read in this order, first match wins:

1. **Flags:** `--host`, `--api-key-file` and `--tls-ca-file` on any command
2. **Environment:** `LABTETHER_HOST`, `LABTETHER_API_KEY` and `LABTETHER_TLS_CA_FILE`
3. **Config file:** `~/.config/labtether/config.json`, written by `config set-host`, `config set-ca` and `config set-key`

## Contributing

If you work with an AI coding agent, point it at [AGENTS.md](AGENTS.md) first.
It covers the layout, the checks to run and the house rules.

## Help

- [CLI guide](https://labtether.com/docs/gateway/cli), [LabTether docs](https://labtether.com/docs) and [GitHub Discussions](https://github.com/labtether/labtether/discussions)
- Security issues: report privately as described in [SECURITY.md](SECURITY.md)

## License

[Apache 2.0](LICENSE). Copyright 2026 LabTether.
