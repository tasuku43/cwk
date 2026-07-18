#!/usr/bin/env bash
# Prove that the canonical gate neutralizes hostile ambient Go configuration
# before its first Go-backed public check.
set -euo pipefail
cd "$(dirname "$0")/.."

fixture_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-go-environment.XXXXXXXX")
cleanup() { rm -rf -- "$fixture_root"; }
trap cleanup EXIT
cp scripts/testdata/fake-go-gate-environment.sh "$fixture_root/go"
chmod 0700 "$fixture_root/go"

status=0
output=$(env \
  PATH="$fixture_root:$PATH" \
  GO111MODULE=off \
  GOENV=/definitely/missing/go.env \
  GOEXPERIMENT=definitely-invalid \
  GOFIPS140=definitely-invalid \
  GOFLAGS=-run=NoTests \
  GOTOOLCHAIN=definitely-invalid \
  GOWORK=/definitely/missing/go.work \
  scripts/check.sh public 2>&1) || status=$?

if [[ $status -ne 73 || $output != *"canonical gate reached Go with a sanitized environment"* ]]; then
  echo "canonical gate did not sanitize ambient Go configuration before its first Go check" >&2
  printf '%s\n' "$output" >&2
  exit 1
fi

echo "test-check-environment: OK"
