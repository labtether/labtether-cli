#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
selector="${repo_root}/scripts/ci/select-qa-contracts.sh"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/labtether-qa-selector-test.XXXXXX")"
cleanup() {
  rm -rf -- "${tmp_dir}"
}
trap cleanup EXIT

printf '%s\n' "cmd/config_security_windows.go" > "${tmp_dir}/files"
output="$("${selector}" --files-from "${tmp_dir}/files")"
grep -Fq "QA contract cross-platform-build:" <<< "${output}"
grep -Fq "QA contract tls-permission-behavior:" <<< "${output}"

printf '%s\n' "main.go" > "${tmp_dir}/files"
output="$("${selector}" --files-from "${tmp_dir}/files")"
grep -Fq "QA contract installed-cli-package:" <<< "${output}"
grep -Fq "QA contract cross-platform-build:" <<< "${output}"

output="$("${selector}" --mode full)"
grep -Fq "QA contract installed-cli-package:" <<< "${output}"
grep -Fq "QA contract cross-platform-build:" <<< "${output}"
grep -Fq "QA contract tls-permission-behavior:" <<< "${output}"

output="$("${selector}" --base 0000000000000000000000000000000000000000 --head HEAD)"
grep -Fq "QA contract installed-cli-package:" <<< "${output}"
grep -Fq "QA contract cross-platform-build:" <<< "${output}"
grep -Fq "QA contract tls-permission-behavior:" <<< "${output}"

printf 'broken\trow\n' > "${tmp_dir}/bad-manifest"
if "${selector}" --files-from "${tmp_dir}/files" --manifest "${tmp_dir}/bad-manifest" >/dev/null 2>&1; then
  echo "malformed QA manifest was accepted" >&2
  exit 1
fi

echo "QA contract selector tests passed"
