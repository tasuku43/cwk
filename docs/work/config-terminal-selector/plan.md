# Work Plan: Single terminal command selector

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Replace the two config catalog leaves with one fixed-target `config` write
command. Keep domain/application profile semantics and the file adapter. Add a
narrow infrastructure terminal controller for terminal detection, raw-mode
entry/restore, and dimensions; keep key parsing, viewport state, ANSI rendering,
selection validation, and final output in CLI. On Enter, validate the proposed
active view, restore terminal state, and only then invoke the existing mutation
boundary. Extend always-on `doctor` as the read-only command-selection
reconciliation task instead of retaining a second config command.

## Alternatives considered

### Keep `config show` for reconciliation

Rejected because the owner explicitly chose one config command. Read-only
reconciliation belongs to always-on `doctor`, which reports the actual stored
selection state without entering the `config` write workflow.

### Use `help` for reconciliation

Rejected because help describes the active command contract; it does not prove
which profile bytes survived an uncertain replacement. `help` remains
discovery only and is not an uncertain-save next action.

### Keep the line grammar for redirected input

Rejected because it would preserve two interaction contracts under one command
and make terminal state ambiguous. Non-terminal invocation fails explicitly.

### Add a general TUI framework

Rejected because this selector needs five keys, one list, and one bounded
viewport. A framework would add unnecessary presentation and supply-chain
surface.

### Import the terminal dependency from CLI

Rejected. A narrow infrastructure adapter imports pinned
`golang.org/x/term v0.36.0` and `golang.org/x/sys v0.37.0`, both BSD-3-Clause
modules with a Go 1.24 requirement compatible with this Go 1.26.5 project.
`x/term` owns portable terminal state; platform-scoped `x/sys` calls own
context-responsive descriptor reads and Windows console mode. `x/term` already
requires this exact `x/sys` version, so the direct requirement does not add a
second dependency to the module graph. The repository's CLI third-party
allowlist remains empty, so no CLI-import ADR exception is needed.

## Design

### Public contract

- Exact command: `config`
- Role/effect: fixed-target `RoleAct`, `EffectWrite`
- Input: terminal key events from stdin; Enter is the only save confirmation
- Output: one final `saved` or `unchanged` text record; the transient alternate
  screen is human presentation, not a machine data stream
- Row schema: cursor, checkbox, literal `read|create|write` badge, exact command
  path, then optional summary within the remaining visible
  width. Cyan/read, yellow/create, and magenta/write may supplement the badge;
  color is never semantic and red is not an effect color.
- Authentication: none
- Reconciliation: always-on read-only `doctor` reports source, state,
  enabled/disabled counts, and a deterministic `sha256:` identity of the
  catalog-ordered canonical enabled selection; an uncertain save fault retains
  expected `source=saved` and the candidate fingerprint so both can be compared
  with doctor without mistaking an identical missing-profile default for a
  save; scoped agent help fixes the dynamic message grammar used by both text
  and JSON errors
- Compatibility: persisted schema/path remain compatible; old doctor/version
  entries alone are tolerated and normalized on save; old `config show` and
  `config edit` paths are removed rather than retained as hidden aliases

### Layer changes

- Domain: no profile schema change.
- Application: rename edit intent/command to exact `config`; persistence policy
  remains explicit-Enter confirmation.
- Infrastructure: add a bounded terminal adapter using
  `golang.org/x/term v0.36.0` for detection/raw mode/restoration/dimensions and
  platform-scoped `golang.org/x/sys v0.37.0` calls for cancelable descriptor
  waiting/reading and Windows VT output mode; retain the file adapter.
- CLI/catalog: replace command topology, key state machine, viewport renderer,
  terminal lifecycle, always-on classification, help/recovery projections, and
  tests.
- Doctor: keep `EffectRead` and add a bounded command-selection diagnostic with
  source, load state, enabled count, disabled count, and a deterministic
  fingerprint suitable for comparing an uncertain save with intended state.
