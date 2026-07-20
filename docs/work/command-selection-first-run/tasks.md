# Work Tasks: Require first-run command selection

## Understand

- [x] Read governing theses, product, architecture, security, and harness sections.
- [x] Reproduce current behavior in an empty temporary configuration home.
- [x] Record verified facts and constraints in `context.md`.
- [x] Record thesis evidence and the public outcome.

## Decide

- [x] Compare credible approaches and record the selected design.
- [x] Identify compatibility, effect, and trust-boundary impact.
- [x] Obtain product-owner approval for the control-plane-only first-run state.

## Implement

- [x] Add failing contract and negative-path tests.
- [x] Implement active-view, routing, help, and diagnostic behavior.
- [x] Update harness enforcement and durable documentation.
- [x] Update README.

## Verify

- [x] Focused tests pass. Evidence: `go test ./internal/cli ./internal/app/configcmd`
- [x] `task check` passes. Evidence: full gate completed with Go 1.26.5.
- [x] Runtime behavior is observed with an empty temporary configuration home. Evidence: root/agent help, `rooms list`, and `doctor` were invoked with an isolated `XDG_CONFIG_HOME`.
- [x] Generated diff and repository status are understood. Evidence: no generated files changed; the worktree contains only this capability slice.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions are promoted out of the work packet.
- [x] Temporary diagnostics and sensitive artifacts are removed.
- [x] Handoff summarizes outcome, checks, and remaining risks.
