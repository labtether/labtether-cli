<div align="center">

<img src=".github/logo.svg" alt="LabTether" width="120" />

</div>

# LabTether CLI

Command-line interface for managing your [LabTether](https://labtether.com) hub.

[![Go](https://img.shields.io/badge/Go-1.27.2+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)

---

## Install

Choose an exact version from [Releases](https://github.com/labtether/labtether-cli/releases), download the binary plus its `.sha256` file, and verify both the checksum and GitHub build provenance before installation:

```bash
sha256sum --check labtether-cli-PLATFORM.sha256
gh attestation verify labtether-cli-PLATFORM -R labtether/labtether-cli
```

Or install an exact version with Go:

```bash
go install github.com/labtether/labtether-cli@vX.Y.Z
```

---

## Quick Start

```bash
# Configure your hub connection
labtether-cli config set-host https://your-hub:8443
labtether-cli config set-ca /path/to/labtether-ca.crt  # for the default private CA
labtether-cli config set-key  # prompts without exposing the key in argv

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
| `docker` | Inspect Docker hosts and containers; start, stop, restart, and read container logs |
| `files` | Browse directories and read remote files |
| `ps` | List and manage processes on assets |
| `alerts` | List, acknowledge, and silence alerts |
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

Run `labtether-cli --help` for the full command reference, or `labtether-cli <command> --help` for subcommand details.

All commands support `--json` for machine-readable output.

---

## Configuration

The CLI reads configuration from three sources, in order of priority:

1. **Flags** -- `--host`, `--api-key-file`, and `--tls-ca-file` on any command
2. **Environment variables** -- `LABTETHER_HOST`, `LABTETHER_API_KEY`, and `LABTETHER_TLS_CA_FILE`
3. **Config file** -- `~/.config/labtether/config.json`, written by `config set-host`, `config set-ca`, and `config set-key`

---

## Links

- **LabTether Hub** -- [github.com/labtether/labtether](https://github.com/labtether/labtether)
- **Documentation** -- [labtether.com/docs](https://labtether.com/docs)
- **Website** -- [labtether.com](https://labtether.com)

## License

[Apache 2.0](LICENSE)
