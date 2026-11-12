# Work Goal: Reconcile the current account's messages

- Status: Implemented and reviewed; actual agent evaluation pending
- Retention: temporary
- Retention reason: None
- Governing contract: docs/00_theses.md through docs/04_harness.md and docs/07–09
- Review/delete trigger: Promote reviewed decisions and remove after implementation and verification
- Owner: Main session
- Target: Feedback item 4
- Related ADRs: None

## Outcome

When a message send has an unknown outcome, return exact-body candidates from the currently authenticated account's recent posts, without external filtering or another write.

## Non-goals

- Claim that a candidate proves the original invocation succeeded or absence proves failure.
- Search other accounts, fetch unbounded history, or automatically resend.

## Acceptance criteria

- [x] The currently authenticated account is obtained once through a bounded provider read under the same binding/context, not inferred from names, recipients, or body.
- [x] Only exact-body messages whose typed sender reference equals that account are returned; source count still describes the complete bounded input window.
- [x] Return the canonical account reference and uncertainty; a previous identical post, changed account, propagation delay or inaccessible history cannot prove the outcome.
- [x] Preserve pre-I/O body validation, optional exact reply resolution, secret-free ports and no writes; invalid identity or provider failures do not yield partial results.
- [ ] Propagate and independently review the thesis/contract revision, implement tests and run task check and relevant readiness checks.
