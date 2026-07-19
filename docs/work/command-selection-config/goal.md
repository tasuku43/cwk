# Work Goal: Persistent command selection

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: [ADR 0003](../../decisions/0003-chatwork-pat-only.md)

## Outcome

Users can curate the exact `cwk` commands presented to and executable by an
agent through `cwk config show` and a line-oriented `cwk config edit` selector.
Commands switched off disappear from every human and machine help projection
and route as unknown before authentication or provider I/O. The setting is a
local attention filter, not an authorization, sandbox, or security boundary.

## Why now

The complete Chatwork surface contains workflows, such as contact-request
management, that many users never delegate. Showing every available operation
adds selection cost and encourages irrelevant discovery even though the full
catalog remains necessary as the product contract.

## Non-goals

- Authorization, access control, policy enforcement, sandboxing, or a claim
  that an agent cannot edit or delete the same local setting.
- Changing Chatwork API permissions, PAT handling, mutation confirmation, or
  provider-side availability.
- Multiple profiles, account-specific or project-local settings, environment
  overrides, shell completion, fuzzy command names, or capability-level rules.
- ANSI/raw-terminal controls, arrow-key navigation, a third-party TUI, or a
  noninteractive flag grammar.
- Removing commands from the complete public catalog, capability ledger, API
  coverage, or release contract.

## Acceptance criteria

- [x] `config show` reports the current source, always-on commands, enabled
  configurable commands, disabled configurable commands, and stale saved paths.
- [x] `config edit` offers catalog-order numbered choices plus `all`, `none`,
  `save`, and `cancel`; invalid lines apply no partial changes.
- [x] `help`, `config show`, and `config edit` are catalog-declared always-on;
  all other exact commands are independently configurable.
- [x] A missing setting preserves the pre-change all-enabled behavior; after
  the first save, newly introduced commands default off until selected.
- [x] Disabled commands disappear from root, namespace, exact, trailing-help,
  agent-help, recovery, and workflow projections and route as the existing
  `unknown_command` before PAT resolution or provider calls.
- [x] Saving rejects a view whose visible required-reference consumers lack a
  reachable producer or whose visible recovery actions point outside the view.
- [x] The setting uses strict, bounded, non-secret platform config storage
  separate from ADR 0003's retired OAuth `config.json`, with Unix rename and
  directory sync and an explicit no-atomicity/durability guarantee on Windows.
- [x] Invalid state never silently enables everything; `config edit` remains a
  deliberate repair path for malformed serialized content and overwrites only
  after explicit `save`, while unsafe storage requires external repair.
- [x] Cancellation, EOF, and context interruption before the save action leave
  the previous file unchanged, including while stdin is blocked; after a
  replacement attempt, uncertain results reconcile through `config show`, and
  confirmed success is not overwritten by late cancellation.
- [x] Documentation and agent-readiness tests state that selection reduces
  cognitive surface but does not weaken or replace existing security controls.
- [x] `task check` passes and the completed change is committed in reviewed
  slices.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1 and 2
- Product contract: supported-outcome promise and public runnable surface
- Architecture: catalog as source of truth and controlled filesystem boundary
- Security: no TTY or local-setting authority assumption; credentials remain
  infrastructure-only
- Existing ADR: ADR 0003 reserves the retired OAuth `cwk/config.json` path

## Completion definition

The work is complete when the catalog-derived command view, local store,
interactive selector, repair behavior, help and routing enforcement, closure
validation, hostile/cancellation tests, governing documentation, and readiness
scenario agree; `task check` succeeds; and the reviewed slices are committed.