- Dispatch: derive a four-command always-on recovery island from the complete
  catalog so `doctor`, `version`, `config`, and their scoped help remain
  reachable when active-profile loading fails. Bare root help returns that
  typed load fault rather than presenting a false valid Chatwork view.
- Migration: recognize only legacy `doctor` and `version` allowlist entries,
  ignore them when deriving the active view, and omit them from the next save.

### Error and cancellation behavior

- Non-TTY stdin or stdout fails with a typed fault before raw mode or save and
  points to exact `help config` recovery.
- Raw-mode entry/read/render/restore failures are typed and save zero times.
- `q`/EOF restores and returns `unchanged`; Ctrl-C/context cancellation restores
  and returns the stable canceled fault.
- If the exact current command identity or an item row does not fit, render
  resize guidance and ignore movement, toggle, and save keys; only non-saving
  exit behavior remains active until a usable size is observed. The first
  actionable key after that transition only redraws the exact identity and is
  consumed; only a later input may act on the displayed frame.
- Enter validates the active view first. Invalid dependency/recovery closure
  remains in the TUI and saves zero times. Its complete escaped diagnostic and
  exact current identity must fit in one frame; otherwise only resize guidance
  and a non-saving exit are available. A valid view then restores terminal
  mode, cursor, and screen; restore failure saves zero times; successful restore
  permits exactly one logical save. Raw post-replacement errors remain
  uncertain, retain expected `source=saved` plus the candidate fingerprint, and
  name `doctor` as the read-only reconciliation action;
  confirmed nil success is never reclassified by late cancellation.

### Security and public boundary

TTY state is a presentation prerequisite, never proof of authority. No secret,
provider destination, subprocess, or credential source is added. The terminal
adapter is infrastructure-only, controls only injected descriptors, and is the
sole importer of BSD-3-Clause `golang.org/x/term v0.36.0` and
`golang.org/x/sys v0.37.0`. Both modules require Go 1.24, compatible with this
Go 1.26.5 project; this does not require a CLI allowlist or ADR exception. Unix
uses bounded poll/read calls. Windows confines `ReadFile` to one locked OS
thread, cancels that exact thread through `CancelSynchronousIo`, and joins it
before returning. Neither path leaves a background read that can consume a
later selector invocation's input.

## Implementation slices

1. Contract/work packet and failing catalog/interaction tests.
2. Pinned infra-only terminal adapter and synthetic lifecycle tests.
3. Single-command CLI state machine, exact doctor/version legacy migration,
   always-on failure routing, and catalog routing.
4. Governing docs, readiness scenario, and skill propagation.
5. Independent review, full gates, and commits.

## Verification

- Unit/contract: key parsing, cursor bounds/wrap decision, viewport, exact
  toggle, Enter-only save, effect text badges, catalog-effect/color mapping,
  doctor reconciliation fields/fingerprint, exact doctor/version legacy
  migration, help topology, and absence of old paths.
- Renderer: byte-deterministic ANSI snapshots, ANSI-stripped semantic
  equivalents, narrow-width layouts that retain the full text badge,
  visible-cell width, summary truncation, and bounded viewport output.
- Negative: q/EOF/Ctrl-C/read/render/restore failure and non-TTY save zero times.
- Security: disabled commands still resolve before PAT/provider; re-enable does
  not bypass confirmation.
- Platform: native race tests; blocked-read cancellation followed by a later
  successful read; and explicit Darwin, Linux, and Windows terminal/storage
  cross-compilation with the pinned `x/term` and `x/sys` versions.
- Manual: PTY observation of arrows, Space, Enter, q, and Ctrl-C.
- Required gate: `task check`.

## Rollout and rollback

This is a deliberate pre-1.0 CLI compatibility change. The stored profile is
forward-compatible. Rolling back to the prior binary still reads profiles saved
by the new selector, but doctor/version will be absent from that older binary's
allowlist until re-enabled there. Forward migration accepts only those two
legacy local paths; it does not weaken rejection of other always-on entries.

## Documentation promotion

Update Axiom 2, product command view, architecture terminal boundary, security
TTY/non-authority and cancellation text, harness tests, readiness scenario,
README, and `$add-capability` guidance.
