#!/usr/bin/env bash
# Validate the release workflow's immutable publication and shared-tap boundaries.
# GitHub expressions and embedded shell snippets are intentionally matched literally.
# shellcheck disable=SC1003,SC2016
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo_root"
workflow=${1:-.github/workflows/release.yml}

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

extract_job() {
  local job_name=$1
  awk -v start="  ${job_name}:" '
    $0 == start { in_job=1 }
    in_job && $0 != start && /^  [A-Za-z0-9_-]+:/ { exit }
    in_job { print }
  ' "$workflow"
}

extract_named_step() {
  local job_text=$1
  local step_name=$2
  printf '%s\n' "$job_text" | awk -v start="      - name: ${step_name}" '
    $0 == start { in_step=1 }
    in_step && seen && /^      - (name:|uses:)/ { exit }
    in_step { print; seen=1 }
  '
}

require_exact_line() {
  local text=$1
  local expected=$2
  local context=$3
  local count
  count=$(printf '%s\n' "$text" | grep -cFx -- "$expected" || true)
  if [[ $count -ne 1 ]]; then
    fail "$context must contain exactly one reviewed line: $expected"
  fi
}

require_line_count() {
  local text=$1
  local expected=$2
  local context=$3
  local actual
  actual=$(printf '%s\n' "$text" | wc -l | tr -d ' ')
  if [[ $actual -ne $expected ]]; then
    fail "$context has unreviewed fields or commands: expected $expected lines, found $actual"
  fi
}

