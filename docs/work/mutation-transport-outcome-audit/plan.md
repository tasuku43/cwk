# Work Plan: Chatwork mutation response audit

- Status: Proposed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Keep generic invocation and pre-I/O fault semantics intact. In the infrastructure adapter, once `http.Do` has been called for a mutating task, convert malformed/unbounded/unexpected/unmapped response outcomes to the existing `chatwork_mutation_outcome_unknown`. Preserve only documented Chatwork provider rejections. Unrecognized status codes are unknown for mutations, while read classification remains unchanged.

## Alternatives considered

### Classify every structured fault as unknown after the application action starts

Rejected because the action callback also contains known pre-I/O validation, authentication, and binding failures. Conflating them would misstate zero-call rejection behavior and break valid recovery.

### Change the generic mutation invoker to whitelist Chatwork fault codes

Rejected because the reusable application boundary must not own provider-specific semantics. The Chatwork adapter owns transport classification; application and CLI preserve its typed result.

## Design

### Public contract

No command or output schema change. Existing mutations continue to expose `chatwork_mutation_outcome_unknown`, `retryable: false`, with the single catalog-declared read-only reconciliation action. Provider rejections retain their current specific stable faults.

### Layer changes

- Domain: None.
- Application: Keep the invoker's structured-fault-first contract; add coverage only if an application-side result-validation hole is confirmed.
- Infrastructure: Classify post-`Do` mutation response failures as unknown; recognize documented provider rejection statuses only.
- CLI and catalog: Preserve existing exact recovery declaration; update catalog availability only if response faults cease to be possible on mutation commands.

### Data and control flow

Validated task -> one `http.Do` call -> either documented provider rejection, confirmed valid success, or non-retryable unknown outcome -> application strips private cause -> CLI attaches catalog recovery and renders structured fault. No retry path is introduced.

### Error and cancellation behavior

Pre-HTTP input/authentication/binding failures retain existing classifications. `Do` failures (including cancellation/deadline) and unproven response failures after dispatch become unknown. Documented provider rejections retain their stable classification. Confirmed valid response is success even if cancellation is observed later. No raw cause/body is emitted.

### Security and public boundary

No new asset, destination, dependency, or fixture. Synthetic local HTTP server responses exercise response boundaries. No HTTP trace is used to claim request delivery or non-delivery.

## Implementation slices

1. Contract and failing tests
2. Infrastructure response classification
3. CLI recovery/no-leak assertion and any catalog accuracy adjustment
4. Coordinated documentation and final audit handoff

## Verification

- Unit and contract tests: Adapter status/body matrix and mutation invoker tests.
- Negative side-effect tests: Exactly one doer call; no second request/retry.
- Opaque-reference and complete-pagination tests: Not applicable.
- Structured output, hostile-output, and recovery tests: Verify exact read-only next command and no body/cause leakage.
- Agent-readiness scenario and discovery-round-trip count: No change to task discovery; recovery metadata remains exact.
- Manual observation: Inspect diff and shared worktree after concurrent docs work completes.
- Required profiles: Focused adapter/CLI tests, then root agent's required gate.
- Generated-diff or artifact checks: Not applicable.

## Rollout and rollback

No external state or migration. A stricter unknown-outcome classification may require read-only reconciliation where the earlier response fault suggested generic help; this is the safe behavior for an unconfirmed write.

## Documentation promotion

Coordinate a precise distinction between post-dispatch response-integrity failures and authoritative provider rejections in docs/03, docs/08, docs/04, docs/09, and `.agents/skills/add-capability/SKILL.md`; do not edit shared documents until the current contributor confirms those paths are available.
