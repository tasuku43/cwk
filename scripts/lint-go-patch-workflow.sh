#!/usr/bin/env bash
# Keep Go patch proposal automation bound to the validated generated change.
set -euo pipefail
cd "$(dirname "$0")/.."

workflow=${1:-.github/workflows/go-patch.yml}
[[ -f $workflow && ! -L $workflow ]] || {
  echo "Go patch workflow must be a regular file: $workflow" >&2
  exit 1
}

fail() {
  echo "Go patch workflow contract failed: $*" >&2
  exit 1
}

require_line() {
  grep -Fqx -- "$2" "$workflow" || fail "$1"
}

extract_job() {
  local job=$1
  awk -v start="  ${job}:" '
    $0 == start { active = 1 }
    active && /^  [A-Za-z0-9_-]+:$/ && $0 != start { exit }
    active { print }
  ' "$workflow"
}

extract_job_keys() {
  local job=$1
  extract_job "$job" | awk '
    /^    [A-Za-z0-9_-]+:/ {
      key = $0
      sub(/^    /, "", key)
      sub(/:.*/, "", key)
      print key
    }
  '
}

extract_job_output_values() {
  local job=$1
  extract_job "$job" | awk '
    /^    outputs:/ { in_outputs = 1; next }
    in_outputs && /^    [A-Za-z0-9_-]+:/ { in_outputs = 0 }
    in_outputs && /^      [A-Za-z0-9_-]+:/ {
      value = $0
      sub(/^      /, "", value)
      print value
    }
  '
}

extract_permission_values() {
  local job=$1
  extract_job "$job" | awk '
    /^    permissions:/ { in_permissions = 1; next }
    in_permissions && /^    [A-Za-z0-9_-]+:/ { in_permissions = 0 }
    in_permissions && /^      [A-Za-z0-9_-]+:/ {
      value = $0
      sub(/^      /, "", value)
      print value
    }
  '
}

extract_step_field_sequences() {
  local job=$1
  extract_job "$job" | awk '
    function emit() {
      if (in_step) print fields
    }
    /^      -([[:space:]]|$)/ {
      emit()
      in_step = 1
      fields = $0
      sub(/^      -[[:space:]]*/, "", fields)
      if (fields ~ /^[A-Za-z0-9_-]+:/) sub(/:.*/, "", fields)
      else fields = "invalid-step-start"
      next
    }
    in_step && /^        [A-Za-z0-9_-]+:/ {
      key = $0
      sub(/^        /, "", key)
      sub(/:.*/, "", key)
      fields = fields "," key
    }
    END { emit() }
  '
}

extract_step_mapping_values() {
  local job=$1 mapping=$2
  extract_job "$job" | awk -v mapping="$mapping" '
    function emit() {
      if (in_step) print "--"
    }
    /^      -([[:space:]]|$)/ {
      emit()
      in_step = 1
      in_mapping = 0
      next
    }
    in_step && $0 == "        " mapping ":" { in_mapping = 1; next }
    in_step && /^        [A-Za-z0-9_-]+:/ { in_mapping = 0 }
    in_step && in_mapping && /^          / {
      value = $0
      sub(/^          /, "", value)
      print value
    }
    END { emit() }
  '
}

extract_named_run() {
  local job=$1 step_name=$2
  extract_job "$job" | awk -v wanted="$step_name" '
    /^      - / {
      in_step = ($0 == "      - name: " wanted)
      in_run = 0
      next
    }
    in_step && /^        run: \|$/ { in_run = 1; next }
    in_step && in_run && /^          / {
      sub(/^          /, "")
      print
      next
    }
    in_step && in_run && /^$/ { print ""; next }
    in_step && in_run { exit }
  '
}

require_block_line() {
  local block=$1 label=$2 line=$3
  grep -Fqx -- "$line" <<<"$block" || fail "$label"
}

