#!/usr/bin/env bash
# Validate release scripts and public binary distribution contracts without publishing.
set -euo pipefail
cd "$(dirname "$0")/.."
export GO111MODULE=on
export GOENV=off
export GOEXPERIMENT=
export GOFIPS140=off
export GOFLAGS=
export GOTOOLCHAIN=local
export GOWORK=off

bash -n \
  scripts/check.sh \
  scripts/package-release.sh \
  scripts/render-formula.sh \
  scripts/audit-formula.sh \
  scripts/lint-release-workflow.sh \
  scripts/test-audit-formula.sh \
  scripts/test-check-environment.sh \
  scripts/test-release-workflow.sh \
  scripts/testdata/fake-go-gate-environment.sh \
  scripts/testdata/fake-brew.sh
if ! command -v shellcheck >/dev/null 2>&1; then
  echo "release gate requires ShellCheck for every repository shell script" >&2
  exit 1
fi
shellcheck_version=$(shellcheck --version | awk '$1 == "version:" { print $2 }')
if [[ ! $shellcheck_version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "could not determine a semantic ShellCheck version" >&2
  exit 1
fi
if ! awk -v current="$shellcheck_version" -v floor=0.9.0 'BEGIN {
  split(current, have, "."); split(floor, need, ".")
  for (i = 1; i <= 3; i++) {
    if ((have[i] + 0) > (need[i] + 0)) exit 0
    if ((have[i] + 0) < (need[i] + 0)) exit 1
  }
  exit 0
}'; then
  echo "release gate requires ShellCheck >= 0.9.0; running $shellcheck_version" >&2
  exit 1
fi
git ls-files -co --exclude-standard -z -- '*.sh' |
  while IFS= read -r -d '' script; do
    [[ -f $script ]] && printf '%s\0' "$script"
  done |
  xargs -0 shellcheck
go test ./tools/archivepack ./tools/internal/releaseversion ./tools/releaseversion
required_go=go$(awk '$1 == "go" { print $2 }' go.mod)
actual_go=$(go env GOVERSION)
if [[ $actual_go != "$required_go" ]]; then
  echo "release gate requires $required_go from go.mod; running $actual_go" >&2
  exit 1
fi
go mod verify >/dev/null
local_module_replacements=$(go list -m -f '{{if .Replace}}{{if not .Replace.Version}}{{.Path}} => {{.Replace.Dir}}{{end}}{{end}}' all)
if [[ -n $local_module_replacements ]]; then
  echo "release gate rejects local filesystem module replacements:" >&2
  printf '%s\n' "$local_module_replacements" >&2
  exit 1
fi
binary=$(go run ./tools/projectmeta --field binary_name)
module=$(go run ./tools/projectmeta --field go_module)
formula_class=$(go run ./tools/projectmeta --field formula_class)
template=Formula/${binary}.rb.template
test -f "$template"

for required in \
  '@@FORMULA_CLASS@@' '@@DESCRIPTION@@' '@@LICENSE_SPDX@@' '@@REPOSITORY_URL@@' '@@VERSION@@' \
  '@@MACOS_ARM64_URL@@' '@@MACOS_AMD64_URL@@' \
  '@@MACOS_ARM64_SHA256@@' '@@MACOS_AMD64_SHA256@@' '@@BINARY_NAME@@'; do
  grep -qF "$required" "$template" || {
    echo "Formula template is missing $required" >&2
    exit 1
  }
done
for required_install in \
  'bin.install "@@BINARY_NAME@@"' \
  'doc.install "LICENSE", "THIRD_PARTY_NOTICES"'; do
  grep -qF "$required_install" "$template" || {
    echo "Formula template is missing required install: $required_install" >&2
    exit 1
  }
done

for forbidden in 'git describe' '{{.VERSION}}' '{{.COMMIT}}'; do
  if grep -qF "$forbidden" Taskfile.yml; then
    echo "local build must not interpolate repository-controlled version metadata: $forbidden" >&2
    exit 1
  fi
done
grep -qF 'go build -buildvcs=false -trimpath -o bin/' Taskfile.yml || {
  echo "local build must use fixed dev metadata without implicit VCS stamping" >&2
  exit 1
}
for required in \
  'export GO111MODULE=on' 'export GOENV=off' 'export GOEXPERIMENT=' 'export GOFIPS140=off' \
  'export GOFLAGS=' 'export GOTOOLCHAIN=local' 'export GOWORK=off'; do
  for go_boundary in scripts/check.sh scripts/package-release.sh; do
    if ! grep -qFx "$required" "$go_boundary"; then
      echo "$go_boundary does not neutralize ambient Go configuration: $required" >&2
      exit 1
    fi
  done
done
scripts/test-check-environment.sh >/dev/null
scripts/test-release-workflow.sh >/dev/null

for forbidden in 'HOMEBREW_GITHUB_API_TOKEN' 'api.github.com/repos/' 'Authorization: Bearer'; do
  if grep -R -F "$forbidden" Formula scripts/render-formula.sh .github/workflows/release.yml >/dev/null 2>&1; then
    echo "public release path contains private-asset behavior: $forbidden" >&2
    exit 1
  fi
done

if scripts/package-release.sh bad-tag 0000000000000000000000000000000000000000 linux amd64 dist >/dev/null 2>&1; then
  echo "package-release accepted an invalid tag" >&2
  exit 1
fi
ambient_status=0
ambient_output=$(env \
  GO111MODULE=off \
  GOENV=/definitely/missing/go.env \
  GOEXPERIMENT=definitely-invalid \
  GOFIPS140=definitely-invalid \
  GOFLAGS=-definitely-invalid \
  GOTOOLCHAIN=definitely-invalid \
  GOWORK=/definitely/missing/go.work \
  scripts/package-release.sh \
    v0.0.0 0000000000000000000000000000000000000000 plan9 amd64 dist 2>&1) || ambient_status=$?
if [[ $ambient_status -ne 2 || $ambient_output != *"unsupported release target: plan9/amd64"* ]]; then
  echo "package-release did not neutralize malicious ambient Go configuration" >&2
  printf '%s\n' "$ambient_output" >&2
  exit 1
fi
if go run ./tools/releaseversion v1.2.3-01 >/dev/null 2>&1; then
  echo "releaseversion accepted a numeric prerelease identifier with a leading zero" >&2
  exit 1
fi
if go run ./tools/releaseversion v1.2.3+different-build >/dev/null 2>&1; then
  echo "releaseversion accepted build metadata excluded by immutable-release policy" >&2
  exit 1
fi
if scripts/render-formula.sh v1.2.3-rc.1 https://github.com/tasuku43/cwk /dev/null >/dev/null 2>&1; then
  echo "render-formula accepted a prerelease tag" >&2
  exit 1
fi

# Build one primary complete matrix for archive, checksum, and Formula checks.
# A second independent matrix below proves that identical inputs reproduce the
# exact archive bytes instead of merely reproducing their names and contents.
sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -- "$1" | awk '{ print $1 }'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -- "$1" | awk '{ print $1 }'
    return
  fi
  echo "release gate requires sha256sum or shasum" >&2
  return 1
}

