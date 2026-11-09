#!/usr/bin/env bash
set -euo pipefail
project_root=$(cd "$(dirname "$0")/.." && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-fingerprint-test.XXXXXXXX")
trap 'rm -rf -- "$test_root"' EXIT
mkdir "$test_root/input" "$test_root/work"
cd "$test_root/input"
printf 'fixture\n' > required
release_root="$test_root/work"
release_input_roots=(required)
# shellcheck source=scripts/release-input-fingerprint.sh
source "$project_root/scripts/release-input-fingerprint.sh"
sha256_of() {
  if [[ ${FAIL_TEST_HASH:-0} == 1 ]]; then return 1; fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -- "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -- "$1" | awk '{print $1}'
  else
    return 1
  fi
}
first=$(release_input_fingerprint)
[[ $first =~ ^[0-9a-f]{64}$ ]]
printf 'changed\n' > required
second=$(release_input_fingerprint)
[[ $first != "$second" ]]
release_input_roots=(required missing)
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted a missing required input: $result" >&2; exit 1
fi
release_input_roots=(required)
FAIL_TEST_HASH=1
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted failed digest: $result" >&2; exit 1
fi
FAIL_TEST_HASH=0
find() {
  if [[ $* == *'! -type d'* ]]; then
    command find "$@"
    return 1
  fi
  command find "$@"
}
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted failed safety enumeration: $result" >&2; exit 1
fi
unset -f find
find() {
  if [[ $* == *'-type f -print0'* && $* != *'! -type d'* ]]; then
    command find "$@"
    return 1
  fi
  command find "$@"
}
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted failed file enumeration: $result" >&2; exit 1
fi
unset -f find
find() {
  if [[ $* == *'-type f -print0'* && $* != *'! -type d'* ]]; then
    command find "$@"
    rm -f -- "$path_list"
    return 0
  fi
  command find "$@"
}
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted a missing path list: $result" >&2; exit 1
fi
unset -f find
wc() { command wc "$@"; rm -f -- "$path_list"; }
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted a path list removed after sizing: $result" >&2; exit 1
fi
unset -f wc
sort() { cat >/dev/null; printf 'required'; }
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted an incomplete path list: $result" >&2; exit 1
fi
unset -f sort
sort() { command sort "$@"; return 1; }
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted failed sort: $result" >&2; exit 1
fi
unset -f sort
mktemp() { return 1; }
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted failed temporary file creation: $result" >&2; exit 1
fi
unset -f mktemp
release_input_roots=(required)
before_optional=$(release_input_fingerprint)
mkdir vendor
printf 'fixture\n' > vendor/modules.txt
after_optional=$(release_input_fingerprint)
if [[ $before_optional == "$after_optional" ]]; then
  echo "fingerprint ignored a newly created optional vendor directory" >&2; exit 1
fi
rm -rf -- vendor
ln -s absent go.sum
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted dangling optional symlink: $result" >&2; exit 1
fi
rm -- go.sum
ln -s required link
release_input_roots=(required link)
if result=$(release_input_fingerprint 2>/dev/null); then
  echo "fingerprint accepted a symlink: $result" >&2; exit 1
fi
release_input_roots=(required)
if result=$(release_input_fingerprint "$test_root/input" 2>/dev/null); then
  echo "fingerprint accepted failed diagnostic write: $result" >&2; exit 1
fi
printf '%s\n' 'test-release-input-fingerprint: OK'
