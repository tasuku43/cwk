# Work Goal: Classify Chatwork mutation responses conservatively

- Status: In progress
- Retention: temporary
- Retention reason: None
- Governing contract: docs/00_theses.md Axiom 7; docs/03_security_model.md Controlled execution boundary; docs/08_external_api_contracts.md Timeout, retry, and idempotency
- Review/delete trigger: Delete after durable conclusions are promoted and the change completes
- Owner: Codex agent
- Target: Terra review
- Related ADRs: None

## Outcome

After a Chatwork mutation request is dispatched, `cwk` reports a definite provider rejection only for a documented response that proves the write was rejected. Every unproven response or transport result is a non-retryable unknown mutation outcome with the command's exact read-only reconciliation action.

## Why now

The application invoker intentionally preserves valid structured faults. `chatworkapi.Client.Execute` currently returns structured response-parsing and unexpected-status faults after a mutation response has arrived, so the caller can mistake an unconfirmed write for a definite failure and repeat it.

## Non-goals

- Change transport attempt counts, timeouts, body ceilings, or provider request construction.
- Change mutation confirmation policy or add another reconciliation UI.
- Change read-task classification or claim that generic timeouts prove a request was not sent.

## Acceptance criteria

- [ ] Post-dispatch malformed, oversized, incomplete, or undocumented mutation responses map to non-retryable `chatwork_mutation_outcome_unknown`.
- [ ] Documented Chatwork rejection statuses retain their stable classifications; undocumented statuses remain unknown for mutations.
- [ ] Transport failure and cancellation after dispatch remain non-retryable unknown outcomes with no raw cause or provider body leakage.
- [ ] Tests prove one and only one provider request, exact read-only catalog recovery, and no replay recommendation.
- [ ] Existing pre-I/O validation, authentication, policy, and provider rejection behavior remains appropriately classified.
- [ ] Focused tests and repository gates selected by the root task pass.

## Governing documents

- Thesis: `docs/00_theses.md`, Axiom 7
- Product contract section: `docs/01_product_contract.md`, Side effects and Mutation confirmation policy
- Architecture or security invariant: `docs/03_security_model.md`, Controlled execution boundary
- Existing ADR: None

## Completion definition

Acceptance criteria have evidence, the policy wording and harness enforcement have been coordinated with concurrent documentation work, focused tests pass, and this temporary packet leaves the final tree.
