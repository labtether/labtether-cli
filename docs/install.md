# Download and check a release

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
