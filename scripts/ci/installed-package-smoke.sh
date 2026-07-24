#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/labtether-cli-package-smoke.XXXXXX")"
cleanup() {
  rm -rf -- "${tmp_dir}"
}
trap cleanup EXIT

version="qa-contract"
target_os="${QA_TARGET_GOOS:-$(go env GOOS)}"
target_arch="${QA_TARGET_GOARCH:-$(go env GOARCH)}"
host_os="$(go env GOOS)"
host_arch="$(go env GOARCH)"
if [[ "${target_os}:${target_arch}" != "${host_os}:${host_arch}" ]]; then
  echo "installed smoke target ${target_os}/${target_arch} is not executable on ${host_os}/${host_arch}" >&2
  exit 2
fi
binary="${tmp_dir}/labtether-cli-${target_os}-${target_arch}"
archive="${tmp_dir}/labtether-cli-${target_os}-${target_arch}-${version}.tar.gz"

(
  cd "${repo_root}"
  CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" go build \
    -trimpath \
    -ldflags="-s -w -X github.com/labtether/labtether-cli/cmd.version=${version}" \
    -o "${binary}" \
    .
)
tar -C "${tmp_dir}" -czf "${archive}" "$(basename "${binary}")"
expected_checksum="$(sha256sum "${archive}" | awk '{print $1}')"
[[ "${expected_checksum}" =~ ^[0-9a-f]{64}$ ]]

mkdir "${tmp_dir}/installed"
tar -C "${tmp_dir}/installed" -xzf "${archive}"
installed="${tmp_dir}/installed/$(basename "${binary}")"
if [[ "$("${installed}" --version)" != "labtether-cli version ${version}" ]]; then
  echo "installed CLI did not report the packaged version" >&2
  exit 1
fi
"${installed}" --help >/dev/null

echo "installed CLI package smoke passed (${target_os}/${target_arch}, version ${version})"
