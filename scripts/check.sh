#!/usr/bin/env bash
# This is the only implementation of repository quality gates. Task, agent
# hooks, CI, and release workflows call a named profile here.
set -euo pipefail
cd "$(dirname "$0")/.."
export GO111MODULE=on
export GOENV=off
export GOEXPERIMENT=
export GOFIPS140=off
export GOFLAGS=
export GOTOOLCHAIN=local
export GOWORK=off
unset GOROOT

profile=${1:-}

usage() {
  echo "usage: $0 <fast|full|security|release|public>" >&2
  exit 2
}

case "$profile" in
  fast|full|security|release|public) ;;
  *) usage ;;
esac

check_go_toolchain() {
  local required_major='' required_minor='' required_patch='' required_version=''
  local directive_count=0 line version_output actual_major actual_minor actual_patch actual_version
  local required_major_number required_minor_number required_patch_number
  local actual_major_number actual_minor_number actual_patch_number

  if [[ ! -r go.mod ]]; then
    echo "go.mod から必要な Go バージョンを確認できません。リポジトリのルートで再実行してください。" >&2
    exit 1
  fi

  while IFS= read -r line || [[ -n $line ]]; do
    if [[ $line =~ ^[[:space:]]*go[[:space:]]+([0-9]+)\.([0-9]+)(\.([0-9]+))?([[:space:]]*(//.*)?)$ ]]; then
      directive_count=$((directive_count + 1))
      required_major=${BASH_REMATCH[1]}
      required_minor=${BASH_REMATCH[2]}
      required_patch=${BASH_REMATCH[4]:-0}
      required_version="$required_major.$required_minor"
      if [[ -n ${BASH_REMATCH[4]:-} ]]; then
        required_version+=".$required_patch"
      fi
    fi
  done < go.mod

  if [[ $directive_count -ne 1 ]]; then
    echo "go.mod にある必要な Go バージョンを一意に読み取れません。" >&2
    exit 1
  fi

  if ! command -v go >/dev/null 2>&1; then
    echo "このリポジトリのチェックには Go $required_version 以上が必要です（検出: 未検出）。" >&2
    echo "Go $required_version 以上をインストールし、その go コマンドを PATH に追加してください。" >&2
    exit 1
  fi

  if ! version_output=$(go version 2>/dev/null); then
    echo "このリポジトリのチェックには Go $required_version 以上が必要です（検出: バージョンを確認できません）。" >&2
    echo "Go $required_version 以上をインストールし、その go コマンドを PATH に追加してください。" >&2
    exit 1
  fi

  if [[ ! $version_output =~ (^|[[:space:]])go([0-9]+)\.([0-9]+)(\.([0-9]+))?([[:space:]]|$) ]]; then
    echo "go version の出力から Go のバージョンを確認できません。" >&2
    echo "Go $required_version 以上をインストールし、その go コマンドを PATH に追加してください。" >&2
    exit 1
  fi
  actual_major=${BASH_REMATCH[2]}
  actual_minor=${BASH_REMATCH[3]}
  actual_patch=${BASH_REMATCH[5]:-0}
  actual_version="$actual_major.$actual_minor.$actual_patch"

  required_major_number=$((10#$required_major))
  required_minor_number=$((10#$required_minor))
  required_patch_number=$((10#$required_patch))
  actual_major_number=$((10#$actual_major))
  actual_minor_number=$((10#$actual_minor))
  actual_patch_number=$((10#$actual_patch))

  if (( actual_major_number < required_major_number )) ||
     (( actual_major_number == required_major_number && actual_minor_number < required_minor_number )) ||
     (( actual_major_number == required_major_number && actual_minor_number == required_minor_number && actual_patch_number < required_patch_number )); then
    echo "このリポジトリのチェックには Go $required_version 以上が必要です（検出: ${actual_version}）。" >&2
    echo "Go $required_version 以上をインストールし、その go コマンドを PATH に追加してください。" >&2
    exit 1
  fi
}

check_go_toolchain

run_fast() {
  local unformatted
  scripts/test-check-environment.sh
  scripts/test-live-chatwork-smoke.sh
  scripts/test-live-chatwork-workflow.sh
  unformatted=$(gofmt -l .)
  if [[ -n "$unformatted" ]]; then
    echo "gofmt is required for:" >&2
    echo "$unformatted" >&2
    return 1
  fi
  go run ./tools/repoguard --scope hygiene
  go run ./tools/archlint
  go run ./tools/contractlint
  go run ./tools/localizationlint
  go test ./...
}

run_security() {
  go mod verify
  go run ./tools/repoguard --scope security
  go run github.com/securego/gosec/v2/cmd/gosec@v2.27.1 -quiet ./...
  go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
}

run_release() {
  ./scripts/lint-release.sh
  go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7
}

run_public() {
  go run ./tools/repoguard --scope public
  go run ./tools/contractlint
}

run_full() {
  run_fast
  go vet ./...
  go test -race ./...
  go mod tidy -diff
  git diff --check
}

case "$profile" in
  fast) run_fast ;;
  full) run_full ;;
  security) run_security ;;
  release) run_release ;;
  public) run_public ;;
esac
