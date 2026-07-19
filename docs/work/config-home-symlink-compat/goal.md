# Work Goal: Config-home symlink compatibility

- Status: Complete
- Owner: Codex
- Target: v0.1.1
- Related ADRs: None

## Outcome

Users whose dotfiles make `~/.config` or `XDG_CONFIG_HOME` a symbolic link can
run `cwk config` and `cwk doctor` without a false `command_selection_unsafe`
failure. `cwk` resolves that existing configuration-home alias to one absolute
real directory before use while retaining strict checks on its own directory
and preference file.

## Why now

A clean Homebrew installation on a macOS dotfiles environment reproduced an
immediate `command_selection_unsafe` failure because `~/.config` was a symbolic
link. The rejected object was the platform configuration-home alias rather
than a `cwk`-owned mutation target.

## Non-goals

- Allowing `${XDG_CONFIG_HOME:-$HOME/.config}/cwk` to be a symbolic link.
- Allowing `command-selection.json` to be a symbolic link, special file, or
  permissively readable Unix file.
- Changing the command-selection schema, selected commands, CLI output, fault
  codes, authentication, or mutation policy.
- Creating or publishing the `v0.1.1` tag or release.

## Acceptance criteria

- [x] An existing absolute configuration home that is a symbolic link to a
  directory supports missing-state load and save/load round trips.
- [x] The resolved target, not the alias path, becomes the stable base used by
  the store.
- [x] Broken/looping configuration-home links fail closed, while `cwk` directory
  and preference-file links remain rejected.
- [x] README installation guidance documents supported dotfiles layouts and
  the retained `cwk`-owned path restrictions.
- [x] Architecture, security, and harness claims distinguish the allowed
  configuration-home alias from rejected owned targets.
- [x] `task check`, `task security`, and `task public:check` pass.

## Governing documents

- Thesis: `docs/00_theses.md`, Axiom 2
- Product contract: command-selection storage and recovery
- Architecture: command-selection adapter
- Security: non-secret command-selection preference boundary
- Existing ADR: none; this narrows an adapter restriction without changing a
  credential or authorization boundary

## Completion definition

The compatibility behavior, negative-path protections, public guidance, and
durable boundary documents agree; required gates pass; and no release or
publication side effect has occurred.
