<div align="center">

<img src=".github/logo.svg" alt="LabTether" width="120" />

</div>

# LabTether CLI

Command-line interface for managing your [LabTether](https://labtether.com) hub.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)

---

## Install

Download the latest binary for your platform from [Releases](https://github.com/labtether/labtether-cli/releases/latest).

Or install with Go:

```bash
go install github.com/labtether/labtether-cli@latest
```

---

## Quick Start

```bash
# Configure your hub connection
labtether-cli config set-host https://your-hub:8443
labtether-cli config set-key lt_your_api_key

# Check who you are and what you can access
labtether-cli whoami

# View hub and fleet status
labtether-cli assets list
```

---

## Commands

| Command | Description |
|:--------|:------------|
| `assets` | List and inspect managed assets |
| `agents` | Manage agent registrations and approvals |
| `exec` | Run a command on a remote asset |
| `services` | Manage system services on remote assets |
| `docker` | Manage Docker containers, images, and hosts |
| `files` | Upload, download, and browse remote files |
| `ps` | List and manage processes on assets |
| `alerts` | List, acknowledge, and silence alerts |
| `incidents` | View and manage incidents |
| `updates` | Manage update plans and runs across fleet |
| `connectors` | Manage hub connectors (Proxmox, TrueNAS, etc.) |
| `proxmox` | Interact with Proxmox clusters, VMs, and Ceph |
| `truenas` | Interact with TrueNAS pools, datasets, and shares |
| `pbs` | Interact with Proxmox Backup Server |
| `topology` | Explore asset dependency graphs and blast radius |
| `discovery` | Trigger scans and manage discovery proposals |
| `search` | Search across all hub objects |
| `audit` | View the audit event log |
| `config` | Manage CLI configuration (host, API key) |
| `whoami` | Show API key info, scopes, and accessible assets |

Run `labtether-cli --help` for the full command reference, or `labtether-cli <command> --help` for subcommand details.

All commands support `--json` for machine-readable output.

---

## Configuration

The CLI reads configuration from three sources, in order of priority:

1. **Flags** -- `--host` and `--api-key` on any command
2. **Environment variables** -- `LABTETHER_HOST` and `LABTETHER_API_KEY`
3. **Config file** -- `~/.config/labtether/config.json`, written by `config set-host` and `config set-key`

---

## Links

- **LabTether Hub** -- [github.com/labtether/labtether](https://github.com/labtether/labtether)
- **Documentation** -- [labtether.com/docs](https://labtether.com/docs)
- **Website** -- [labtether.com](https://labtether.com)

## License

[Apache 2.0](LICENSE)
