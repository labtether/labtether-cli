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
labtether-cli assets list
labtether-cli exec <asset> "uptime"
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

With Go installed, you can also run:

```bash
go install github.com/labtether/labtether-cli@vX.Y.Z
```

## Quick start

```bash
labtether-cli config set-host https://your-hub:8443
labtether-cli config set-ca /path/to/labtether-ca.crt   # if your hub uses its own CA
labtether-cli config set-key                            # asks for your API key
labtether-cli whoami                                    # what this key can reach
labtether-cli assets list
```

Create the API key in the hub. Give it only the scopes you need.

## Commands

| Command | Description |
|:--------|:------------|
| `assets` | List and inspect managed assets |
| `agents` | Manage agent registrations and approvals |
| `exec` | Run a command on a remote asset |
| `services` | Manage system services on remote assets |
| `docker` | Inspect Docker hosts and containers; start, stop, restart, and read container logs |
| `files` | Browse directories and read remote files |
| `ps` | List and manage processes on assets |
| `alerts` | List, acknowledge, and resolve alerts |
| `incidents` | View and manage incidents |
| `updates` | Manage update plans and runs across fleet |
| `connectors` | Inspect and test configured hub connectors (Proxmox, TrueNAS, etc.) |
| `proxmox` | Interact with Proxmox clusters, VMs, and Ceph |
| `truenas` | Interact with TrueNAS pools, datasets, and shares |
| `pbs` | Interact with Proxmox Backup Server |
| `topology` | Explore asset dependency graphs and blast radius |
| `discovery` | Trigger scans and manage discovery proposals |
| `search` | Search across all hub objects |
| `audit` | View the audit event log |
| `config` | Manage CLI configuration (host, API key) |
| `whoami` | Show API key info, scopes, and accessible assets |

Run `labtether-cli --help` for everything, or `labtether-cli <command> --help`
for one command.

## JSON output

Add `--json` to get machine-readable results. Hub commands print the hub's
`data` payload, and config changes print a small status object. A failure
exits nonzero and prints an `error` object, unless a result was already
printed. A failed remote `exec` keeps its result and its nonzero exit code.
`files cat` always streams the original bytes and `docker logs` always prints
log text, even with `--json`; their errors go to standard error.

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