release_root=$(mktemp -d "${TMPDIR:-/tmp}/${binary}-release-check.XXXXXXXX")
cleanup() { rm -rf -- "$release_root"; }
trap cleanup EXIT
release_input_roots=(
  go.mod
  LICENSE
  THIRD_PARTY_NOTICES
  .harness/project.json
  .codex
  .github/workflows/release.yml
  Formula
  Taskfile.yml
  scripts
  cmd
  internal
  tools
)
for optional_input in go.sum vendor; do
  if [[ -e $optional_input ]]; then
    release_input_roots+=("$optional_input")
  fi
done
release_input_fingerprint() {
  local diagnostic=${1:-}
  local manifest path_list unsafe_path_list path digest
  manifest=$(mktemp "$release_root/package-inputs.XXXXXXXX")
  path_list=$(mktemp "$release_root/package-input-paths.XXXXXXXX")
  unsafe_path_list=$(mktemp "$release_root/package-input-unsafe-paths.XXXXXXXX")
  : >"$manifest"
  find "${release_input_roots[@]}" ! -type d ! -type f -print0 >"$unsafe_path_list"
  if [[ -s $unsafe_path_list ]]; then
    echo "release inputs must contain only regular files and directories" >&2
    return 1
  fi
  find "${release_input_roots[@]}" -type f -print0 | LC_ALL=C sort -z >"$path_list"
  if [[ ! -s $path_list ]]; then
    echo "release input set is empty" >&2
    return 1
  fi
  if [[ -n $diagnostic ]]; then
    : >"$diagnostic"
  fi
  while IFS= read -r -d '' path; do
    digest=$(sha256_of "$path")
    printf '%s\0%s\0' "$path" "$digest" >>"$manifest"
    if [[ -n $diagnostic ]]; then
      printf '%s  %q\n' "$digest" "$path" >>"$diagnostic"
    fi
  done <"$path_list"
  digest=$(sha256_of "$manifest")
  rm -f -- "$manifest" "$path_list" "$unsafe_path_list"
  printf '%s' "$digest"
}