root_header=$(sed -n '1,15p' "$workflow")
expected_header=$'name: Go patch update\n\non:\n  schedule:\n    - cron: "17 6 * * 1"\n  workflow_dispatch:\n\npermissions:\n  contents: read\n\nconcurrency:\n  group: go-patch-${{ github.workflow }}\n  cancel-in-progress: false\n\njobs:'
[[ $root_header == "$expected_header" ]] || fail "weekly/manual triggers, read-only default permissions, and non-canceling concurrency must match the reviewed contract"

root_keys=$(awk '/^[A-Za-z0-9_-]+:/ { key = $0; sub(/:.*/, "", key); print key }' "$workflow")
[[ $root_keys == $'name\non\npermissions\nconcurrency\njobs' ]] || fail "root-level env/defaults/runtime injection and extra workflow keys are forbidden"

if grep -Eq '^  (push|pull_request|pull_request_target|workflow_run):' "$workflow"; then
  fail "workflow must not execute on push, pull_request, pull_request_target, or workflow_run"
fi
if grep -Eq 'github\.event\.pull_request|github\.head_ref|pull_request_target|actions: write|auto[-_]merge|gh pr merge|pulls/.*/merge' "$workflow"; then
  fail "workflow must not run PR-head code, dispatch privileged PR events, grant actions write, or merge automatically"
fi

for required in \
  '  prepare:' \
  '  validate:' \
  '  propose:' \
  "    if: github.ref == format('refs/heads/{0}', github.event.repository.default_branch)" \
  '    needs: prepare' \
  "    if: needs.prepare.outputs.changed == 'true'" \
  '    runs-on: ubuntu-latest' \
  '        id: patch' \
  "      base_sha: \${{ steps.patch.outputs.base_sha }}" \
  "          ref: \${{ github.sha }}" \
  "          ref: \${{ needs.prepare.outputs.base_sha }}" \
  "          TRIGGER_REF: \${{ github.ref }}" \
  "          BASE_SHA: \${{ github.sha }}" \
  "          DEFAULT_BRANCH: \${{ github.event.repository.default_branch }}" \
  "          BASE_SHA: \${{ needs.prepare.outputs.base_sha }}" \
  "          CANDIDATE: \${{ needs.prepare.outputs.candidate }}" \
  "          PATCH_MINOR: \${{ needs.prepare.outputs.minor }}" \
  "          [[ \"\$TRIGGER_REF\" == \"refs/heads/\$DEFAULT_BRANCH\" ]]" \
  '          persist-credentials: false' \
  '          go-version-file: go.mod' \
  "          go-version: \${{ needs.prepare.outputs.candidate }}" \
  "          [[ \"\$(git rev-parse HEAD)\" == \"\$BASE_SHA\" ]]" \
  "          [[ \"\$(git status --porcelain=v1 --untracked-files=all)\" == \" M go.mod\" ]]" \
  "          [[ \"\$(git diff --numstat -- go.mod)\" == \"\$(printf '1\\t1\\tgo.mod')\" ]]" \
  "          go run ./tools/gopatch apply \"\$CANDIDATE\"" \
  '          ./scripts/check.sh full' \
  '          ./scripts/check.sh security' \
  '          ./scripts/check.sh release' \
  '      contents: read' \
  '      contents: write' \
  '      pull-requests: write' \
  '          add-paths: go.mod' \
  "          base: \${{ github.event.repository.default_branch }}" \
  "          branch: codex/go-patch-\${{ needs.prepare.outputs.minor }}" \
  '          delete-branch: false' \
  "          token: \${{ github.token }}" \
  "          title: \"Go \${{ needs.prepare.outputs.candidate }} へ更新\"" \
  "            変更対象は \`go.mod\` の \`go\` 指令だけです。自動mergeは行いません。" \
  "            \`GITHUB_TOKEN\` が作成したpull requestのCIは、GitHubの設定により実行承認が必要な場合があります。このworkflowの事前検証はPRに紐づくCI checkを偽装しません。"; do
  require_line "missing or changed reviewed workflow line: $required" "$required"
