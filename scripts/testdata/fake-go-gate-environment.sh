#!/usr/bin/env bash
# Test double used by test-check-environment.sh. It must be installed as `go`.
set -euo pipefail

[[ ${GO111MODULE:-} == on ]]
[[ ${GOENV:-} == off ]]
[[ ${GOEXPERIMENT+x} == x && -z $GOEXPERIMENT ]]
[[ ${GOFIPS140:-} == off ]]
[[ ${GOFLAGS+x} == x && -z $GOFLAGS ]]
[[ ${GOTOOLCHAIN:-} == local ]]
[[ ${GOWORK:-} == off ]]
[[ ! ${GOROOT+x} ]]

if [[ -n ${FAKE_GO_LOG:-} ]]; then
  printf '%s\n' "$*" >> "$FAKE_GO_LOG"
fi

if [[ ${1:-} == version ]]; then
  echo "go version ${FAKE_GO_VERSION:-go1.26.10} linux/amd64"
  exit 0
fi

echo "canonical gate reached Go with a sanitized environment" >&2
exit 73
