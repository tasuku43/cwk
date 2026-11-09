#!/usr/bin/env bash
# shellcheck disable=SC2154
# Sourceable helper: release_input_roots, release_root, and sha256_of belong to
# the caller. Explicit checks remain effective in command substitutions.
release_input_fingerprint() {
  local diagnostic=${1:-}
  local manifest path_list unsafe_path_list path digest optional_input expected_bytes consumed_bytes=0
  local LC_ALL=C
  local -a fingerprint_input_roots=("${release_input_roots[@]}")
  for optional_input in go.sum vendor; do
    if [[ -e $optional_input || -L $optional_input ]]; then
      fingerprint_input_roots+=("$optional_input")
    fi
  done
  manifest=$(mktemp "$release_root/package-inputs.XXXXXXXX") || return 1
  path_list=$(mktemp "$release_root/package-input-paths.XXXXXXXX") || return 1
  unsafe_path_list=$(mktemp "$release_root/package-input-unsafe-paths.XXXXXXXX") || return 1
  : >"$manifest" || return 1
  if ! find "${fingerprint_input_roots[@]}" ! -type d ! -type f -print0 >"$unsafe_path_list"; then
    echo "could not inspect every release input" >&2
    return 1
  fi
  if [[ -s $unsafe_path_list ]]; then
    echo "release inputs must contain only regular files and directories" >&2
    return 1
  fi
  if ! (set -o pipefail; find "${fingerprint_input_roots[@]}" -type f -print0 | LC_ALL=C sort -z >"$path_list"); then
    echo "could not enumerate every release input" >&2
    return 1
  fi
  if [[ ! -s $path_list ]]; then
    echo "release input set is empty" >&2
    return 1
  fi
  if [[ -n $diagnostic ]]; then
    : >"$diagnostic" || return 1
  fi
  expected_bytes=$(wc -c <"$path_list") || return 1
  [[ $expected_bytes =~ ^[[:space:]]*[0-9]+[[:space:]]*$ ]] || return 1
  while IFS= read -r -d '' path; do
    consumed_bytes=$((consumed_bytes + ${#path} + 1))
    digest=$(sha256_of "$path") || return 1
    [[ $digest =~ ^[0-9a-f]{64}$ ]] || return 1
    printf '%s\0%s\0' "$path" "$digest" >>"$manifest" || return 1
    if [[ -n $diagnostic ]]; then
      printf '%s  %q\n' "$digest" "$path" >>"$diagnostic" || return 1
    fi
  done <"$path_list" || return 1
  if [[ $consumed_bytes -ne $expected_bytes ]]; then
    echo "release input path list could not be read completely" >&2
    return 1
  fi
  digest=$(sha256_of "$manifest") || return 1
  [[ $digest =~ ^[0-9a-f]{64}$ ]] || return 1
  rm -f -- "$manifest" "$path_list" "$unsafe_path_list" || return 1
  printf '%s' "$digest"
}