done

prepare=$(extract_job prepare)
validate=$(extract_job validate)
propose=$(extract_job propose)
[[ -n $prepare && -n $validate && -n $propose ]] || fail "prepare, validate, and propose jobs must all exist"
job_ids=$(awk '/^jobs:$/ { in_jobs = 1; next } in_jobs && /^  [A-Za-z0-9_-]+:$/ { key = $0; sub(/^  /, "", key); sub(/:.*/, "", key); print key }' "$workflow")
[[ $job_ids == $'prepare\nvalidate\npropose' ]] || fail "only the reviewed prepare, validate, and propose jobs are allowed"
[[ $(extract_job_keys prepare) == $'name\nif\nruns-on\npermissions\noutputs\nsteps' ]] || fail "prepare job fields must match the reviewed allowlist"
[[ $(extract_job_keys validate) == $'name\nneeds\nif\nruns-on\npermissions\nsteps' ]] || fail "validate job fields must match the reviewed allowlist"
[[ $(extract_job_keys propose) == $'name\nneeds\nif\nruns-on\npermissions\nsteps' ]] || fail "proposal job fields must match the reviewed allowlist"
[[ $(extract_job_output_values prepare) == $'changed: ${{ steps.patch.outputs.changed }}\ncandidate: ${{ steps.patch.outputs.candidate }}\nminor: ${{ steps.patch.outputs.minor }}\nbase_sha: ${{ steps.patch.outputs.base_sha }}' ]] || fail "prepare job outputs must bind only to the patch step"
[[ $(grep -F -c '    runs-on: ubuntu-latest' "$workflow" || true) == 3 ]] || fail "all three jobs must use the reviewed runner"
[[ $(grep -F -x -c "    if: needs.prepare.outputs.changed == 'true'" "$workflow" || true) == 1 ]] || fail "candidate validation must run only when a patch is available"
[[ $(extract_permission_values prepare) == 'contents: read' ]] || fail "prepare job permissions must be exactly Contents read"
[[ $(extract_permission_values validate) == 'contents: read' ]] || fail "validate job permissions must be exactly Contents read"
[[ $(extract_permission_values propose) == $'contents: write\npull-requests: write' ]] || fail "proposal job permissions must be exactly Contents and Pull requests write"
[[ $(extract_step_field_sequences prepare) == $'uses,with\nuses,with\nname,id,env,run' ]] || fail "prepare steps or their field shapes changed"
[[ $(extract_step_field_sequences validate) == $'uses,with\nuses,with\nname,env,run\nname,run' ]] || fail "validate steps or their field shapes changed"
[[ $(extract_step_field_sequences propose) == $'uses,with\nuses,with\nname,env,run\nname,uses,with' ]] || fail "proposal steps or their field shapes changed"
expected_prepare_with=$'ref: ${{ github.sha }}\npersist-credentials: false\n--\ngo-version-file: go.mod\ncache: true\n--\n--'
expected_validate_with=$'ref: ${{ needs.prepare.outputs.base_sha }}\npersist-credentials: false\n--\ngo-version: ${{ needs.prepare.outputs.candidate }}\ncache: true\n--\n--\n--'
expected_propose_with=$'ref: ${{ needs.prepare.outputs.base_sha }}\npersist-credentials: false\n--\ngo-version: ${{ needs.prepare.outputs.candidate }}\ncache: false\n--\n--\ntoken: ${{ github.token }}\nadd-paths: go.mod\nbase: ${{ github.event.repository.default_branch }}\nbranch: codex/go-patch-${{ needs.prepare.outputs.minor }}\ndelete-branch: false\ncommit-message: "Go ${{ needs.prepare.outputs.candidate }} へ更新"\ntitle: "Go ${{ needs.prepare.outputs.candidate }} へ更新"\nbody: |\n  公式Goリリース一覧から同一minorの安定patchを検出しました。\n  既定branchの固定SHA上で候補を再構成し、候補の正確なGo toolchainで full・security・release gate を通過してから作成しています。\n  変更対象は `go.mod` の `go` 指令だけです。自動mergeは行いません。\n  `GITHUB_TOKEN` が作成したpull requestのCIは、GitHubの設定により実行承認が必要な場合があります。このworkflowの事前検証はPRに紐づくCI checkを偽装しません。\n--'
[[ $(extract_step_mapping_values prepare with) == "$expected_prepare_with" ]] || fail "prepare action inputs or values changed"
[[ $(extract_step_mapping_values validate with) == "$expected_validate_with" ]] || fail "validation action inputs or values changed"
[[ $(extract_step_mapping_values propose with) == "$expected_propose_with" ]] || fail "proposal action inputs, values, or body changed"
expected_prepare_env=$'--\n--\nBASE_SHA: ${{ github.sha }}\nDEFAULT_BRANCH: ${{ github.event.repository.default_branch }}\nTRIGGER_REF: ${{ github.ref }}\n--'
expected_validate_env=$'--\n--\nBASE_SHA: ${{ needs.prepare.outputs.base_sha }}\nCANDIDATE: ${{ needs.prepare.outputs.candidate }}\n--\n--'
expected_propose_env=$'--\n--\nBASE_SHA: ${{ needs.prepare.outputs.base_sha }}\nCANDIDATE: ${{ needs.prepare.outputs.candidate }}\nPATCH_MINOR: ${{ needs.prepare.outputs.minor }}\n--\n--'
[[ $(extract_step_mapping_values prepare env) == "$expected_prepare_env" ]] || fail "prepare environment inputs or values changed"
[[ $(extract_step_mapping_values validate env) == "$expected_validate_env" ]] || fail "validation environment inputs or values changed"
[[ $(extract_step_mapping_values propose env) == "$expected_propose_env" ]] || fail "proposal environment inputs or values changed"
require_block_line "$prepare" "prepare job must be read-only" '      contents: read'
require_block_line "$prepare" "prepare must checkout the immutable workflow SHA" "          ref: \${{ github.sha }}"
require_block_line "$validate" "validate job must be read-only" '      contents: read'
require_block_line "$validate" "validate must checkout the prepared immutable SHA" "          ref: \${{ needs.prepare.outputs.base_sha }}"
require_block_line "$validate" "validate job must reconstruct the validated candidate" "          go run ./tools/gopatch apply \"\$CANDIDATE\""
require_block_line "$validate" "validate job must run the full gate" '          ./scripts/check.sh full'
require_block_line "$validate" "validate job must run the security gate" '          ./scripts/check.sh security'
require_block_line "$validate" "validate job must run the release gate" '          ./scripts/check.sh release'
require_block_line "$propose" "proposal job must depend on successful candidate validation" '    needs: [prepare, validate]'
require_block_line "$propose" "proposal job must run only after validation succeeds" "    if: needs.prepare.outputs.changed == 'true' && needs.validate.result == 'success'"
require_block_line "$propose" "proposal job may not receive an App/PAT secret" "          token: \${{ github.token }}"
require_block_line "$propose" "proposal must checkout the prepared immutable SHA" "          ref: \${{ needs.prepare.outputs.base_sha }}"
require_block_line "$propose" "proposal must reapply only the validated candidate" "          go run ./tools/gopatch apply \"\$CANDIDATE\""
require_block_line "$propose" "proposal must stage only go.mod" '          add-paths: go.mod'

