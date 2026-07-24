#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
manifest="${repo_root}/scripts/ci/qa-contracts.tsv"
mode="auto"
base=""
head="HEAD"
files_from=""

usage() {
  echo "usage: $0 [--mode auto|full] [--base SHA] [--head SHA] [--files-from PATH] [--manifest PATH]" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) mode="${2:?missing mode}"; shift 2 ;;
    --base) base="${2:?missing base SHA}"; shift 2 ;;
    --head) head="${2:?missing head SHA}"; shift 2 ;;
    --files-from) files_from="${2:?missing files path}"; shift 2 ;;
    --manifest) manifest="${2:?missing manifest path}"; shift 2 ;;
    *) usage; exit 2 ;;
  esac
done

if [[ "${mode}" != "auto" && "${mode}" != "full" ]]; then
  echo "invalid QA selection mode: ${mode}" >&2
  exit 2
fi
if [[ ! -f "${manifest}" ]]; then
  echo "QA contract manifest not found: ${manifest}" >&2
  exit 2
fi

files_tmp="$(mktemp "${TMPDIR:-/tmp}/labtether-qa-files.XXXXXX")"
selected_tmp="$(mktemp "${TMPDIR:-/tmp}/labtether-qa-selected.XXXXXX")"
cleanup() {
  rm -f -- "${files_tmp}" "${selected_tmp}"
}
trap cleanup EXIT

if [[ "${mode}" == "auto" ]]; then
  if [[ -n "${files_from}" ]]; then
    if [[ ! -f "${files_from}" ]]; then
      echo "changed-files input not found: ${files_from}" >&2
      exit 2
    fi
    cp "${files_from}" "${files_tmp}"
  elif [[ -z "${base}" || "${base}" =~ ^0+$ ]] \
    || ! git -C "${repo_root}" cat-file -e "${base}^{commit}" 2>/dev/null; then
    mode="full"
  else
    git -C "${repo_root}" diff \
      --name-only \
      --diff-filter=ACMRTUXB \
      "${base}" "${head}" > "${files_tmp}"
  fi
fi

while IFS=$'\t' read -r contract_id path_glob reason extra; do
  [[ -z "${contract_id}" || "${contract_id:0:1}" == "#" ]] && continue
  if [[ -z "${path_glob}" || -z "${reason}" || -n "${extra:-}" ]]; then
    echo "invalid QA contract row for '${contract_id}'; expected ID<TAB>GLOB<TAB>REASON" >&2
    exit 2
  fi

  if [[ "${mode}" == "full" ]]; then
    printf '%s\t%s\t%s\n' "${contract_id}" "${reason}" "[full mode]" >> "${selected_tmp}"
    continue
  fi

  while IFS= read -r changed_file; do
    [[ -z "${changed_file}" ]] && continue
    # shellcheck disable=SC2053 # The manifest value is intentionally a glob.
    if [[ "${changed_file}" == ${path_glob} ]]; then
      printf '%s\t%s\t%s\n' "${contract_id}" "${reason}" "${changed_file}" >> "${selected_tmp}"
      break
    fi
  done < "${files_tmp}"
done < "${manifest}"

deduped="$(awk -F '\t' '!seen[$1]++' "${selected_tmp}")"
contracts="$(printf '%s\n' "${deduped}" | awk -F '\t' 'NF { values = values separator $1; separator = "," } END { print values }')"

if [[ -n "${deduped}" ]]; then
  while IFS=$'\t' read -r contract_id reason trigger; do
    echo "QA contract ${contract_id}: ${reason} (trigger: ${trigger})"
  done <<< "${deduped}"
else
  echo "No slow QA contracts selected; fast checks remain required."
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "contracts=${contracts}"
    echo "mode=${mode}"
  } >> "${GITHUB_OUTPUT}"
fi
