#!/bin/sh
# Explicit, separate live check; normal repository gates never invoke this file.
set -eu
umask 077

if [ "$#" -ne 1 ] || [ ! -x "$1" ]; then
  printf '%s\n' 'live smoke: 実行可能な cwk のパスを一つ指定してください' >&2
  exit 2
fi
if [ -z "${CWK_API_TOKEN:-}" ]; then
  printf '%s\n' 'live smoke: 専用テストアカウントの CWK_API_TOKEN が必要です' >&2
  exit 2
fi

smoke_binary=$1
smoke_root=$(mktemp -d "${TMPDIR:-/tmp}/cwk-live-smoke.XXXXXXXX")
trap 'rm -rf -- "$smoke_root"' EXIT
trap 'exit 1' HUP INT TERM
mkdir "$smoke_root/cwk"
printf '%s\n' '{"schema_version":1,"enabled_commands":["account show","account status","rooms list"]}' > "$smoke_root/cwk/command-selection.json"
export XDG_CONFIG_HOME="$smoke_root"

# Provider-controlled text and runtime errors may contain private information.
# Only the fixed task name and success/failure are written to the job log.
for smoke_task in 'account show' 'account status' 'rooms list'; do
  case "$smoke_task" in
    'account show') set -- account show ;;
    'account status') set -- account status ;;
    'rooms list') set -- rooms list ;;
  esac
  if "$smoke_binary" "$@" >/dev/null 2>&1; then
    printf 'live smoke: %s OK\n' "$smoke_task"
  else
    printf 'live smoke: %s failed\n' "$smoke_task" >&2
    exit 1
  fi
done