report_release_input_drift() {
  local phase=$1
  local before=$2
  local after=$3
  echo "release inputs changed $phase:" >&2
  diff -u "$before" "$after" >&2 || true
}

targets=(
  linux/amd64/tar.gz
  linux/arm64/tar.gz
  darwin/amd64/tar.gz
  darwin/arm64/tar.gz
  windows/amd64/zip
)
reviewed_notice_modules=$release_root/reviewed-notice-modules
printf '%s\n' \
  'golang.org/x/sys v0.37.0' \
  'golang.org/x/term v0.36.0' >"$reviewed_notice_modules"
production_modules_raw=$release_root/production-modules-raw
: >"$production_modules_raw"
for target in "${targets[@]}"; do
  goos=${target%%/*}
  remainder=${target#*/}
  goarch=${remainder%%/*}
  target_environment=(CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch")
  case "$goarch" in
    amd64) target_environment+=(GOAMD64=v1) ;;
    arm64) target_environment+=(GOARM64=v8.0) ;;
  esac
  env "${target_environment[@]}" go list -deps \
    -f '{{with .Module}}{{if not .Main}}{{if .Replace}}{{.Replace.Path}} {{.Replace.Version}}{{else}}{{.Path}} {{.Version}}{{end}}{{end}}{{end}}' \
    "./cmd/$binary" >>"$production_modules_raw"
done
production_modules=$release_root/production-modules
awk 'NF == 2' "$production_modules_raw" | LC_ALL=C sort -u >"$production_modules"
if ! cmp -s "$reviewed_notice_modules" "$production_modules"; then
  echo "linked production modules do not match the reviewed notice manifest:" >&2
  diff -u "$reviewed_notice_modules" "$production_modules" >&2 || true
  exit 1
fi

go_license=$(go env GOROOT)/LICENSE
go_patents=$(go env GOROOT)/PATENTS
term_license=$(go list -m -f '{{.Dir}}/LICENSE' golang.org/x/term)
sys_license=$(go list -m -f '{{.Dir}}/LICENSE' golang.org/x/sys)
for dependency_license in "$go_license" "$go_patents" "$term_license" "$sys_license"; do
  if [[ ! -f $dependency_license || -L $dependency_license ]]; then
    echo "reviewed dependency license is not a regular file: $dependency_license" >&2
    exit 1
  fi
done
expected_notices=$release_root/expected-third-party-notices
{
  printf '%s\n\n' 'THIRD-PARTY SOFTWARE NOTICES'
  printf '%s\n%s\n' 'Go standard library and runtime go1.26.5' '------------------------------------------'
  cat -- "$go_license"
  printf '\n'
  printf '%s\n%s\n' 'Go standard library and runtime PATENTS' '---------------------------------------'
  cat -- "$go_patents"
  printf '\n'
  printf '%s\n%s\n' 'golang.org/x/term v0.36.0' '--------------------------'
  cat -- "$term_license"
  printf '\n'
  printf '%s\n%s\n' 'golang.org/x/sys v0.37.0' '-------------------------'
  cat -- "$sys_license"
} >"$expected_notices"
if ! cmp -s THIRD_PARTY_NOTICES "$expected_notices"; then
  echo "THIRD_PARTY_NOTICES does not reproduce the reviewed Go license, patent grant, and module licenses verbatim" >&2
  exit 1
fi

dist=$release_root/dist
primary_go_cache=$release_root/go-cache-primary
reproduction_go_cache=$release_root/go-cache-reproduction
mkdir -p "$dist" "$primary_go_cache" "$reproduction_go_cache"
release_tag=v0.0.0
release_revision=0000000000000000000000000000000000000000
expected_assets=$release_root/expected-assets.txt
: >"$expected_assets"
primary_inputs_before=$release_root/primary-inputs-before.txt
primary_inputs_after=$release_root/primary-inputs-after.txt
inputs_before_primary=$(release_input_fingerprint "$primary_inputs_before")
go mod verify >/dev/null

for target in "${targets[@]}"; do
  goos=${target%%/*}
  remainder=${target#*/}
  goarch=${remainder%%/*}
  extension=${target##*/}
  asset=${binary}_${release_tag}_${goos}_${goarch}.${extension}
  executable=$binary
  if [[ $goos == windows ]]; then
    executable=${binary}.exe
  fi

  env GOCACHE="$primary_go_cache" scripts/package-release.sh \
    "$release_tag" "$release_revision" "$goos" "$goarch" "$dist" >/dev/null
  archive=$dist/$asset
  test -s "$archive"
  printf '%s\n' "$asset" >>"$expected_assets"

  if [[ $extension == zip ]]; then
    members=$(unzip -Z1 "$archive")
  else
    members=$(tar -tzf "$archive")
  fi
  expected_members=$(printf 'LICENSE\nTHIRD_PARTY_NOTICES\n%s' "$executable")
  if [[ $members != "$expected_members" ]]; then
    echo "archive $asset contains unexpected entries: $members" >&2
    exit 1
  fi

  extract_dir=$release_root/extract-${goos}-${goarch}
  mkdir -p "$extract_dir"
  if [[ $extension == zip ]]; then
    unzip -q "$archive" -d "$extract_dir"
  else
    tar -xzf "$archive" -C "$extract_dir"
  fi
  if [[ $(find "$extract_dir" -mindepth 1 -maxdepth 1 -type f | wc -l | tr -d ' ') -ne 3 || \
        -n $(find "$extract_dir" -mindepth 1 -maxdepth 1 ! -type f -print -quit) || \
        ! -f $extract_dir/$executable || ! -f $extract_dir/LICENSE || ! -f $extract_dir/THIRD_PARTY_NOTICES ]]; then
    echo "archive $asset did not extract to exactly $executable, LICENSE, and THIRD_PARTY_NOTICES" >&2
    exit 1
  fi
  go run ./tools/archivepack verify \
    "$extension" \
    "$archive" \
    "$extract_dir/$executable" "$executable" 0755 \
    THIRD_PARTY_NOTICES THIRD_PARTY_NOTICES 0644 \
    LICENSE LICENSE 0644
  if ! cmp -s LICENSE "$extract_dir/LICENSE"; then
    echo "archive $asset contains changed project license" >&2
    exit 1
  fi
  if ! cmp -s THIRD_PARTY_NOTICES "$extract_dir/THIRD_PARTY_NOTICES"; then
    echo "archive $asset contains changed third-party notices" >&2
    exit 1
  fi
  metadata=$(go version -m "$extract_dir/$executable")
  for required_metadata in "$module" "GOOS=$goos" "GOARCH=$goarch"; do
    if ! printf '%s\n' "$metadata" | grep -Fq "$required_metadata"; then
      echo "archive $asset is missing build metadata: $required_metadata" >&2
      exit 1
    fi
  done