if grep -Eq 'contents: write|pull-requests: write|secrets\.' <<<"$prepare$validate"; then
  fail "candidate preparation and validation must not receive write permissions or secrets"
fi
[[ $(grep -F -c '      contents: read' "$workflow" || true) == 2 ]] || fail "exactly the prepare and validate jobs must receive Contents read"
[[ $(grep -F -c '      contents: write' "$workflow" || true) == 1 ]] || fail "only the proposal job may receive Contents write"
[[ $(grep -F -c '      pull-requests: write' "$workflow" || true) == 1 ]] || fail "only the proposal job may receive Pull requests write"
[[ $(grep -F -c '      actions: write' "$workflow" || true) == 0 ]] || fail "workflow dispatch permission is forbidden"

[[ $(grep -F -c 'actions/checkout@' "$workflow" || true) == 3 ]] || fail "workflow must have exactly three checkout steps"
[[ $(grep -F -c 'actions/setup-go@' "$workflow" || true) == 3 ]] || fail "workflow must have exactly three Go setup steps"
[[ $(grep -F -c "          ref: \${{ github.sha }}" "$workflow" || true) == 1 ]] || fail "only candidate preparation may checkout the event SHA"
[[ $(grep -F -c "          ref: \${{ needs.prepare.outputs.base_sha }}" "$workflow" || true) == 2 ]] || fail "validation and proposal must checkout only the prepared SHA"
[[ $(grep -F -c "          go-version: \${{ needs.prepare.outputs.candidate }}" "$workflow" || true) == 2 ]] || fail "validation and proposal must use the exact candidate toolchain"
[[ $(grep -F -c 'persist-credentials: false' "$workflow" || true) == 3 ]] || fail "all three checkouts must disable persisted credentials"
action_lines=$(grep -E '^[[:space:]]+(-[[:space:]]+)?uses:' "$workflow" | sed -E 's/^[[:space:]]*(-[[:space:]]+)?//')
expected_action_lines=$'uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7\nuses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0\nuses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7\nuses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0\nuses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7\nuses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0\nuses: peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8'
[[ $action_lines == "$expected_action_lines" ]] || fail "all action names and immutable revisions must match the reviewed sequence exactly"
[[ $(grep -F -c 'add-paths:' "$workflow" || true) == 1 ]] || fail "the proposal must stage exactly one path"
[[ $(grep -F -c "token: \${{ github.token }}" "$workflow" || true) == 1 ]] || fail "the write token must be passed only to the pull-request action"
[[ $(grep -F -c '        env:' "$workflow" || true) == 3 ]] || fail "only the three reviewed run steps may declare environment inputs"
[[ $(grep -E -c '^          [A-Z][A-Z0-9_]*:' "$workflow" || true) == 8 ]] || fail "step environment keys must match the reviewed allowlist"
[[ $(grep -F -c '        with:' "$workflow" || true) == 7 ]] || fail "only the seven reviewed action steps may declare with inputs"
[[ $(grep -E -c '^          [a-z][A-Za-z0-9_-]*:' "$workflow" || true) == 20 ]] || fail "with input keys must match the reviewed allowlist"

