<div align="center">

<img src=".github/logo.svg" alt="LabTether" width="120" />

</div>

# LabTether CLI

Command-line interface for managing your [LabTether](https://labtether.com) hub.

[![Go](https://img.shields.io/badge/Go-1.27.2+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)

---

## Install

Each [release](https://github.com/labtether/labtether-cli/releases) has these
assets (`vX.Y.Z` is the release tag):

| Platform | Binary | Archive (binary, `LICENSE`, `README.md`) |
|:--|:--|:--|
| Linux x86-64 | `labtether-cli-linux-amd64` | `labtether-cli-linux-amd64-vX.Y.Z.tar.gz` |
| Linux ARM64 | `labtether-cli-linux-arm64` | `labtether-cli-linux-arm64-vX.Y.Z.tar.gz` |
| macOS Intel | `labtether-cli-darwin-amd64` | `labtether-cli-darwin-amd64-vX.Y.Z.tar.gz` |
| macOS Apple silicon | `labtether-cli-darwin-arm64` | `labtether-cli-darwin-arm64-vX.Y.Z.tar.gz` |
| Windows x86-64 | `labtether-cli-windows-amd64.exe` | `labtether-cli-windows-amd64-vX.Y.Z.zip` |
| Windows ARM64 | `labtether-cli-windows-arm64.exe` | `labtether-cli-windows-arm64-vX.Y.Z.zip` |
| All | `SHA256SUMS` | SHA-256 checksums for every asset above |

Download the file for your platform and `SHA256SUMS` from the same release,
then check both the checksum and the GitHub build provenance:

```bash
VERSION=vX.Y.Z
FILE=labtether-cli-linux-amd64
gh release download "$VERSION" -R labtether/labtether-cli -p "$FILE" -p SHA256SUMS
sha256sum --check --ignore-missing SHA256SUMS   # macOS: shasum -a 256 --check --ignore-missing SHA256SUMS
gh attestation verify "$FILE" -R labtether/labtether-cli
chmod +x "$FILE" && sudo mv "$FILE" /usr/local/bin/labtether-cli
```

On Windows PowerShell, compare `(Get-FileHash .\labtether-cli-windows-amd64.exe).Hash`
with the matching line in `SHA256SUMS`.

macOS binaries are not signed with an Apple Developer ID or notarized. If you
download one with a web browser, Gatekeeper blocks it; after verifying the
checksum, clear the quarantine flag:

```bash
xattr -d com.apple.quarantine labtether-cli-darwin-arm64
```

Files downloaded with `gh` or `curl` are not quarantined.

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

Run `labtether-cli --help` for the full command reference, or `labtether-cli <command> --help` for subcommand details.

Use `--json` for machine-readable read and action results. Hub commands return
the Hub's `data` payload; configuration changes return a small status object.
Failures return a nonzero exit code and an `error` object if no result was
already printed. A failed remote `exec` keeps its result and nonzero exit code.
`files cat` always streams the original bytes and `docker logs` always prints
log text, even when `--json` is set. Their errors stay on standard error.

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

## Security

Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## License

Apache License 2.0. See [LICENSE](LICENSE).
