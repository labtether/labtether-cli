# LabTether CLI in Rust

A Rust port of the Go CLI in this repository, built in small steps. The Go
CLI keeps shipping until the Rust one does the same job against a real hub.
Nothing here is released yet.

## What it does so far

The `labtether-cli-rs` binary has the same commands, aliases, flags, help
text, config file, environment variables and exit codes as the Go
`labtether-cli`. It runs:

- every command that only reads from the hub (lists, `get`, `status`,
  `map`, `metrics`, `whoami`, `search`, `audit`, `docker logs`, `files ls`
  and the rest), with the same table and JSON output byte for byte;
- `config set-host`, `set-key`, `set-ca` and `show`, which change only the
  local config file;
- `help`, `--version`, and help for every command, including the ones it
  does not run yet.

It reads settings in the Go CLI's order: flags, then `LABTETHER_HOST`,
`LABTETHER_API_KEY` and `LABTETHER_TLS_CA_FILE`, then
`~/.config/labtether/config.json`. `--api-key` is refused, as in Go. It uses
`HTTPS_PROXY`, `HTTP_PROXY` and `NO_PROXY` as the Go CLI does, and never a
proxy for localhost.

Some error texts differ from the Go CLI: connection, TLS and timeout
failures and Hub answers of an unexpected shape are described in plain
words. The exit codes are the same.

Connecting, the proxy and the TLS handshake together get 10 seconds, and the
whole request 5 minutes. Go allows 30 seconds to connect plus 10 for the
TLS handshake.

## Not yet

Commands that change something on the hub or a device, run remote commands,
transfer files or stream are refused with one line saying the command is
not in the Rust build yet, and exit 1: `exec`, `reboot`, `shutdown`, `wake`,
`files cat`, the `start`/`stop`/`restart` actions, `agents approve` and
`reject`, `alerts ack` and `resolve`, `issues investigate` and `resolve`,
`connectors test`, `groups create`, `ha call`, `packages install` and
`update`, `ps kill`, `discovery run`, `accept` and `dismiss`,
`failover check-readiness`, `updates plans execute` and `web-services sync`.
`completion bash|zsh|fish|powershell` and the hidden `__complete` command
that the scripts call are not in the Rust build yet either.

## Layout

| Folder | Holds |
|---|---|
| `crates/client` | The hub client on ureq: URL rules, TLS with the system roots and a private CA, proxy settings, same-origin redirects, size limit, timeout, the v2 envelope |
| `crates/platform` | Config file protection: mode 0600 on Unix, a private ACL on Windows (the only `unsafe` code) |
| `cli` | The binary: command tree and flag parsing (a copy of Cobra's rules, so flags, help and their errors match), settings, commands, tables, and JSON printed in the Go CLI's layout (a `serde_json` formatter) |
| `cli/src/types` | One serde struct per table, with exactly the fields the table prints |

## How to check it

Run these on a lab machine, not on a development laptop:

```bash
cd rust
rustup update stable
cargo fmt --check
cargo clippy --locked --all-targets -- -D warnings
cargo test --locked
```

`cargo test` includes the shared route table test: it reads
`../testdata/hub-routes.json`, the same file the Go tests read, so a command
added to one CLI and not the other fails both test suites.
