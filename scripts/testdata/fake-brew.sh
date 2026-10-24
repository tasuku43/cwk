#!/usr/bin/env bash
set -euo pipefail

: "${FAKE_BREW_LOG:?}"
: "${FAKE_BREW_ROOT:?}"
: "${FAKE_BREW_EXISTING_TAP:?}"
: "${FAKE_BREW_EXPECT_CONFIG_ROOT:?}"

[[ ${HOMEBREW_DEVELOPER:-} == 1 ]]
[[ ${XDG_CONFIG_HOME:-} == "$FAKE_BREW_EXPECT_CONFIG_ROOT"/* ]]
[[ -d ${XDG_CONFIG_HOME:?} ]]

command_name=${1:-}
shift || true
printf '%s' "$command_name" >>"$FAKE_BREW_LOG"
for argument in "$@"; do
  printf ' %s' "$argument" >>"$FAKE_BREW_LOG"
done
printf '\n' >>"$FAKE_BREW_LOG"

tap_path() {
  printf '%s/%s' "$FAKE_BREW_ROOT" "${1//\//__}"
}

file_mode() {
  if mode=$(stat -f '%Lp' "$1" 2>/dev/null); then
    printf '%s' "$mode"
    return
  fi
  stat -c '%a' "$1"
}

case "$command_name" in
  tap)
    printf '%s\n' "$FAKE_BREW_EXISTING_TAP"
    ;;
  tap-new)
    [[ ${1:-} == --no-git ]]
    tap=${2:?}
    path=$(tap_path "$tap")
    mkdir -p "$path/Formula"
    printf '%s' "$tap" >"$FAKE_BREW_ROOT/created-tap"
    ;;
  --repository)
    tap=${1:?}
    [[ -f $FAKE_BREW_ROOT/created-tap ]]
    [[ $(<"$FAKE_BREW_ROOT/created-tap") == "$tap" ]]
    tap_path "$tap"
    ;;
  trust)
    [[ ${1:-} == --formula ]]
    formula=${2:?}
    tap=${formula%/*}
    binary=${formula##*/}
    [[ -f $FAKE_BREW_ROOT/created-tap ]]
    [[ $(<"$FAKE_BREW_ROOT/created-tap") == "$tap" ]]
    [[ -f "$(tap_path "$tap")/Formula/${binary}.rb" ]]
    printf '%s' "$formula" >"$FAKE_BREW_ROOT/trusted-formula"
    ;;
  audit)
    formula_ref=${2:?}
    [[ -f $FAKE_BREW_ROOT/trusted-formula ]]
    [[ $(<"$FAKE_BREW_ROOT/trusted-formula") == "$formula_ref" ]]
    if [[ -n ${FAKE_BREW_EXPECT_FORMULA_MODE:-} ]]; then
      [[ ${1:-} == --strict ]]
      tap=${formula_ref%/*}
      binary=${formula_ref##*/}
      formula_path=$(tap_path "$tap")/Formula/${binary}.rb
      actual_mode=$(file_mode "$formula_path")
      if [[ $actual_mode != "$FAKE_BREW_EXPECT_FORMULA_MODE" ]]; then
        echo "Formula mode $actual_mode, want $FAKE_BREW_EXPECT_FORMULA_MODE" >&2
        exit 8
      fi
    fi
    if [[ ${FAKE_BREW_AUDIT_FAIL:-false} == true ]]; then
      exit 9
    fi
    ;;
  untap)
    tap=${1:?}
    if [[ $tap == "$FAKE_BREW_EXISTING_TAP" ]]; then
      echo "attempted to remove an existing user tap" >&2
      exit 97
    fi
    [[ -f $FAKE_BREW_ROOT/created-tap ]]
    [[ $(<"$FAKE_BREW_ROOT/created-tap") == "$tap" ]]
    [[ -f $FAKE_BREW_ROOT/trusted-formula ]]
    rm -rf -- "$(tap_path "$tap")"
    rm -f -- "$FAKE_BREW_ROOT/trusted-formula"
    ;;
  *)
    echo "unexpected fake brew command: $command_name" >&2
    exit 98
    ;;
esac
