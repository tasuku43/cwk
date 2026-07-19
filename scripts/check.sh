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

profile=${1:-}

usage() {
  echo "usage: $0 <fast|full|security|release|public>" >&2
  exit 2
}

run_fast() {
  local unformatted
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
  run_security
  run_release
  run_public
}

case "$profile" in
  fast) run_fast ;;
  full) run_full ;;
  security) run_security ;;
  release) run_release ;;
  public) run_public ;;
  *) usage ;;
esac
