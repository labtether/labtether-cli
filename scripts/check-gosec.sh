#!/usr/bin/env bash
# Runs gosec and fails on any finding, and also when gosec could not load or
# analyse every Go package. In the other Go repos gosec v2.29.0 could not read
# Go 1.27 packages; with -quiet it then returned success before counting its
# Go errors, so their scans analysed nothing (found 2026-11-16). Copied from
# the protocol repo's scripts/check-gosec.sh.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v gosec >/dev/null 2>&1; then
  echo "gosec not found in PATH." >&2
  echo "install the version pinned in .github/workflows/ci.yml:" >&2
  echo "  go install github.com/securego/gosec/v2/cmd/gosec@v2.30.0" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf -- "${tmp_dir}"' EXIT

(cd "${ROOT_DIR}" && gosec -fmt=json ./... >"${tmp_dir}/gosec.json" 2>"${tmp_dir}/gosec.log") || true
if ! jq -e '.' "${tmp_dir}/gosec.json" >/dev/null 2>&1; then
  echo "gosec check failed: scanner did not produce valid JSON output" >&2
  tail -n 20 "${tmp_dir}/gosec.log" >&2 || true
  exit 1
fi

go_errors="$(jq '[(.["Golang errors"] // {})[] | length] | add // 0' "${tmp_dir}/gosec.json")"
if [[ "${go_errors}" != "0" ]]; then
  echo "gosec check failed: gosec could not load the code (${go_errors} Go errors)" >&2
  jq -r '(.["Golang errors"] // {}) | to_entries[] | .key as $f | .value[] | "\($f):\(.line): \(.error)"' \
    "${tmp_dir}/gosec.json" | head -n 5 >&2 || true
  echo "Use a gosec release built for this Go version." >&2
  exit 1
fi

if grep -F "Error building the SSA representation" "${tmp_dir}/gosec.log" >&2; then
  echo "gosec check failed: gosec skipped analysis of the packages above" >&2
  exit 1
fi

(cd "${ROOT_DIR}" && go list -f '{{.Dir}}' ./...) | LC_ALL=C sort -u >"${tmp_dir}/expected"
sed -n 's/^.*Import directory: //p' "${tmp_dir}/gosec.log" | LC_ALL=C sort -u >"${tmp_dir}/analysed"
missing="$(comm -23 "${tmp_dir}/expected" "${tmp_dir}/analysed")"
if [[ ! -s "${tmp_dir}/expected" || -n "${missing}" ]]; then
  echo "gosec check failed: gosec did not analyse every Go package" >&2
  printf '%s\n' "${missing}" >&2
  exit 1
fi

findings="$(jq -r '.Issues[]? | "\(.rule_id) \(.file):\(.line) \(.details)"' "${tmp_dir}/gosec.json")"
if [[ -n "${findings}" ]]; then
  echo "gosec check failed: findings detected" >&2
  printf '%s\n' "${findings}" >&2
  exit 1
fi

echo "gosec check passed ($(wc -l <"${tmp_dir}/expected" | tr -d '[:space:]') packages, no findings)."
