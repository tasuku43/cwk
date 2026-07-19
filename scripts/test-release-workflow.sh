#!/usr/bin/env bash
# Prove that release workflow policy rejects credential and shared-tap drift.
# GitHub expressions are intentionally passed as literal mutation fixtures.
# shellcheck disable=SC2016
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo_root"
checker=./scripts/lint-release-workflow.sh
workflow=.github/workflows/release.yml
tmpdir=$(mktemp -d "${TMPDIR:-/tmp}/cwk-release-workflow.XXXXXXXX")
trap 'rm -rf -- "$tmpdir"' EXIT

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

expect_rejected() {
  local name=$1
  local mutant=$2
  if "$checker" "$mutant" >/dev/null 2>&1; then
    fail "release workflow mutation was accepted: $name"
  fi
}

replace_nth_line() {
  local name=$1
  local from=$2
  local to=$3
  local occurrence=${4:-1}
  local mutant="$tmpdir/${name}.yml"
  if ! awk -v from="$from" -v to="$to" -v wanted="$occurrence" '
    $0 == from {
      seen++
      if (seen == wanted) {
        $0=to
        changed=1
      }
    }
    { print }
    END { if (!changed) exit 42 }
  ' "$workflow" >"$mutant"; then
    fail "could not construct release workflow mutation: $name"
  fi
  expect_rejected "$name" "$mutant"
}

insert_after_line() {
  local name=$1
  local after=$2
  local inserted=$3
  local occurrence=${4:-1}
  local mutant="$tmpdir/${name}.yml"
  if ! awk -v after="$after" -v inserted="$inserted" -v wanted="$occurrence" '
    { print }
    $0 == after {
      seen++
      if (!changed && seen == wanted) {
        print inserted
        changed=1
      }
    }
    END { if (!changed) exit 42 }
  ' "$workflow" >"$mutant"; then
    fail "could not construct release workflow mutation: $name"
  fi
  expect_rejected "$name" "$mutant"
}

insert_token_consumer() {
  local mutant="$tmpdir/extra-token-consumer.yml"
  if ! awk '
    { print }
    $0 == "          persist-credentials: false" {
      seen++
      if (!changed && seen == 5) {
        print "      - name: App tokenを直接利用"
        print "        env:"
        print "          GH_TOKEN: ${{ steps.homebrew-token.outputs.token }}"
        print "        run: git -C homebrew-tap push origin HEAD:main"
        changed=1
      }
    }
    END { if (!changed) exit 42 }
  ' "$workflow" >"$mutant"; then
    fail "could not construct release workflow mutation: extra-token-consumer"
  fi
  expect_rejected "extra-token-consumer" "$mutant"
}

"$checker" "$workflow" >/dev/null

insert_after_line workflow-level-env \
  'name: Release' \
  'env: { NODE_OPTIONS: "--require=/tmp/read-action-input.js" }'
insert_after_line workflow-level-defaults \
  'name: Release' \
  'defaults: { run: { shell: "bash --noprofile --norc -eo pipefail {0}" } }'
replace_nth_line workflow-root-permission \
  '  contents: read' \
  '  contents: write'
replace_nth_line generated-release-notes \
  '          args=(--verify-tag --title "${tag}" --notes-from-tag)' \
  '          args=(--verify-tag --title "${tag}" --generate-notes)'
replace_nth_line optional-recovery-tag \
  '        required: true' \
  '        required: false'
replace_nth_line unpinned-publish-tag-checkout \
  '          ref: ${{ needs.preflight.outputs.tag }}' \
  '          ref: ${{ github.ref }}'
replace_nth_line nonstable-recovery-tag \
  '            go run ./tools/releaseversion --stable "${tag}" >>"${GITHUB_OUTPUT}"' \
  '            go run ./tools/releaseversion "${tag}" >>"${GITHUB_OUTPUT}"'
replace_nth_line moving-recovery-revision \
  '          revision=$(git rev-parse --verify "${tag}^{commit}")' \
  '          revision=$(git rev-parse --verify HEAD)'
replace_nth_line prerelease-formula-recovery \
  "    if: github.event_name == 'workflow_dispatch' && needs.preflight.outputs.stable == 'true'" \
  "    if: github.event_name == 'workflow_dispatch'"
replace_nth_line ignored-release-metadata \
  '          test "${metadata}" = "${expected_metadata}"' \
  '          true'
