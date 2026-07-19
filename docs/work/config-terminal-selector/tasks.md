# Work Tasks: Single terminal command selector

## Understand and decide

- [x] Read governing documents and `$add-capability`.
- [x] Observe the current line selector, command topology, persistence, help,
  and always-on behavior.
- [x] Fix outcome, non-goals, completion boundary, compatibility, and
  non-security semantics before implementation.
- [x] Complete parallel architecture, terminal, and contract audits.
- [x] Select infrastructure-only `golang.org/x/term v0.36.0` and
  `golang.org/x/sys v0.37.0`; both are BSD-3-Clause, require Go 1.24, are
  compatible with this Go 1.26.5 project, and require no CLI allowlist/ADR
  exception. Record that `x/term` already requires that exact `x/sys` version.

## Implement contract and terminal boundary

- [x] Add failing tests for one exact `config` command and always-on local
  operations.
- [x] Add terminal detection/raw/restore/size plus context-responsive read
  adapter with Darwin/Linux/Windows build coverage and no abandoned reader
  goroutine.
- [x] Add deterministic key parsing, bounded viewport, and terminal lifecycle
  tests using synthetic streams.
- [x] Render a literal read/create/write badge on every selectable row, with
  cyan/yellow/magenta as optional non-semantic cues and no red effect mapping.
- [x] Add deterministic ANSI, ANSI-stripped semantic, narrow-width,
  visible-cell, truncation, and badge-preservation renderer tests.
- [x] Add typed non-TTY rejection and prove every pre-Enter exit restores the
  terminal and saves zero times.

## Implement selector and migration

- [x] Replace `config show/edit` with `config` in app intent, catalog, routing,
  help, errors, and recovery.
- [x] Make Up/Down/Space/Enter/q/Ctrl-C/EOF behavior exact; Enter must validate,
  then restore, then save exactly once.
- [x] Remove the purpose/security/source presentation header.
- [x] Keep doctor/version/config and always-on scoped help reachable when
  active-profile load fails; make bare root help return the load fault rather
  than present a false normal Chatwork view.
- [x] Add read-only doctor reconciliation with source, state, enabled/disabled
  counts, and deterministic versioned SHA-256 fingerprint; require both
  `source=saved` and the candidate fingerprint for uncertain-save
  reconciliation, and never use help as that action.
- [x] Publish and test the uncertain fault's exact message grammar in scoped
  agent help and JSON output.
- [x] Derive the profile-failure recovery island and always-on screen facts from
  catalog metadata; block toggle/save when exact current identity is not shown,
  consume the first actionable key that only redraws after resize, and require
  a complete validation notice plus exact identity before reenabling mutations.
- [x] Tolerate and normalize only legacy saved `doctor`/`version` entries;
  retain rejection of all other always-on selections.
- [x] Preserve dependency/recovery closure, repair, storage, and zero-I/O
  enforcement.

## Verify and document

- [x] Update semantic/catalog fixtures, human/agent help, readiness, hostile
  output, cancellation, and migration tests.
- [x] Update theses, product, architecture, security, harness, README, work
  packet, capability skill, and dependency evidence.
- [x] Record x/term v0.36.0 and x/sys v0.37.0, their BSD-3-Clause licenses, Go
  1.24 minimum, Go 1.26.5 compatibility, exact dependency relationship, and
  platform-scoped uses in the work packet.
- [x] Run focused tests, race, Unix/Windows cross-builds, and PTY manual checks.
- [x] Obtain independent architecture/security/UX review and resolve P1/P2.
- [ ] Run `task check` on the final clean-tree candidate.
- [ ] Mark this packet complete and commit reviewed slices.