while IFS= read -r action_line; do
  [[ $action_line =~ uses:[[:space:]][^@]+@([0-9a-f]{40})[[:space:]]# ]] || fail "every external action must use a pinned 40-character revision: $action_line"
done < <(grep -E '^[[:space:]]+(-[[:space:]]+)?uses:' "$workflow")

expected_prepare_run=$(cat <<'EOF'
set -euo pipefail
[[ "$TRIGGER_REF" == "refs/heads/$DEFAULT_BRANCH" ]]
[[ "$(git rev-parse HEAD)" == "$BASE_SHA" ]]
candidate="$(go run ./tools/gopatch)"
printf 'base_sha=%s\n' "$BASE_SHA" >>"$GITHUB_OUTPUT"
if [[ -z "$candidate" ]]; then
  [[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]]
  printf 'changed=false\n' >>"$GITHUB_OUTPUT"
  exit 0
fi
[[ "$candidate" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
[[ "$(git status --porcelain=v1 --untracked-files=all)" == " M go.mod" ]]
[[ "$(git diff --numstat -- go.mod)" == "$(printf '1\t1\tgo.mod')" ]]
[[ "$(go run ./tools/gopatch version)" == "$candidate" ]]
minor="$(go run ./tools/gopatch minor)"
[[ "$minor" =~ ^[0-9]+\.[0-9]+$ ]]
printf 'candidate=%s\nminor=%s\nchanged=true\n' "$candidate" "$minor" >>"$GITHUB_OUTPUT"
EOF
)
expected_validate_run=$(cat <<'EOF'
set -euo pipefail
[[ "$BASE_SHA" =~ ^[0-9a-f]{40}$ ]]
[[ "$CANDIDATE" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
[[ "$(git rev-parse HEAD)" == "$BASE_SHA" ]]
go run ./tools/gopatch apply "$CANDIDATE"
[[ "$(git status --porcelain=v1 --untracked-files=all)" == " M go.mod" ]]
[[ "$(git diff --numstat -- go.mod)" == "$(printf '1\t1\tgo.mod')" ]]
[[ "$(go run ./tools/gopatch version)" == "$CANDIDATE" ]]
EOF
)
expected_gates_run=$(cat <<'EOF'
./scripts/check.sh full
./scripts/check.sh security
./scripts/check.sh release
EOF
)
expected_propose_run=$(cat <<'EOF'
set -euo pipefail
[[ "$BASE_SHA" =~ ^[0-9a-f]{40}$ ]]
[[ "$CANDIDATE" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
[[ "$PATCH_MINOR" =~ ^[0-9]+\.[0-9]+$ ]]
[[ "$(git rev-parse HEAD)" == "$BASE_SHA" ]]
[[ "$(go run ./tools/gopatch minor)" == "$PATCH_MINOR" ]]
go run ./tools/gopatch apply "$CANDIDATE"
[[ "$(git status --porcelain=v1 --untracked-files=all)" == " M go.mod" ]]
[[ "$(git diff --numstat -- go.mod)" == "$(printf '1\t1\tgo.mod')" ]]
[[ "$(go run ./tools/gopatch version)" == "$CANDIDATE" ]]
EOF
)
[[ $(extract_named_run prepare '固定した既定branch SHAと候補差分を確認') == "$expected_prepare_run" ]] || fail "prepare shell body changed; review its exact read-only candidate semantics"
[[ $(extract_named_run validate '固定SHAへ候補だけを再構成') == "$expected_validate_run" ]] || fail "validate shell body changed; review its exact candidate reconstruction"
[[ $(extract_named_run validate '必須gateを候補toolchainで実行') == "$expected_gates_run" ]] || fail "candidate validation must run exactly full, security, and release"
[[ $(extract_named_run propose 'SHAとgo.modだけの候補差分を再検証') == "$expected_propose_run" ]] || fail "proposal shell body changed; review its exact write-boundary revalidation"

validate_step=$(grep -n -F '      - name: 必須gateを候補toolchainで実行' "$workflow" | cut -d: -f1)
proposal_step=$(grep -n -F '      - name: 検証済みgo.modだけでpull requestを作成・再利用' "$workflow" | cut -d: -f1)
candidate_recheck=$(grep -n -F '      - name: SHAとgo.modだけの候補差分を再検証' "$workflow" | cut -d: -f1)
[[ -n $candidate_recheck && -n $validate_step && -n $proposal_step ]] || fail "candidate validation, proposal recheck, and PR creation steps are required"
((candidate_recheck < proposal_step)) || fail "proposal revalidation must precede PR write"
((validate_step < proposal_step)) || fail "full/security/release validation must precede PR write"

if grep -Eq 'pulls/[0-9]+/merge|enable-auto-merge|auto-merge' "$workflow"; then
  fail "automatic merge is forbidden"
fi

echo "Go patch workflow contract passed"