replace_nth_line ignored-release-checksums \
  '          (cd dist && sha256sum --check checksums.txt)' \
  '          true'
insert_after_line workflow-root-extra-permission \
  '  contents: read' \
  '  issues: write'
replace_nth_line repository-scope \
  '          repositories: homebrew-tap' \
  '          repositories: homebrew-tap,other'
replace_nth_line requested-permission \
  '          permission-pull-requests: write' \
  '          permission-pull-requests: read'
insert_after_line extra-permission \
  '          permission-pull-requests: write' \
  '          permission-issues: write'
replace_nth_line source-token-permissions \
  '      contents: read' \
  '      contents: write'
replace_nth_line publish-source-token-permissions \
  '    permissions: {}' \
  '    permissions: write-all'
insert_after_line ignored-Formula-job \
  '    runs-on: macos-15' \
  '    continue-on-error: true'
replace_nth_line missing-audit-dependency \
  '    needs: [preflight, formula]' \
  '    needs: [preflight]'
replace_nth_line checksum-action \
  '        uses: actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8' \
  '        run: true' \
  2
replace_nth_line tap-destination \
  '          repository: tasuku43/homebrew-tap' \
  '          repository: tasuku43/cwk'
replace_nth_line pull-request-token \
  '          token: ${{ steps.homebrew-token.outputs.token }}' \
  '          token: ${{ github.token }}' \
  2
replace_nth_line pull-request-path \
  '          path: homebrew-tap' \
  '          path: .' \
  2
replace_nth_line staged-path \
  '          add-paths: Formula/cwk.rb' \
  '          add-paths: Formula/*.rb'
replace_nth_line pull-request-base \
  '          base: main' \
  '          base: develop'
replace_nth_line pull-request-title \
  '          title: "Update Homebrew formula for ${{ needs.preflight.outputs.tag }}"' \
  '          title: "Formula update for ${{ needs.preflight.outputs.tag }}"'
replace_nth_line persisted-tap-credential \
  '          persist-credentials: false' \
  '          persist-credentials: true' \
  5
replace_nth_line Formula-source-binding \
  '          source_formula="${RUNNER_TEMP}/audited-formula/cwk.rb"' \
  '          source_formula="Formula/cwk.rb"'
replace_nth_line symlinked-Formula-directory \
  '          test ! -L "${formula_dir}"' \
  '          true'
replace_nth_line symlinked-Formula-target \
  '          test ! -L "${target_formula}"' \
  '          true'
insert_after_line publish-job-env-after-steps \
  '            Formulaは正確なtag sourceからrenderし、checksum固定とauditを完了しています。' \
  '    env: { NODE_OPTIONS: "--require=/tmp/read-action-input.js" }'
replace_nth_line disabled-audit \
  '          ./scripts/audit-formula.sh "${formula}"' \
  '          ./scripts/audit-formula.sh "${formula}" || true'
insert_after_line ignored-audit-step \
  '        id: formula' \
  '        continue-on-error: true'
insert_after_line repository-code-in-publish-job \
  '          test ! -L "${entries[0]}"' \
  '      - run: ./scripts/render-formula.sh'
insert_token_consumer
insert_after_line anonymous-token-consumer \
  '          persist-credentials: false' \
  "      - run: GH_TOKEN=\"\${{ steps['homebrew-token'].outputs.token }}\" git -C homebrew-tap push origin HEAD:main" \
  5
insert_after_line extra-token-action \
  '          persist-credentials: false' \
  '      - uses: actions/create-github-app-token@bcd2ba49218906704ab6c1aa796996da409d3eb1 # v3' \
  5
insert_after_line secret-outside-token-step \
  '          persist-credentials: false' \
  '          leaked: ${{ secrets.HOMEBREW_APP_KEY }}' \
  5
insert_after_line whole-secrets-context \
  '          persist-credentials: false' \
  '          leaked: ${{ toJSON(secrets) }}' \
  5
insert_after_line duplicate-secret-reference \
  '            Formulaは正確なtag sourceからrenderし、checksum固定とauditを完了しています。' \
  '            ${{ secrets.HOMEBREW_APP_KEY }}'
insert_after_line ignored-pull-request \
  '            Formulaは正確なtag sourceからrenderし、checksum固定とauditを完了しています。' \
  '        continue-on-error: true'

printf '%s\n' "test-release-workflow: OK"
