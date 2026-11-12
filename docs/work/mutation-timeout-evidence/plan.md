# Work Plan: Chatwork mutation timeout evidence

- Status: Complete
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Keep the accepted numeric limits and one-attempt rule. Add short tests proving the selected task policies, shared production header/client limits, and the outcome when an earlier caller deadline expires after a mutation reaches the injected doer. Add one build-tagged localhost observation to exercise the actual production HTTP client with delayed headers and delayed response bodies. Run it once and retain its bounded, synthetic output here.

## Alternatives considered

### Increase mutation timeouts from the supplied wall times

Rejected. The aggregate timing can show that uncertainty correlates with calls that ran beyond 20 seconds, but it cannot establish that the provider did not receive the mutation, that a longer wait returns success, or that 20 seconds is otherwise the wrong limit. A timeout extension changes future latency/resource bounds and needs a separate reviewed decision with stronger representative evidence.

### Add a 20-second delay to ordinary tests

Rejected. A long wall-clock case belongs behind an explicit build tag; ordinary gates enforce the contract with short deterministic tests.

## Design

### Public contract

No command, catalog, fault, recovery, or output change. Unknown post-dispatch mutations remain non-retryable and use the existing read-only reconciliation action.

### Layer changes

- Domain: None.
- Application: None.
- Infrastructure: Tests only; no production timeout, client, or retry change.
- CLI and catalog: None.

### Data and control flow

The short test injects a doer that waits until an earlier caller deadline cancels its dispatched mutation request. The opt-in test uses the real production HTTP client against a local handler: one response delays headers and one flushes headers before delaying its body. Both requests use the existing upload task policy, which separates the 60-second task context/client ceiling from the shared 20-second response-header limit.

### Error and cancellation behavior

An early caller deadline before dispatch remains a provable zero-call case under the existing adapter contract. A deadline after `Do` begins remains non-retryable `chatwork_mutation_outcome_unknown`, with one request and no private cause. The delayed-header upload should fail at the shared 20-second response-header limit despite the 60-second upload ceiling; the delayed-body upload should finish successfully at about 20.7 seconds, below its 60-second overall ceiling. These are local transport observations only.

### Security and public boundary

Only fixed synthetic auth and bodies are used. The local-only test checks the synthetic header without printing its value. No real user logs, provider response, or external API response is retained.

## Implementation slices

1. Pin policy and transport fields in fast tests; add an earlier-caller-deadline no-replay boundary case.
2. Add an opt-in build-tag localhost delay observation.
3. Run the ordinary focused tests, then run the opt-in observation once.
4. Record output, scope, and inference limits in `context.md`; make no production timeout change.

## Verification

- Unit and contract tests: `go test ./internal/infra/chatworkapi -run 'TestTransportPolicyIsPinnedAndSingleAttempt|TestMutationCallerDeadlineAfterDispatchIsUnknownWithoutReplay' -count=1`
- Manual observation: `go test -tags=timeout_observation ./internal/infra/chatworkapi -run '^TestObserveProductionTimeoutEdges$' -count=1 -v` exactly once.
- Required profiles: Parent task owns `task check`; this packet adds no production behavior or dependencies.
- Generated-diff or artifact checks: Not applicable.

## Rollout and rollback

No external state or public contract migration. Removing the tagged test or packet cannot change runtime behavior.

## Documentation promotion

The exact interaction between the shared `ResponseHeaderTimeout` and the 60-second upload ceiling is stated in `docs/03_security_model.md`. Numeric policy is unchanged; no thesis or provider contract revision is proposed.