done
if ! go mod verify >/dev/null; then
  echo "module inputs changed or failed verification during the primary archive pass" >&2
  exit 1
fi
inputs_after_primary=$(release_input_fingerprint "$primary_inputs_after")
if [[ $inputs_before_primary != "$inputs_after_primary" ]]; then
  report_release_input_drift "during the primary archive pass; reproducibility comparison was not attempted" "$primary_inputs_before" "$primary_inputs_after"
  exit 1
fi

LC_ALL=C sort -o "$expected_assets" "$expected_assets"
actual_assets=$release_root/actual-assets.txt
find "$dist" -maxdepth 1 -type f -exec basename {} \; | LC_ALL=C sort >"$actual_assets"
if ! cmp -s "$expected_assets" "$actual_assets"; then
  echo "release archive set does not match the supported five-target matrix" >&2
  exit 1
fi

repro_dist=$release_root/repro-dist
mkdir -p "$repro_dist"
reproduction_inputs_before=$release_root/reproduction-inputs-before.txt
reproduction_inputs_after=$release_root/reproduction-inputs-after.txt
inputs_before_reproduction=$(release_input_fingerprint "$reproduction_inputs_before")
if [[ $inputs_after_primary != "$inputs_before_reproduction" ]]; then
  report_release_input_drift "before the reproduction archive pass" "$primary_inputs_after" "$reproduction_inputs_before"
  exit 1
