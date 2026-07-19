#!/usr/bin/env bash
# Prove Formula audit cleanup never removes a pre-existing user tap.
set -euo pipefail
cd "$(dirname "$0")/.."

test_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-audit-test.XXXXXXXX")
cleanup() { rm -rf -- "$test_root"; }
trap cleanup EXIT

fake_brew=$PWD/scripts/testdata/fake-brew.sh
formula=$test_root/cwk.rb
printf '%s\n' \
  'class Cwk < Formula' \
  '  def install' \
  '    doc.install "LICENSE", "THIRD_PARTY_NOTICES"' \
  '  end' \
  'end' >"$formula"
chmod 0600 "$formula"
legacy_tap=cwk-ci/audit

missing_notices_formula=$test_root/cwk-missing-notices.rb
printf '%s\n' 'class Cwk < Formula' 'end' >"$missing_notices_formula"
if BREW_COMMAND=$fake_brew AUDIT_FORMULA_BINARY=cwk \
  scripts/audit-formula.sh "$missing_notices_formula" >/dev/null 2>&1; then
  echo "audit-formula accepted a Formula that discards release notices" >&2
  exit 1
fi

run_case() {
  expected=$1
  audit_failure=$2
  case_root=$test_root/$expected
  mkdir -p "$case_root"
  log=$case_root/brew.log
  : >"$log"

  set +e
  FAKE_BREW_LOG=$log \
    FAKE_BREW_ROOT=$case_root \
    FAKE_BREW_EXISTING_TAP=$legacy_tap \
    FAKE_BREW_AUDIT_FAIL=$audit_failure \
    FAKE_BREW_EXPECT_FORMULA_MODE=644 \
    BREW_COMMAND=$fake_brew \
    AUDIT_FORMULA_BINARY=cwk \
    scripts/audit-formula.sh "$formula" >/dev/null 2>&1
  status=$?
  set -e

  if [[ $expected == success && $status -ne 0 ]]; then
    echo "audit-formula success fixture failed with status $status" >&2
    exit 1
  fi
  if [[ $expected == failure && $status -eq 0 ]]; then
    echo "audit-formula failure fixture unexpectedly succeeded" >&2
    exit 1
  fi
  if grep -Fxq "untap $legacy_tap" "$log"; then
    echo "audit-formula removed a pre-existing user tap" >&2
    exit 1
  fi
  created_tap=$(awk '$1 == "tap-new" && $2 == "--no-git" { print $3 }' "$log")
  if [[ -z $created_tap || $created_tap == "$legacy_tap" ]]; then
    echo "audit-formula did not create a unique owned tap" >&2
    exit 1
  fi
  if [[ $(grep -Fc "untap $created_tap" "$log") -ne 1 ]]; then
    echo "audit-formula did not clean up exactly its owned tap" >&2
    exit 1
  fi
}

run_case success false
run_case failure true

echo "test-audit-formula: OK"
