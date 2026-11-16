#!/usr/bin/env bash
# Builds labtether-cli-rs for one target in release mode, runs it once on
# this runner, and copies it to the output folder under its asset name.
# The Rust CLI build workflow (.github/workflows/rust-cli-build.yml) calls
# this once per target; it publishes nothing.
#
# Usage: build-rust-cli.sh <rust-target> <asset-name> <version> <out-dir>
# SMOKE_PREFIX, if set, runs the binary through that command, for example
# "arch -x86_64" to run an Intel macOS build under Rosetta 2.
set -Eeuo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <rust-target> <asset-name> <version> <out-dir>" >&2
  exit 2
fi
target="$1"
asset="$2"
version="$3"
out_dir="$4"

# The version is baked into the binary and compared below; keep it to
# characters a release tag or a dev label uses.
if [[ ! "${version}" =~ ^[A-Za-z0-9][A-Za-z0-9._+-]*$ ]]; then
  echo "version '${version}' may only hold letters, digits and . _ + -" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
rust_dir="${repo_root}/rust"

on_windows=false
case "$(uname -s)" in
  MINGW* | MSYS* | CYGWIN*) on_windows=true ;;
esac

# Windows needs native paths in rustc flags and in USERPROFILE.
native_path() {
  if [[ "${on_windows}" == true ]]; then
    cygpath -w "$1"
  else
    printf '%s\n' "$1"
  fi
}

cd "${rust_dir}"
rustup update stable --no-self-update
rustup target add --toolchain stable "${target}"
rustc --version
cargo --version

# Paths of this checkout and of the registry are left out of the binary, so
# the same source builds the same file on another runner. Windows builds
# link the C runtime statically, as the Go CLI needs no runtime installed,
# and ask the linker for a fixed PE timestamp.
cargo_home="${CARGO_HOME:-${HOME}/.cargo}"
rustflags=(
  "--remap-path-prefix=$(native_path "${repo_root}")=."
  "--remap-path-prefix=$(native_path "${cargo_home}")=cargo"
)
case "${target}" in
  *-windows-msvc) rustflags+=("-Ctarget-feature=+crt-static" "-Clink-arg=/Brepro") ;;
esac
export RUSTFLAGS="${rustflags[*]}"
# ring's C code for a musl target is compiled with musl-tools' gcc wrapper.
if [[ "${target}" == *-linux-musl ]]; then
  export "CC_${target//-/_}=musl-gcc"
fi
export LABTETHER_CLI_VERSION="${version}"
export CARGO_INCREMENTAL=0

cargo build --locked --release --target "${target}" -p labtether-cli-rs

suffix=""
[[ "${target}" == *-windows-* ]] && suffix=".exe"
built="${rust_dir}/target/${target}/release/labtether-cli-rs${suffix}"
mkdir -p "${out_dir}"
binary="${out_dir}/${asset}"
cp "${built}" "${binary}"

# The file must be for the asked architecture, not the runner's own.
case "${target}" in
  *-apple-darwin)
    want="${target%%-*}"
    [[ "${want}" == aarch64 ]] && want=arm64
    got="$(lipo -archs "${binary}")"
    [[ "${got}" == "${want}" ]] || { echo "${asset} is ${got}, expected ${want}" >&2; exit 1; }
    ;;
  *-linux-musl)
    file "${binary}"
    file "${binary}" | grep -q "static" || { echo "${asset} is not statically linked" >&2; exit 1; }
    ;;
esac

# Smoke run: the version this build was given, and `config show` with an
# empty home folder and no LabTether settings in the environment.
read -r -a prefix <<< "${SMOKE_PREFIX:-}"
reported="$(${prefix[@]+"${prefix[@]}"} "${binary}" --version)"
if [[ "${reported}" != "labtether-cli version ${version}" ]]; then
  echo "${asset} reports '${reported}', expected 'labtether-cli version ${version}'" >&2
  exit 1
fi

home="$(mktemp -d)"
trap 'rm -rf -- "${home}"' EXIT
shown="$(env -u LABTETHER_HOST -u LABTETHER_API_KEY -u LABTETHER_TLS_CA_FILE \
  HOME="${home}" USERPROFILE="$(native_path "${home}")" \
  ${prefix[@]+"${prefix[@]}"} "${binary}" config show)"
printf '%s\n' "${shown}"
for line in "API key: (not set)" "TLS CA:  (system trust only)"; do
  grep -qxF "${line}" <<< "${shown//$'\r'/}" || { echo "config show is missing '${line}'" >&2; exit 1; }
done
if [[ -n "$(find "${home}" -mindepth 1 -print -quit)" ]]; then
  echo "config show wrote into the empty home folder" >&2
  exit 1
fi

echo "built and ran ${asset} (${target}, version ${version})"
