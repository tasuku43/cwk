# Work Tasks: Chatwork mutation response audit

## Understand

- [x] Read governing thesis, product, architecture, security, harness, API, authentication, and readiness documents.
- [x] Observe current adapter and invoker classifications.
- [x] Record verified facts and external status references in `context.md`.
- [x] Confirm scope and non-goals with root agent.

## Decide

- [x] Compare adapter-level classification with generic invoker whitelisting.
- [x] Confirm precise authoritative rejection set against the fixed API contract: documented 400/401/403/404/429; undocumented 422 and other statuses remain unknown for mutations.
- [x] Confirm shared documentation ownership before policy edits.

## Implement

- [x] Add failing mutation response classification tests.
- [x] Map post-dispatch unproven mutation outcomes to `chatwork_mutation_outcome_unknown`.
- [x] Preserve authoritative rejection faults and known pre-I/O faults.
- [x] Add runtime coverage for exact read-only recovery, no duplicate call, and redaction.
- [x] Preserve a validated nil-error mutation acknowledgement across later cancellation; retain read cancellation suppression.
- [x] Add infrastructure-only pre-dispatch cancellation fault at proven pre-Do boundaries and retain post-Do unknown outcomes.
- [x] Add catalog contract and CLI coverage for retryable same-command recovery only when pre-dispatch is proven.
- [x] Update coordinated security, API contract, and readiness documentation.

## Verify

- [x] Focused adapter and CLI tests pass after concurrent catalog assertions settle. Evidence: `go test ./internal/app/chatworkcmd ./internal/infra/chatworkapi ./internal/cli -count=1` passed on 2026-11-07.
- [ ] Root-selected repository gate passes. Evidence:
- [ ] Shared worktree changes are understood. Evidence:

## Hand off

- [ ] Acceptance criteria have evidence.
- [ ] Durable policy changes are promoted.
- [ ] Temporary packet is removed before final tree.
- [ ] Terra review receives the concrete findings and verification evidence.
