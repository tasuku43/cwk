#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."

report=$(mktemp)
cleanup() { rm -f -- "$report"; }
trap cleanup EXIT

if ./scripts/check.sh fast >"$report" 2>&1; then
  exit 0
fi

cat "$report" >&2
printf '%s\n' '{"continue":false,"stopReason":"The canonical fast repository gate failed.","systemMessage":"Run ./scripts/check.sh fast, fix every reported failure, and rerun it before stopping."}'
