#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
workflow=.github/workflows/live-chatwork.yml
scripts/lint-live-chatwork-workflow.sh "$workflow"
test_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-live-workflow.XXXXXXXX")
trap 'rm -rf -- "$test_root"' EXIT
reject() {
  local label=$1 expression=$2
  sed "$expression" "$workflow" > "$test_root/workflow.yml"
  if cmp -s "$workflow" "$test_root/workflow.yml"; then
    echo "test did not mutate workflow: $label" >&2; exit 1
  fi
  if scripts/lint-live-chatwork-workflow.sh "$test_root/workflow.yml" >/dev/null 2>&1; then
    echo "workflow contract accepted: $label" >&2; exit 1
  fi
}
reject 'PR trigger' 's/schedule:/pull_request_target:/'
reject 'missing opt-in' "s/ && vars.CWK_LIVE_SMOKE_ENABLED == 'true'//"
reject 'missing schedule restriction' "s/github.event_name == 'schedule' \&\& //"
reject 'write permission' 's/contents: read/contents: write/'
reject 'checkout credential persistence' 's/persist-credentials: false/persist-credentials: true/'
reject 'unpinned action' 's/actions\/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1/actions\/checkout@v7/'
reject 'provider output artifact' 's@run: ./scripts/live-chatwork-smoke.sh .*@run: cat /tmp/provider-output@'
awk '{ print; if ($0 == "on:") print "  workflow_dispatch:" }' "$workflow" > "$test_root/workflow.yml"
if scripts/lint-live-chatwork-workflow.sh "$test_root/workflow.yml" >/dev/null 2>&1; then
  echo 'workflow contract accepted manual dispatch' >&2; exit 1
fi
cp "$workflow" "$test_root/workflow.yml"
printf '%s\n' '      - run: echo injected' >> "$test_root/workflow.yml"
if scripts/lint-live-chatwork-workflow.sh "$test_root/workflow.yml" >/dev/null 2>&1; then
  echo 'workflow contract accepted an extra step' >&2; exit 1
fi
printf '%s\n' 'test-live-chatwork-workflow: OK'