fi
go mod verify >/dev/null
for target in "${targets[@]}"; do
  goos=${target%%/*}
  remainder=${target#*/}
  goarch=${remainder%%/*}
  extension=${target##*/}
  asset=${binary}_${release_tag}_${goos}_${goarch}.${extension}

  env GOCACHE="$reproduction_go_cache" scripts/package-release.sh \
    "$release_tag" "$release_revision" "$goos" "$goarch" "$repro_dist" >/dev/null
done
if ! go mod verify >/dev/null; then
  echo "module inputs changed or failed verification during the reproduction archive pass" >&2
  exit 1
fi
inputs_after_reproduction=$(release_input_fingerprint "$reproduction_inputs_after")
if [[ $inputs_before_reproduction != "$inputs_after_reproduction" ]]; then
  report_release_input_drift "during the reproduction archive pass; digest comparison is invalid" "$reproduction_inputs_before" "$reproduction_inputs_after"
  exit 1
fi
for target in "${targets[@]}"; do
  goos=${target%%/*}
  remainder=${target#*/}
  goarch=${remainder%%/*}
  extension=${target##*/}
  asset=${binary}_${release_tag}_${goos}_${goarch}.${extension}

  primary_digest=$(sha256_of "$dist/$asset")
  reproduced_digest=$(sha256_of "$repro_dist/$asset")
  if [[ $primary_digest != "$reproduced_digest" ]]; then
    echo "release archive is not byte-for-byte reproducible: $asset" >&2
    exit 1
  fi
done
repro_assets=$release_root/repro-assets.txt
find "$repro_dist" -maxdepth 1 -type f -exec basename {} \; | LC_ALL=C sort >"$repro_assets"
if ! cmp -s "$expected_assets" "$repro_assets"; then
  echo "reproduced archive set does not match the supported five-target matrix" >&2
  exit 1
fi

