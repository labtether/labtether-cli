# Building the Rust CLI for release

The workflow `.github/workflows/rust-cli-build.yml` builds `labtether-cli-rs`
for the six targets the Go CLI ships. It keeps the result as workflow
artifacts for five days and publishes nothing.

| Asset | Built on | Run on |
|---|---|---|
| `labtether-cli-rs-linux-amd64` | `ubuntu-24.04`, musl, static | same runner |
| `labtether-cli-rs-linux-arm64` | `ubuntu-24.04-arm`, musl, static | same runner |
| `labtether-cli-rs-darwin-amd64` | `macos-26` (arm64), x86_64 target | same runner, Rosetta 2 |
| `labtether-cli-rs-darwin-arm64` | `macos-26` | same runner |
| `labtether-cli-rs-windows-amd64.exe` | `windows-2025-vs2026`, static C runtime | same runner |
| `labtether-cli-rs-windows-arm64.exe` | `windows-11-arm`, static C runtime | same runner |

Each job runs `scripts/release/build-rust-cli.sh`: latest stable Rust,
`cargo build --locked --release`, then the binary runs `--version` and
`config show` with an empty home folder. Each binary gets a GitHub build
attestation (`actions/attest`), and a last job writes `SHA256SUMS` in the
Go release's format. There is no build cache, so every attested binary
comes from a clean build.

## Start a build

From the Actions tab, or:

```bash
gh workflow run rust-cli-build.yml --repo labtether/labtether-cli \
  -f ref=main -f version=dev
```

`version` is what `labtether-cli-rs --version` prints. A pull request that
changes the workflow or the build script runs it once.

## Check a binary

```bash
gh run download <run-id> --repo labtether/labtether-cli --dir rust-cli
cd rust-cli
shasum -a 256 -c SHA256SUMS        # sha256sum -c on Linux
gh attestation verify labtether-cli-rs-linux-amd64 \
  --repo labtether/labtether-cli \
  --signer-workflow labtether/labtether-cli/.github/workflows/rust-cli-build.yml
```

`gh attestation verify --bundle labtether-cli-rs-linux-amd64.sigstore.json`
checks against the bundle kept with the binary instead of GitHub's copy.

## Not done yet

| Step | Waits for |
|---|---|
| Attaching the binaries to a GitHub release or tagging them | The owner's decision to publish the Rust CLI |
| Signing with the project's Ed25519 release key | The same decision, and the owner's go-ahead for each release |
| Apple Developer ID signing and notarization, Windows Authenticode signing | The same decision; signing runs only with the owner's go-ahead |
| Replacing the Go CLI with the Rust one | The Rust CLI doing the same job against a real hub on the lab machines, and the owner's decision |
| Archives with the license and readme, as the Go release ships | The decision to publish |
