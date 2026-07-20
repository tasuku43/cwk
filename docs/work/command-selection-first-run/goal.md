# Work Goal: Require first-run command selection

- Status: Complete
- Owner: Product owner
- Target: Current development cycle
- Related ADRs: None

## Outcome

An installation without a saved command-selection profile exposes only the local control commands and clearly explains in human help that `config` has not been completed. Attempting a known Chatwork task before configuration fails before PAT resolution or provider I/O with a stable `command_selection_required` recovery to `config`.

## Why now

The current missing-profile default exposes all Chatwork commands. That defeats the selector's primary purpose during the first agent interaction: reducing irrelevant command context, token consumption, and command-selection mistakes.

## Non-goals

- Add a non-interactive configuration grammar.
- Change Chatwork credentials, permissions, or mutation confirmation.
- Treat command selection as authorization or sandboxing.
- Change the behavior of a deliberately saved empty selection.

## Acceptance criteria

- [x] Missing profile derives an active view containing only `help`, `doctor`, `version`, and `config`.
- [x] Root human help clearly says that `config` is not set and that only control commands are currently shown, with the token-efficiency reason and exact `cwk config` next step.
- [x] A known configurable command or its scoped help fails as `rejected` / `command_selection_required` before PAT resolution or provider I/O.
- [x] An actually unknown command remains `unknown_command`; a saved disabled command retains the existing `unknown_command` behavior.
- [x] `config` opens with all current Chatwork commands selected and saves only on Enter; non-TTY behavior remains explicit.
- [x] `doctor` reports the missing profile as an unconfigured state rather than a valid all-enabled default.
- [x] Theses, product, architecture, security, harness, and README agree with the new first-run contract.
- [x] `task check` passes.

## Governing documents

- Thesis: Axiom 2, bounded command discovery and shared active view
- Product contract section: User-selected command view
- Architecture or security invariant: Catalog as the public source of truth; command-selection preference
- Existing ADR: None

## Completion definition

The work is complete when the first-run state, routing, help, diagnostics, documentation, and negative-path tests agree and `task check` passes.