# The package command is create-only. This negative check reaches the collision
# guard before another build, so the two verified matrices above remain the
# only builds performed by this profile.
first_asset=$dist/${binary}_${release_tag}_linux_amd64.tar.gz
first_digest_before=$(sha256_of "$first_asset")
if scripts/package-release.sh "$release_tag" "$release_revision" linux amd64 "$dist" >/dev/null 2>&1; then
  echo "package-release overwrote an existing archive" >&2
  exit 1
fi
first_digest_after=$(sha256_of "$first_asset")
if [[ $first_digest_before != "$first_digest_after" ]]; then
  echo "package-release changed an existing archive on collision" >&2
  exit 1
fi

checksums=$dist/checksums.txt
: >"$checksums"
while IFS= read -r asset; do
  printf '%s  %s\n' "$(sha256_of "$dist/$asset")" "$asset" >>"$checksums"
done <"$expected_assets"
if [[ $(wc -l <"$checksums" | tr -d ' ') -ne 5 ]]; then
  echo "checksums.txt does not contain exactly five archives" >&2
  exit 1
fi
checksum_assets=$release_root/checksum-assets.txt
awk '{ print $2 }' "$checksums" | LC_ALL=C sort >"$checksum_assets"
if ! cmp -s "$expected_assets" "$checksum_assets"; then
  echo "checksums.txt does not correspond to the complete archive set" >&2
  exit 1
fi
while read -r digest asset extra; do
  if [[ -n ${extra:-} ]] || ! printf '%s' "$digest" | grep -Eq '^[0-9a-f]{64}$'; then
    echo "invalid checksum record for $asset" >&2
    exit 1
  fi
  if [[ $digest != "$(sha256_of "$dist/$asset")" ]]; then
    echo "checksum mismatch for $asset" >&2
    exit 1
  fi
done <"$checksums"

formula=$release_root/${binary}.rb
repository_url=https://github.com/tasuku43/cwk
scripts/render-formula.sh "$release_tag" "$repository_url" "$checksums" "$formula" >/dev/null
test -s "$formula"
arm64_asset=${binary}_${release_tag}_darwin_arm64.tar.gz
amd64_asset=${binary}_${release_tag}_darwin_amd64.tar.gz
arm64_sha=$(awk -v asset="$arm64_asset" '$2 == asset { print $1 }' "$checksums")
amd64_sha=$(awk -v asset="$amd64_asset" '$2 == asset { print $1 }' "$checksums")
for expected_formula in \
  "class $formula_class < Formula" \
  "version \"${release_tag#v}\"" \
  "$repository_url/releases/download/$release_tag/$arm64_asset" \
  "$repository_url/releases/download/$release_tag/$amd64_asset" \
  "sha256 \"$arm64_sha\"" \
  "sha256 \"$amd64_sha\"" \
  'doc.install "LICENSE", "THIRD_PARTY_NOTICES"'; do
  if ! grep -Fq "$expected_formula" "$formula"; then
    echo "rendered Formula is missing: $expected_formula" >&2
    exit 1
  fi
done
if grep -qE '@@[A-Z0-9_]+@@' "$formula"; then
  echo "positive Formula render retained a placeholder" >&2
  exit 1
fi
if ! command -v ruby >/dev/null 2>&1; then
  echo "release gate requires Ruby for Formula syntax validation; install Ruby or use the documented CI release gate" >&2
  exit 1
fi
ruby -c "$formula" >/dev/null

scripts/test-audit-formula.sh >/dev/null

release_checks_inputs_after=$release_root/release-checks-inputs-after.txt
inputs_after_release_checks=$(release_input_fingerprint "$release_checks_inputs_after")
if [[ $inputs_after_reproduction != "$inputs_after_release_checks" ]]; then
  report_release_input_drift "during checksum or Formula validation" "$reproduction_inputs_after" "$release_checks_inputs_after"
  exit 1
fi

echo "lint-release: OK"
