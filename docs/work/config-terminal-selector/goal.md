# Work Goal: Single terminal command selector

- Status: Accepted
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: [ADR 0003](../../decisions/0003-chatwork-pat-only.md)

## Outcome

`cwk config` opens one focused terminal selector. Up/down moves the cursor,
Space toggles the selected Chatwork command, Enter persists the complete
selection, and quitting or closing before Enter leaves the last saved profile
unchanged. Human-facing selector output contains no repeated purpose/schema
preamble. `doctor`, `version`, `config`, and scoped help for those always-on
local commands remain available after a saved-profile load failure; bare root
help fails with that typed fault instead of resembling a valid empty view.
`doctor` is the read-only reconciliation task for command-selection state.

## Why now

The first line-oriented `config show`/`config edit` slice proved the catalog and
persistence boundaries but exposed a numbered-command interaction that requires
more reading and typing than a local preference selector needs. The owner has
selected direct cursor navigation and one exact command as the intended UX.

## Non-goals

- Treating command selection as authorization, sandboxing, provider scope, or
  mutation approval.
- Changing the JSON schema, XDG/AppData location, missing-file all-enabled
  default, or exact-path allowlist semantics for Chatwork task commands.
- Adding profiles, project-local configuration, fuzzy search, mouse input,
  command grouping, bulk presets, or a general-purpose TUI framework.
- Changing Chatwork API tasks, output presentation, authentication, references,
  effects, or confirmation policies.
- Making terminal interaction available through redirected/non-terminal input.

## Acceptance criteria

- [x] `cwk config` is the sole public command-selection command; `config show`
  and `config edit` are absent from routing and every help projection.
- [x] On an interactive terminal, Up/Down moves a visible cursor, Space toggles
  exactly one choice, and Enter is the only key that starts persistence.
- [x] ASCII Space and fragmented UTF-8 U+3000 full-width space produce the same
  single toggle, so a Japanese terminal input method does not make the
  advertised interaction inert.
- [x] Every selectable row displays its literal `read`, `create`, or `write`
  effect badge. Cyan/read, yellow/create, and magenta/write are supplemental
  cues only; the badge remains sufficient without color, and red is not used
  to imply destructiveness that the effect contract does not establish.
- [x] Quitting, EOF/terminal closure, Ctrl-C, or context cancellation before
  Enter restores terminal mode and leaves the prior profile byte-for-byte
  unchanged.
- [x] The selector starts from the last saved state, uses a bounded viewport,
  and restores the prior screen/cursor state on every exit path.
- [x] A resize-only frame cannot authorize the key that first observes usable
  dimensions: that key only redraws the exact current identity, and a later
  input is required to toggle or save.
- [x] The interactive screen omits the old
  `config purpose=attention-only security-boundary=false source=...` record;
  the non-security contract remains in exact help, agent help, and docs.
- [x] `help`, `doctor`, `version`, and `config` are always-on catalog leaves;
  only Chatwork task leaves are selectable.
- [x] `doctor` reports command-selection source, state, enabled and disabled
  counts, and a deterministic `sha256:` fingerprint without entering a write
  workflow; uncertain saves name the expected `source=saved` plus candidate
  fingerprint and use `doctor`, never `help`, as their reconciliation task.
- [x] A malformed, unsafe, unavailable, or otherwise unreadable active profile
  keeps `doctor`, `version`, `config`, and always-on scoped help reachable while
  bare root help returns the typed load fault instead of masquerading as a valid
  empty or all-enabled Chatwork view.
- [x] Existing profiles containing formerly selectable `doctor` or `version`
  remain loadable and are normalized on the next Enter save. No other
  always-on path is accepted through this legacy migration.
- [x] A non-terminal invocation fails predictably before persistence with an
  exact typed fault and help recovery path.
- [x] Enter follows the exact order validate selection, restore terminal, then
  invoke one save; failed validation or restoration performs zero saves.
- [x] A failed-validation notice is actionable only when its complete escaped
  text and the exact current identity fit in the same frame; otherwise width or
  height constraints show resize guidance and admit only a non-saving exit.
- [x] Required-reference and recovery closure validation, disabled-route
  zero-PAT/provider-I/O behavior, malformed-content repair, unsafe-storage
  refusal, and phase-sensitive mutation outcome handling remain intact.
- [x] Human/agent help, README, theses, product, architecture, security,
  harness, readiness evidence, and `$add-capability` agree on the new command.
- [x] The terminal boundary uses pinned `golang.org/x/term v0.36.0` and
  `golang.org/x/sys v0.37.0` only in infrastructure. Both are BSD-3-Clause Go
  1.24 modules compatible with this Go 1.26.5 project; their terminal-only use
  requires no CLI import allowlist or ADR exception.
- [x] Canceling a blocked terminal read returns promptly without leaving a
  background reader that can consume input from a later invocation; native
  tests and Darwin/Linux/Windows builds cover the platform implementations.
- [x] Deterministic renderer tests cover ANSI structure, ANSI-stripped semantic
  output, narrow terminal widths, badge/path preservation, resize-redraw
  consumption, complete-notice gating, blocked unseen mutations, and bounded
  viewport behavior.
- [ ] Focused tests, Unix race tests, Unix and Windows cross-compilation, and
  `task check` pass; independent reviewers report no unresolved P1/P2 findings.
- [ ] The change is committed in intentional slices and the worktree is clean.

## Governing documents

- Thesis: `docs/00_theses.md`, Axiom 2
- Product contract: user-selected command view
- Architecture: catalog as public source of truth and controlled terminal/file
  boundaries
- Security: command selection is not authority; mutation cancellation is
  phase-sensitive
- Existing ADR: ADR 0003 reserves the retired OAuth configuration path

## Completion definition

Stop when the single-command terminal interaction, catalog migration,
persistence/cancellation behavior, tests, durable documentation, and readiness
scenario agree; the required checks pass; independent review is clear; and the
reviewed changes are committed. Do not expand into additional TUI features.
