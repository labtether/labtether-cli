# LabTether CLI in Rust

A Rust port of the Go CLI in this repository, built in small steps. The Go
CLI keeps shipping until the Rust one does the same job against a real hub.
Nothing here is released yet.

## What it does so far

The `labtether-cli-rs` binary has the same commands, aliases, flags, help
text, config file, environment variables and exit codes as the Go
`labtether-cli`, and runs all of them:

- the commands that read from the hub, with the same table and JSON output
  byte for byte;
- the commands that change something (`exec`, `reboot`, `shutdown`,
  `wake`, the `start`/`stop`/`restart` actions, `agents approve`,
  `issues resolve` and the rest), with the same requests and request
  bodies, output and exit codes. Like the Go CLI they ask no question and
  read nothing from standard input;
- `files cat`, which writes the file as it arrives, at most 512 MiB;
- `config set-host`, `set-key`, `set-ca` and `show`;
- `help`, `--version`, help for every command, `completion
  bash|zsh|fish|powershell` (Cobra's scripts) and the hidden `__complete`
  command those scripts call.

It reads settings in the Go CLI's order: flags, then `LABTETHER_HOST`,
`LABTETHER_API_KEY` and `LABTETHER_TLS_CA_FILE`, then
`~/.config/labtether/config.json`. `--api-key` is refused, as in Go. It uses
`HTTPS_PROXY`, `HTTP_PROXY` and `NO_PROXY` as the Go CLI does, and never a
proxy for localhost.

Without a home folder it reads no config file and saves none, as the Go
CLI: set `LABTETHER_HOST` and `LABTETHER_API_KEY`. One difference:
`config set-key` refuses before it asks for the key, where Go asks first.
The temporary copy of the config file is private from the moment it
exists; on Windows it gets its private ACL when it is created.

Secret values in Hub answers (keys naming a password, secret, token, API
key or private key) are shown as `[redacted]` in `--json` output, in table
cells that hold an object, and in the error text for an answer that is not
the Hub's usual JSON, with the Go CLI's rules and words.

Some error texts differ from the Go CLI: connection, TLS and timeout
failures and Hub answers of an unexpected shape are described in plain
words. The exit codes are the same.

Connecting, the proxy and the TLS handshake together get 10 seconds, and the
whole request 5 minutes (12 minutes for package and Proxmox actions, as in
Go). Go allows 30 seconds to connect plus 10 for the TLS handshake.

## Layout

| Folder | Holds |
|---|---|
| `crates/client` | The hub client on ureq: URL rules, TLS with the system roots and a private CA, proxy settings, same-origin redirects, size limit, timeout, the v2 envelope; the secret redaction and the Go-style JSON writer the CLI prints with |
| `crates/platform` | Config file protection: mode 0600 on Unix, a private ACL on Windows, set when the file is created (the only `unsafe` code) |
| `cli` | The binary: command tree and flag parsing (a copy of Cobra's rules, so flags, help and their errors match), settings, commands and tables |
| `cli/src/completion` | Cobra 1.10.2's completion scripts for this CLI, as the Go CLI prints them |
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
