# Work Tasks: Preserve multiple explicit message replies

## Understand

- [x] Read governing and external API contracts.
- [x] Trace the single-reply assumption across all four layers.
- [x] Record the black-box evidence and semantic failure.
- [x] Obtain the requested one-or-many output design.

## Implement

- [x] Add failing multi-reply parser and projection tests.
- [x] Replace the single reply slot with ordered typed replies.
- [x] Resolve, select, and recursively complete every reply edge.
- [x] Preserve single-reply text and add list-valued multi-reply text.
- [x] Update agent contract, durable docs, harness, and skill guidance.

## Verify

- [x] Focused tests pass. Evidence: parser, domain, application closure/selection, capsule, CLI, and exact-help packages pass.
- [x] `task check` passes. Evidence: full gate on 2026-07-20, including security/public checks and vulnerability scan.
- [x] Synthetic end-to-end output is observed. Evidence: capsule fixture proves `reply=[#1,#2] to=[a1,a2]`, zero unresolved/unknown; mixed fixture proves `reply=[#1,?999]` and one unresolved edge.
- [x] Worktree and generated diff are understood. Evidence: `git diff --check`, repository-wide old single-field search, and full gate pass.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions are promoted out of the work packet.
- [x] Temporary diagnostics and sensitive artifacts are removed.
- [x] Handoff summarizes compatibility and verification.
