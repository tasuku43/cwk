#!/usr/bin/env bash
# Prove that the canonical gate sanitizes Go settings and checks its toolchain
# before the first profile command.
set -euo pipefail
cd "$(dirname "$0")/.."

repo_root=$PWD
fixture_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-go-environment.XXXXXXXX")
cleanup() { rm -rf -- "$fixture_root"; }
trap cleanup EXIT
cp scripts/testdata/fake-go-gate-environment.sh "$fixture_root/go"
chmod 0700 "$fixture_root/go"

required_major='' required_minor='' required_patch=0 required_version=''
while IFS= read -r line || [[ -n $line ]]; do
  if [[ $line =~ ^[[:space:]]*go[[:space:]]+([0-9]+)\.([0-9]+)(\.([0-9]+))?([[:space:]]*(//.*)?)$ ]]; then
    required_major=${BASH_REMATCH[1]}
    required_minor=${BASH_REMATCH[2]}
    required_patch=${BASH_REMATCH[4]:-0}
    required_version="$required_major.$required_minor"
    if [[ -n ${BASH_REMATCH[4]:-} ]]; then
      required_version+=".$required_patch"
    fi
  fi
done < go.mod
if [[ -z $required_version ]]; then
  echo "test fixture could not read the go.mod version" >&2
  exit 1
fi

required_minor_number=$((10#$required_minor))
required_patch_number=$((10#$required_patch))
newer_patch=$((required_patch_number + 1))
if (( required_patch_number > 0 )); then
  older_patch=$((required_patch_number - 1))
  older_patch_version="$required_major.$required_minor.$older_patch"
elif (( required_minor_number > 0 )); then
  older_patch_version="$required_major.$((required_minor_number - 1)).999"
else
  older_patch_version="$((10#$required_major - 1)).999.999"
fi
if (( required_minor_number > 0 )); then
  older_minor_version="$required_major.$((required_minor_number - 1)).999"
else
  older_minor_version="$((10#$required_major - 1)).999.999"
fi

run_fake_go() {
  local version=$1 log_path=$2
  CHECK_STATUS=0
  CHECK_OUTPUT=$(env \
    PATH="$fixture_root:$PATH" \
    GO111MODULE=off \
    GOENV=/definitely/missing/go.env \
    GOEXPERIMENT=definitely-invalid \
    GOFIPS140=definitely-invalid \
    GOFLAGS=-run=NoTests \
    GOTOOLCHAIN=definitely-invalid \
    GOWORK=/definitely/missing/go.work \
    GOROOT=/definitely/missing/go-root \
    FAKE_GO_VERSION="$version" \
    FAKE_GO_LOG="$log_path" \
    scripts/check.sh public 2>&1) || CHECK_STATUS=$?
}

run_fake_go "go$required_version" "$fixture_root/exact.log"
if [[ $CHECK_STATUS -ne 73 || $CHECK_OUTPUT != *"canonical gate reached Go with a sanitized environment"* ]]; then
  echo "canonical gate did not accept the required Go version and sanitized environment" >&2
  printf '%s\n' "$CHECK_OUTPUT" >&2
  exit 1
fi
if [[ $(<"$fixture_root/exact.log") != $'version\nrun ./tools/repoguard --scope public' ]]; then
  echo "canonical gate did not check the Go version before the first Go-backed profile check" >&2
  cat "$fixture_root/exact.log" >&2
  exit 1
fi

run_fake_go "go$required_major.$required_minor.$newer_patch" "$fixture_root/newer-patch.log"
if [[ $CHECK_STATUS -ne 73 || $CHECK_OUTPUT != *"canonical gate reached Go with a sanitized environment"* ]]; then
  echo "canonical gate rejected a newer patch release" >&2
  printf '%s\n' "$CHECK_OUTPUT" >&2
  exit 1
fi

run_fake_go "go$required_major.$((required_minor_number + 1)).0" "$fixture_root/newer-minor.log"
if [[ $CHECK_STATUS -ne 73 || $CHECK_OUTPUT != *"canonical gate reached Go with a sanitized environment"* ]]; then
  echo "canonical gate rejected a newer minor release" >&2
  printf '%s\n' "$CHECK_OUTPUT" >&2
  exit 1
fi

for version in "$older_patch_version" "$older_minor_version"; do
  log_path="$fixture_root/${version//./-}.log"
  run_fake_go "go$version" "$log_path"
  if [[ $CHECK_STATUS -ne 1 || $CHECK_OUTPUT != *"Go $required_version 以上が必要"* ||
        $CHECK_OUTPUT != *"検出: $version"* || $CHECK_OUTPUT != *"PATH"* ||
        $CHECK_OUTPUT == *"canonical gate reached Go"* ]]; then
    echo "canonical gate did not give an actionable diagnostic for $version" >&2
    printf '%s\n' "$CHECK_OUTPUT" >&2
    exit 1
  fi
  if [[ $(<"$log_path") != version ]]; then
    echo "canonical gate invoked a Go-backed check after rejecting $version" >&2
    cat "$log_path" >&2
    exit 1
  fi
done

# Use a minimal PATH to prove a missing Go executable is diagnosed before any
# other repository check. The Bash process is selected by absolute path.
no_go_path="$fixture_root/no-go"
mkdir -m 0700 "$no_go_path"
ln -s "$(command -v dirname)" "$no_go_path/dirname"
CHECK_STATUS=0
CHECK_OUTPUT=$(PATH="$no_go_path" "$BASH" "$repo_root/scripts/check.sh" public 2>&1) || CHECK_STATUS=$?
if [[ $CHECK_STATUS -ne 1 || $CHECK_OUTPUT != *"Go $required_version 以上が必要"* ||
      $CHECK_OUTPUT != *"検出: 未検出"* || $CHECK_OUTPUT != *"PATH"* ]]; then
  echo "canonical gate did not give an actionable diagnostic when Go is missing" >&2
  printf '%s\n' "$CHECK_OUTPUT" >&2
  exit 1
fi

# An invalid profile retains its normal usage failure even when Go is absent.
CHECK_STATUS=0
CHECK_OUTPUT=$(PATH="$no_go_path" "$BASH" "$repo_root/scripts/check.sh" invalid 2>&1) || CHECK_STATUS=$?
if [[ $CHECK_STATUS -ne 2 || $CHECK_OUTPUT != *"usage:"* || $CHECK_OUTPUT == *"Go $required_version"* ]]; then
  echo "canonical gate did not report profile usage before checking for Go" >&2
  printf '%s\n' "$CHECK_OUTPUT" >&2
  exit 1
fi

echo "test-check-environment: OK"
