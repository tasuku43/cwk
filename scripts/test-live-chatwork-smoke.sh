#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-smoke-test.XXXXXXXX")
trap 'rm -rf -- "$root"' EXIT
mkdir "$root/tmp" "$root/user-config"
cat > "$root/fake-cwk" <<'FAKE'
#!/bin/sh
set -eu
case "$(ls -ld "$XDG_CONFIG_HOME")" in drwx------*) ;; *) exit 3 ;; esac
case "$(ls -ld "$XDG_CONFIG_HOME/cwk")" in drwx------*) ;; *) exit 3 ;; esac
case "$(ls -l "$XDG_CONFIG_HOME/cwk/command-selection.json")" in -rw-------*) ;; *) exit 3 ;; esac
printf '%s\n' "$*" >> "$SMOKE_TEST_LOG"
printf '%s\n' "$XDG_CONFIG_HOME" >> "$SMOKE_TEST_ROOTS"
cat "$XDG_CONFIG_HOME/cwk/command-selection.json" > "$SMOKE_TEST_CONFIG"
printf 'PRIVATE_PROVIDER_OUTPUT %s\n' "$CWK_API_TOKEN"
printf 'PRIVATE_PROVIDER_ERROR %s\n' "$CWK_API_TOKEN" >&2
if [ "${SMOKE_TEST_FAIL:-}" = "$*" ]; then exit 1; fi
FAKE
chmod +x "$root/fake-cwk"
export TMPDIR="$root/tmp" XDG_CONFIG_HOME="$root/user-config"
export SMOKE_TEST_LOG="$root/calls" SMOKE_TEST_ROOTS="$root/roots" SMOKE_TEST_CONFIG="$root/config"
export CWK_API_TOKEN='synthetic-secret-canary'
output=$(scripts/live-chatwork-smoke.sh "$root/fake-cwk" 2>&1)
expected=$(printf 'account show\naccount status\nrooms list')
[[ $(cat "$root/calls") == "$expected" ]]
[[ $(cat "$root/config") == '{"schema_version":1,"enabled_commands":["account show","account status","rooms list"]}' ]]
[[ "$output" != *PRIVATE_PROVIDER* && "$output" != *synthetic-secret-canary* ]]
while IFS= read -r used_root; do
  [[ "$used_root" != "$XDG_CONFIG_HOME" && ! -e "$used_root" ]]
done < "$root/roots"
[[ -z $(ls -A "$root/user-config") ]]
: > "$root/calls"
export SMOKE_TEST_FAIL='account status'
if output=$(scripts/live-chatwork-smoke.sh "$root/fake-cwk" 2>&1); then
  echo 'smoke test: failed task returned success' >&2; exit 1
fi
[[ $(cat "$root/calls") == "$(printf 'account show\naccount status')" ]]
[[ "$output" == *'account status failed'* && "$output" != *PRIVATE_PROVIDER* && "$output" != *synthetic-secret-canary* ]]
unset CWK_API_TOKEN
: > "$root/calls"
if scripts/live-chatwork-smoke.sh "$root/fake-cwk" >"$root/output" 2>&1; then
  echo 'smoke test: missing token returned success' >&2; exit 1
fi
[[ ! -s "$root/calls" ]]
printf '%s\n' 'test-live-chatwork-smoke: OK'
