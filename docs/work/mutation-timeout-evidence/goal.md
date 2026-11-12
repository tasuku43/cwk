# Work Goal: Reassess Chatwork mutation timeout evidence

- Status: Complete
- Retention: evidence
- Retention reason: Preserve the exact output of one opt-in, 20.7-second synthetic localhost run. Git records the reproducible test source, but not this historical scheduler-dependent timing observation, which helps future reviews avoid reading the test as a provider-delivery or timeout-optimality benchmark.
- Governing contract: `docs/00_theses.md` Axiom 7; `docs/03_security_model.md` Chatwork timeout and mutation policy; `docs/04_harness.md` failure handling; `docs/08_external_api_contracts.md` Timeout, retry, and idempotency
- Review/delete trigger: Revisit when an approved transport-timeout or retry-policy change is proposed. Delete this packet when that decision is recorded and the historical synthetic timing observation no longer provides evidence that cannot be recovered from durable tests and Git history.
- Owner: Codex agent
- Target: Terra review
- Related ADRs: None

## Outcome

The current 20-second mutation limit is reassessed against the supplied aggregate observations and one synthetic localhost observation. The code and durable contract state exactly which deadline applies; the review does not infer provider delivery, rollback, or an optimal replacement timeout from wall time alone.

## Why now

The supplied feedback reports slow `messages send` calls that ended with `chatwork_mutation_outcome_unknown` and asks whether the 20-second limit is adequate. Once an unsafe mutation may have been dispatched, neither a timeout nor a follow-up absence check proves it was not applied. A bounded observation is needed before changing the numeric limit.

## Non-goals

- Change production timeouts, attempt count, idempotency, or mutation recovery.
- Contact Chatwork or use credentials, private session logs, or personal data.
- Claim that a longer timeout would make Chatwork apply a write or return success.
- Convert generic post-dispatch uncertainty into a retryable or definitely-not-sent result.

## Acceptance criteria

- [x] Pin the 20-second per-task context for ordinary mutations, 60-second upload context/client ceiling, shared 20-second response-header timeout, and one-attempt policy with short tests. Evidence: `TestTransportPolicyIsPinnedAndSingleAttempt`.
- [x] Prove a controlled caller deadline after a mutation enters the transport returns non-retryable unknown outcome after one call. Evidence: doer-entry handshake in `TestMutationCallerDeadlineAfterDispatchIsUnknownWithoutReplay`.
- [x] Preserve one opt-in, bounded local delayed-header/delayed-body observation outside ordinary tests; run it once, with synthetic requests only. Evidence: `timeout_observation_test.go` ran once using `httptest.Server`.
- [x] Record raw timing results and explain their limited inference; do not claim provider delivery or timeout optimality. Evidence: raw output and limits in `context.md`.
- [x] Preserve production timeout values unless new evidence supports a reviewed policy decision. Evidence: no production timeout or retry value changed.
- [x] `docs/03_security_model.md` explains the shared response-header cap, including its interaction with uploads.

## Governing documents

- Thesis: `docs/00_theses.md`, Axiom 7
- Product contract section: `docs/01_product_contract.md`, Side effects and mutation recovery
- Architecture or security invariant: `docs/03_security_model.md`, Chatwork timeout and mutation policy
- Existing ADR: None

## Completion definition

The current limits, early caller-deadline behavior, no-replay guarantee, and synthetic timeout-edge observations have evidence; the durable security contract is precise; no live provider request or timeout change was made; and the retained evidence packet names its review/delete trigger.