require_with_keys() {
  local text=$1
  local expected=$2
  local context=$3
  local actual
  actual=$(printf '%s\n' "$text" | awk '
    $0 == "        with:" { in_with=1; next }
    in_with && /^          [A-Za-z0-9_-]+:/ {
      key=$0
      sub(/^          /, "", key)
      sub(/:.*/, "", key)
      print key
    }
  ')
  if [[ $actual != "$expected" ]]; then
    printf '%s\n' "$context has unreviewed with keys" >&2
    diff -u <(printf '%s\n' "$expected") <(printf '%s\n' "$actual") >&2 || true
    exit 1
  fi
}

[[ -f $workflow ]] || fail "release workflow does not exist: $workflow"

workflow_root_lines=$(awk '/^[^[:space:]#]/ { print }' "$workflow")
expected_workflow_root_lines=$'name: Release\non:\npermissions:\nconcurrency:\njobs:'
if [[ $workflow_root_lines != "$expected_workflow_root_lines" ]]; then
  printf '%s\n' "release workflow top-level fields must match the reviewed allowlist" >&2
  diff -u \
    <(printf '%s\n' "$expected_workflow_root_lines") \
    <(printf '%s\n' "$workflow_root_lines") >&2 || true
  exit 1
fi

workflow_triggers=$(awk '
  $0 == "on:" { in_triggers=1 }
  in_triggers && seen && /^[^[:space:]#]/ { exit }
  in_triggers { print; seen=1 }
' "$workflow")
expected_workflow_triggers=$'on:\n  push:\n    tags:\n      - "v*"\n  workflow_dispatch:\n    inputs:\n      tag:\n        description: "既存のstable ReleaseからHomebrew Formula公開を再開するtag"\n        required: true\n        type: string'
if [[ $workflow_triggers != "$expected_workflow_triggers" ]]; then
  printf '%s\n' "release workflow triggers must match the reviewed tag-push and Formula-recovery allowlist" >&2
  diff -u \
    <(printf '%s\n' "$expected_workflow_triggers") \
    <(printf '%s\n' "$workflow_triggers") >&2 || true
  exit 1
fi

workflow_permissions=$(awk '
  $0 == "permissions:" { in_permissions=1 }
  in_permissions && seen && /^[^[:space:]#]/ { exit }
  in_permissions { print; seen=1 }
' "$workflow")
expected_workflow_permissions=$'permissions:\n  contents: read'
if [[ $workflow_permissions != "$expected_workflow_permissions" ]]; then
  printf '%s\n' "release workflow root permissions must be exactly contents: read" >&2
  diff -u \
    <(printf '%s\n' "$expected_workflow_permissions") \
    <(printf '%s\n' "$workflow_permissions") >&2 || true
  exit 1
fi

if grep -qF -- '--clobber' "$workflow"; then
  fail "release workflow must never overwrite existing release assets"
fi
if grep -qF -- '--generate-notes' "$workflow"; then
  fail "release workflow must not replace reviewed annotated-tag notes with generated notes"
fi
grep -qF -- '--notes-from-tag' "$workflow" ||
  fail "release workflow must publish the reviewed annotated-tag notes"
grep -qF 'already exists; refusing to replace immutable release assets' "$workflow" ||
  fail "release workflow does not fail closed when the tag already has a release"
grep -qF "github.event_name == 'workflow_dispatch'" "$workflow" ||
  fail "release workflow is missing the reviewed Formula recovery trigger"

for required in \
  './scripts/check.sh full' './scripts/package-release.sh' 'checksums.txt' \
  'gh release create' 'Formula/' 'scripts/render-formula.sh' \
  'repository: tasuku43/homebrew-tap'; do
  grep -qF "$required" "$workflow" || fail "release workflow is missing: $required"
done

formula_job=$(extract_job formula)
formula_publish_job=$(extract_job formula_publish)
build_job=$(extract_job build)
publish_job=$(extract_job publish)
recovery_job=$(extract_job recover_formula_input)
preflight_job=$(extract_job preflight)
[[ -n $formula_job ]] || fail "release workflow is missing the Formula job"
[[ -n $formula_publish_job ]] || fail "release workflow is missing the Formula publish job"
[[ -n $build_job ]] || fail "release workflow is missing the build job"
[[ -n $publish_job ]] || fail "release workflow is missing the GitHub Release publish job"
[[ -n $recovery_job ]] || fail "release workflow is missing the existing-Release Formula recovery job"
[[ -n $preflight_job ]] || fail "release workflow is missing the release preflight job"

formula_header=$(printf '%s\n' "$formula_job" | awk '
  { print }
  /^    steps:$/ { exit }
')
expected_formula_header=$'  formula:\n    name: checksum固定済みHomebrew Formulaを生成・audit\n    if: >-\n      always() &&\n      needs.preflight.result == \'success\' &&\n      needs.preflight.outputs.stable == \'true\' &&\n      ((github.event_name == \'push\' && needs.publish.result == \'success\') ||\n      (github.event_name == \'workflow_dispatch\' && needs.recover_formula_input.result == \'success\'))\n    needs: [preflight, publish, recover_formula_input]\n    runs-on: macos-15\n    permissions:\n      contents: read\n    steps:'
if [[ $formula_header != "$expected_formula_header" ]]; then
  printf '%s\n' "Formula job header must match the reviewed stable-only fail-closed contract" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_header") \
    <(printf '%s\n' "$formula_header") >&2 || true
  exit 1
fi
formula_job_fields=$(printf '%s\n' "$formula_job" | awk '/^    [^[:space:]#]/ { print }')
expected_formula_job_fields=$'    name: checksum固定済みHomebrew Formulaを生成・audit\n    if: >-\n    needs: [preflight, publish, recover_formula_input]\n    runs-on: macos-15\n    permissions:\n    steps:'
if [[ $formula_job_fields != "$expected_formula_job_fields" ]]; then
  printf '%s\n' "Formula job fields must match the reviewed allowlist" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_job_fields") \
    <(printf '%s\n' "$formula_job_fields") >&2 || true
  exit 1
fi

formula_publish_header=$(printf '%s\n' "$formula_publish_job" | awk '
  { print }
  /^    steps:$/ { exit }
')
expected_formula_publish_header=$'  formula_publish:\n    name: audit済みHomebrew Formulaを共有tapへ提案\n    if: >-\n      always() &&\n      needs.preflight.result == \'success\' &&\n      needs.preflight.outputs.stable == \'true\' &&\n      needs.formula.result == \'success\'\n    needs: [preflight, formula]\n    runs-on: ubuntu-latest\n    permissions: {}\n    steps:'
if [[ $formula_publish_header != "$expected_formula_publish_header" ]]; then
  printf '%s\n' "Formula publish job must use a fresh runner after the audit job" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_publish_header") \
    <(printf '%s\n' "$formula_publish_header") >&2 || true
  exit 1
fi
formula_publish_job_fields=$(printf '%s\n' "$formula_publish_job" | awk '/^    [^[:space:]#]/ { print }')
expected_formula_publish_job_fields=$'    name: audit済みHomebrew Formulaを共有tapへ提案\n    if: >-\n    needs: [preflight, formula]\n    runs-on: ubuntu-latest\n    permissions: {}\n    steps:'
if [[ $formula_publish_job_fields != "$expected_formula_publish_job_fields" ]]; then
  printf '%s\n' "Formula publish job fields must match the reviewed allowlist" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_publish_job_fields") \
    <(printf '%s\n' "$formula_publish_job_fields") >&2 || true
  exit 1
fi

release_revision_ref='          ref: ${{ needs.preflight.outputs.revision }}'
build_checkout=$(printf '%s\n' "$build_job" | awk '
  /uses: actions\/checkout@/ { in_step=1 }
  in_step && seen && /^      - (name:|uses:)/ { exit }
  in_step { print; seen=1 }
')
require_line_count "$build_checkout" 4 "matrix build checkout"
require_exact_line "$build_checkout" \
  '      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7' \
  "matrix build checkout"
require_exact_line "$build_checkout" "$release_revision_ref" "matrix build checkout"
require_exact_line "$build_checkout" '          persist-credentials: false' "matrix build checkout"
require_with_keys "$build_checkout" $'ref\npersist-credentials' "matrix build checkout"

for required in \
  '      tag: ${{ steps.release.outputs.tag }}' \
  '          ref: ${{ github.event_name == '\''workflow_dispatch'\'' && inputs.tag || github.ref }}' \
  '          RECOVERY_TAG: ${{ inputs.tag }}' \
  '            go run ./tools/releaseversion --stable "${tag}" >>"${GITHUB_OUTPUT}"' \
  '          revision=$(git rev-list -n 1 -- "${tag}")' \
  '          echo "tag=${tag}" >>"${GITHUB_OUTPUT}"'; do
  require_exact_line "$preflight_job" "$required" "release preflight recovery binding"
done

publish_checkout=$(extract_named_step "$publish_job" "正確なrelease tagをcheckout")
require_line_count "$publish_checkout" 6 "GitHub Release tag checkout"
for required in \
  '        uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7' \
  '        with:' \
  '          ref: ${{ needs.preflight.outputs.tag }}' \
  '          fetch-depth: 0' \
  '          persist-credentials: false'; do
  require_exact_line "$publish_checkout" "$required" "GitHub Release tag checkout"
done
require_with_keys "$publish_checkout" $'ref\nfetch-depth\npersist-credentials' "GitHub Release tag checkout"
publish_checkout_line=$(printf '%s\n' "$publish_job" | grep -n -m1 -F 'name: 正確なrelease tagをcheckout' | cut -d: -f1)
release_create_line=$(printf '%s\n' "$publish_job" | grep -n -m1 -F 'gh release create' | cut -d: -f1)
if ((publish_checkout_line >= release_create_line)); then
  fail "annotated release tag must be checked out before GitHub Release creation"
fi
for required in \
  '          tag="${{ needs.preflight.outputs.tag }}"' \
  '          args=(--verify-tag --title "${tag}" --notes-from-tag)' \
  '          gh release create "${tag}" dist/* "${args[@]}"'; do
  require_exact_line "$publish_job" "$required" "GitHub Release annotated-note publication"
done

recovery_header=$(printf '%s\n' "$recovery_job" | awk '
  { print }
  /^    steps:$/ { exit }
')
expected_recovery_header=$(printf '%s\n' \
  '  recover_formula_input:' \
  '    name: 既存ReleaseのFormula入力を検証' \
  "    if: github.event_name == 'workflow_dispatch' && needs.preflight.outputs.stable == 'true'" \
  '    needs: preflight' \
  '    runs-on: ubuntu-latest' \
  '    permissions:' \
  '      contents: read' \
  '    steps:')
if [[ $recovery_header != "$expected_recovery_header" ]]; then
  printf '%s\n' "Formula recovery job must be stable-only and Contents-read-only" >&2
  diff -u \
    <(printf '%s\n' "$expected_recovery_header") \
    <(printf '%s\n' "$recovery_header") >&2 || true
  exit 1
fi
recovery_steps=$(printf '%s\n' "$recovery_job" | grep -E '^      - ' || true)
expected_recovery_steps=$'      - name: 公開済み資産をdownload\n      - name: 公開済み資産とchecksumを検証\n      - name: Formula job用checksumをupload'
if [[ $recovery_steps != "$expected_recovery_steps" ]]; then
  printf '%s\n' "Formula recovery steps must match the reviewed read/verify/upload order" >&2
  diff -u \
    <(printf '%s\n' "$expected_recovery_steps") \
    <(printf '%s\n' "$recovery_steps") >&2 || true
  exit 1
fi
for required in \
  '          GH_TOKEN: ${{ github.token }}' \
  '          RELEASE_TAG: ${{ needs.preflight.outputs.tag }}' \
  '          metadata=$(gh release view "${RELEASE_TAG}" --json tagName,isDraft,isPrerelease --jq '\''[.tagName, .isDraft, .isPrerelease] | @tsv'\'')' \
  '          test "${metadata}" = "${expected_metadata}"' \
  '          gh release download "${RELEASE_TAG}" --dir dist' \
  '          test "${entries[*]}" = "${expected[*]}"' \
  '          test "$(wc -l < dist/checksums.txt)" -eq 5' \
  '          (cd dist && sha256sum --check checksums.txt)' \
  '          name: release-checksums' \
  '          path: dist/checksums.txt' \
  '          if-no-files-found: error'; do
  require_exact_line "$recovery_job" "$required" "existing-Release Formula recovery"
done
if printf '%s\n' "$recovery_job" | grep -Eq 'actions/checkout@|scripts/|secrets\.'; then
  fail "Formula recovery job must treat published assets as data without source execution or secrets"
fi

formula_permissions=$(printf '%s\n' "$formula_job" | awk '
  /^    permissions:$/ { in_permissions=1 }
  in_permissions { print }
  in_permissions && /^    steps:$/ { exit }
')
expected_formula_permissions=$'    permissions:\n      contents: read\n    steps:'
if [[ $formula_permissions != "$expected_formula_permissions" ]]; then
  printf '%s\n' "Formula job permissions must be exactly contents: read" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_permissions") \
    <(printf '%s\n' "$formula_permissions") >&2 || true
  exit 1
fi
formula_steps=$(printf '%s\n' "$formula_job" | grep -E '^      - ' || true)
expected_formula_steps=$'      - name: 正確なrelease sourceをcheckout\n      - name: checksumをdownload\n      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0\n      - name: Formulaをrenderしてaudit\n      - name: audit済みFormulaをupload'
if [[ $formula_steps != "$expected_formula_steps" ]]; then
  printf '%s\n' "Formula job steps must match the reviewed allowlist and order" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_steps") \
    <(printf '%s\n' "$formula_steps") >&2 || true
  exit 1
fi

formula_publish_steps=$(printf '%s\n' "$formula_publish_job" | grep -E '^      - ' || true)
expected_formula_publish_steps=$'      - name: audit済みFormulaをdownload\n      - name: Formula artifactをdataとして検証\n      - name: Homebrew tap用GitHub App tokenを作成\n      - name: 共有Homebrew tapをcheckout\n      - name: audit済みFormulaを共有tapへ配置\n      - name: Formula更新pull requestを作成'
if [[ $formula_publish_steps != "$expected_formula_publish_steps" ]]; then
  printf '%s\n' "Formula publish job steps must match the reviewed fresh-runner allowlist and order" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_publish_steps") \
    <(printf '%s\n' "$formula_publish_steps") >&2 || true
  exit 1
fi

source_checkout=$(extract_named_step "$formula_job" "正確なrelease sourceをcheckout")
require_line_count "$source_checkout" 5 "exact release source checkout"
require_exact_line "$source_checkout" \
  '        uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7' \
  "exact release source checkout"
require_exact_line "$source_checkout" "$release_revision_ref" "exact release source checkout"
require_exact_line "$source_checkout" '          persist-credentials: false' "exact release source checkout"
require_with_keys "$source_checkout" $'ref\npersist-credentials' "exact release source checkout"

checksum_step=$(extract_named_step "$formula_job" "checksumをdownload")
require_line_count "$checksum_step" 5 "Formula checksum download step"
for required in \
  '        uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8' \
  '        with:' \
  '          name: release-checksums' \
  '          path: dist'; do
  require_exact_line "$checksum_step" "$required" "Formula checksum download step"
done
require_with_keys "$checksum_step" $'name\npath' "Formula checksum download step"

setup_go_step=$(printf '%s\n' "$formula_job" | awk '
  $0 == "      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0" { in_step=1 }
  in_step && seen && /^      - / { exit }
  in_step { print; seen=1 }
')
require_line_count "$setup_go_step" 4 "Formula setup-go step"
for required in \
  '      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0' \
  '        with:' \
  '          go-version-file: go.mod' \
  '          cache: false'; do
  require_exact_line "$setup_go_step" "$required" "Formula setup-go step"
done
require_with_keys "$setup_go_step" $'go-version-file\ncache' "Formula setup-go step"

render_step=$(extract_named_step "$formula_job" "Formulaをrenderしてaudit")
require_line_count "$render_step" 14 "Formula render/audit step"
for required in \
  '        id: formula' \
  '        run: |' \
  '          formula_dir=$(mktemp -d "${RUNNER_TEMP}/formula.XXXXXXXX")' \
  '          binary=$(go run ./tools/projectmeta --field binary_name)' \
  '          formula="${formula_dir}/${binary}.rb"' \
  '          ./scripts/render-formula.sh \' \
  '            "${{ needs.preflight.outputs.tag }}" \' \
  '            "https://github.com/${GITHUB_REPOSITORY}" \' \
  '            dist/checksums.txt \' \
  '            "${formula}"' \
  '          ruby -c "${formula}"' \
  '          ./scripts/audit-formula.sh "${formula}"' \
  '          echo "path=${formula}" >>"${GITHUB_OUTPUT}"'; do
  require_exact_line "$render_step" "$required" "Formula render/audit step"
done

formula_upload_step=$(extract_named_step "$formula_job" "audit済みFormulaをupload")
require_line_count "$formula_upload_step" 7 "audited Formula upload step"
for required in \
  '        uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7' \
  '        with:' \
  '          name: audited-homebrew-formula' \
  '          path: ${{ steps.formula.outputs.path }}' \
  '          if-no-files-found: error' \
  '          retention-days: 1'; do
  require_exact_line "$formula_upload_step" "$required" "audited Formula upload step"
done
require_with_keys "$formula_upload_step" \
  $'name\npath\nif-no-files-found\nretention-days' \
  "audited Formula upload step"

formula_download_step=$(extract_named_step "$formula_publish_job" "audit済みFormulaをdownload")
require_line_count "$formula_download_step" 5 "audited Formula download step"
for required in \
  '        uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8' \
  '        with:' \
  '          name: audited-homebrew-formula' \
  '          path: ${{ runner.temp }}/audited-formula'; do
  require_exact_line "$formula_download_step" "$required" "audited Formula download step"
done
require_with_keys "$formula_download_step" $'name\npath' "audited Formula download step"

formula_validation_step=$(extract_named_step "$formula_publish_job" "Formula artifactをdataとして検証")
require_line_count "$formula_validation_step" 9 "Formula artifact validation step"
for required in \
  '        shell: bash' \
  '        run: |' \
  '          formula_root="${RUNNER_TEMP}/audited-formula"' \
  "          mapfile -d '' -t entries < <(find \"\${formula_root}\" -mindepth 1 -maxdepth 1 -print0)" \
  '          test "${#entries[@]}" -eq 1' \
  '          test "${entries[0]}" = "${formula_root}/cwk.rb"' \
  '          test -f "${entries[0]}"' \
  '          test ! -L "${entries[0]}"'; do
  require_exact_line "$formula_validation_step" "$required" "Formula artifact validation step"
done

app_id_input="          app-id: \${{ secrets.HOMEBREW_APP_ID }}"
private_key_input="          private-key: \${{ secrets.HOMEBREW_APP_KEY }}"
app_token='          token: ${{ steps.homebrew-token.outputs.token }}'
token_step=$(extract_named_step "$formula_publish_job" "Homebrew tap用GitHub App tokenを作成")
require_line_count "$token_step" 10 "Homebrew App token step"
for required in \
  '        id: homebrew-token' \
  '        uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3' \
  "$app_id_input" "$private_key_input" \
  '          owner: tasuku43' \
  '          repositories: homebrew-tap' \
  '          permission-contents: write' \
  '          permission-pull-requests: write'; do
  require_exact_line "$token_step" "$required" "Homebrew App token step"
done
require_with_keys "$token_step" \
  $'app-id\nprivate-key\nowner\nrepositories\npermission-contents\npermission-pull-requests' \
  "Homebrew App token step"

app_action_count=$(grep -oF 'actions/create-github-app-token@' "$workflow" | wc -l | tr -d ' ')
if [[ $app_action_count -ne 1 ]]; then
  fail "release workflow must create a Homebrew App token exactly once"
fi
for secret_name in HOMEBREW_APP_ID HOMEBREW_APP_KEY; do
  secret_name_count=$(grep -oF "$secret_name" "$workflow" | wc -l | tr -d ' ')
  if [[ $secret_name_count -ne 1 ]]; then
    fail "release workflow must reference $secret_name exactly once in the reviewed token step"
  fi
done
workflow_without_reviewed_secrets=$(grep -vFx "$app_id_input" "$workflow" | grep -vFx "$private_key_input" || true)
if printf '%s\n' "$workflow_without_reviewed_secrets" | grep -Eq '(^|[^A-Za-z0-9_])secrets([^A-Za-z0-9_]|$)'; then
  fail "release workflow must not access the secrets context outside the reviewed token inputs"
fi

tap_checkout=$(extract_named_step "$formula_publish_job" "共有Homebrew tapをcheckout")
require_line_count "$tap_checkout" 8 "shared Homebrew tap checkout"
for required in \
  '        uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7' \
  '          repository: tasuku43/homebrew-tap' \
  '          ref: main' "$app_token" \
  '          path: homebrew-tap' \
  '          persist-credentials: false'; do
  require_exact_line "$tap_checkout" "$required" "shared Homebrew tap checkout"
done
require_with_keys "$tap_checkout" \
  $'repository\nref\ntoken\npath\npersist-credentials' \
  "shared Homebrew tap checkout"

stage_step=$(extract_named_step "$formula_publish_job" "audit済みFormulaを共有tapへ配置")
require_line_count "$stage_step" 14 "shared-tap Formula staging step"
for required in \
  '        run: |' \
  '          source_formula="${RUNNER_TEMP}/audited-formula/cwk.rb"' \
  '          formula_dir="homebrew-tap/Formula"' \
  '          target_formula="${formula_dir}/cwk.rb"' \
  '          test -f "${source_formula}"' \
  '          test -d "${formula_dir}"' \
  '          test ! -L "${formula_dir}"' \
  '          test ! -L "${target_formula}"' \
  '          if [[ -e "${target_formula}" ]]; then' \
  '            test -f "${target_formula}"' \
  '          fi' \
  '          cp "${source_formula}" "${target_formula}"' \
  '          cmp -s "${source_formula}" "${target_formula}"'; do
  require_exact_line "$stage_step" "$required" "shared-tap Formula staging step"
done

pull_request_step=$(extract_named_step "$formula_publish_job" "Formula更新pull requestを作成")
require_line_count "$pull_request_step" 14 "Formula pull-request step"
pull_request_header=$(printf '%s\n' "$pull_request_step" | awk '
  { print }
  $0 == "          body: |" { exit }
')
require_line_count "$pull_request_header" 12 "Formula pull-request step header"
for required in \
  '        uses: peter-evans/create-pull-request@5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8' \
  "$app_token" \
  '          path: homebrew-tap' \
  '          add-paths: Formula/cwk.rb' \
  '          base: main' \
  '          branch: chore/homebrew-formula-${{ needs.preflight.outputs.tag }}-cwk' \
  '          delete-branch: true' \
  '          commit-message: "Update Homebrew formula for ${{ needs.preflight.outputs.tag }}"' \
  '          title: "Update Homebrew formula for ${{ needs.preflight.outputs.tag }}"' \
  '          body: |' \
  '            Formula/cwk.rbを${{ needs.preflight.outputs.tag }}の公開macOS release archiveへ更新します。' \
  '            Formulaは正確なtag sourceからrenderし、checksum固定とauditを完了しています。'; do
  require_exact_line "$pull_request_step" "$required" "Formula pull-request step"
done
require_with_keys "$pull_request_step" \
  $'token\npath\nadd-paths\nbase\nbranch\ndelete-branch\ncommit-message\ntitle\nbody' \
  "Formula pull-request step"

formula_secret_refs=$(printf '%s\n' "$formula_publish_job" | grep -oE 'secrets\.[A-Za-z0-9_]+' || true)
expected_formula_secret_refs=$'secrets.HOMEBREW_APP_ID\nsecrets.HOMEBREW_APP_KEY'
if [[ $formula_secret_refs != "$expected_formula_secret_refs" ]]; then
  printf '%s\n' "Formula job must use only the reviewed Homebrew GitHub App secrets" >&2
  diff -u \
    <(printf '%s\n' "$expected_formula_secret_refs") \
    <(printf '%s\n' "$formula_secret_refs") >&2 || true
  exit 1
fi
app_token_output='${{ steps.homebrew-token.outputs.token }}'
app_token_output_count=$(grep -oF "$app_token_output" "$workflow" | wc -l | tr -d ' ')
if [[ $app_token_output_count -ne 2 ]]; then
  fail "Homebrew App token output must be used exactly by tap checkout and pull-request steps"
fi
if printf '%s\n' "$workflow_without_reviewed_secrets" | grep -Eq 'secrets[[:space:]]*\['; then
  fail "release workflow must not use an alternate unreviewed secret reference"
fi
if printf '%s\n' "$formula_publish_job" | grep -qF 'Formula/*.rb'; then
  fail "Formula pull request must stage only the exact cwk Formula"
fi
if printf '%s\n' "$formula_job$formula_publish_job" | grep -qF './scripts/check.sh release'; then
  fail "Formula job must not repeat the Linux preflight release profile"
fi

release_checkout_line=$(printf '%s\n' "$formula_job" | grep -n -m1 -F "$release_revision_ref" | cut -d: -f1)
render_line=$(printf '%s\n' "$formula_job" | grep -n -m1 -F './scripts/render-formula.sh' | cut -d: -f1)
audit_line=$(printf '%s\n' "$formula_job" | grep -n -m1 -F './scripts/audit-formula.sh' | cut -d: -f1)
upload_line=$(printf '%s\n' "$formula_job" | grep -n -m1 -F 'actions/upload-artifact@' | cut -d: -f1)
if ((release_checkout_line >= render_line || render_line >= audit_line || audit_line >= upload_line)); then
  fail "Formula must be rendered and audited before its data artifact is uploaded"
fi
download_line=$(printf '%s\n' "$formula_publish_job" | grep -n -m1 -F 'actions/download-artifact@' | cut -d: -f1)
validate_line=$(printf '%s\n' "$formula_publish_job" | grep -n -m1 -F 'Formula artifactをdataとして検証' | cut -d: -f1)
token_line=$(printf '%s\n' "$formula_publish_job" | grep -n -m1 -F 'actions/create-github-app-token@' | cut -d: -f1)
tap_checkout_line=$(printf '%s\n' "$formula_publish_job" | grep -n -m1 -F 'repository: tasuku43/homebrew-tap' | cut -d: -f1)
stage_line=$(printf '%s\n' "$formula_publish_job" | grep -n -m1 -F 'audit済みFormulaを共有tapへ配置' | cut -d: -f1)
pull_request_line=$(printf '%s\n' "$formula_publish_job" | grep -n -m1 -F 'peter-evans/create-pull-request@' | cut -d: -f1)
if ((download_line >= validate_line || validate_line >= token_line || token_line >= tap_checkout_line || tap_checkout_line >= stage_line || stage_line >= pull_request_line)); then
  fail "fresh Formula publish job must validate the audited data before the scoped tap token and pull request"
fi

printf '%s\n' "lint-release-workflow: OK"
