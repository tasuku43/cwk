# Work Tasks: Chatwork mutation timeout evidence

## Understand

- [x] Read governing thesis, product, architecture, security, harness, external API, and readiness contracts.
- [x] Read the supplied feedback attachment's #4 only.
- [x] Inspect current `CallPolicy`, task context, HTTP client, and response-header limits.
- [x] Record current facts and unknowns without copying private session history.

## Decide

- [x] Preserve one-attempt, 20-second ordinary-task, and 60-second upload policies pending stronger evidence.
- [x] Keep delayed runtime observation opt-in and bounded outside ordinary tests.
- [x] Clarify the shared header-timeout interaction with upload timing.

## Implement

- [x] Add short tests for selected task timeouts, response-header/client timeout fields, and single-attempt policy.
- [x] Add a short caller-deadline-after-dispatch test proving unknown outcome and no replay.
- [x] Add one opt-in local delayed-header/delayed-body observation using only synthetic inputs.
- [x] Preserve production timeout and retry values.

## Verify

- [x] Focused ordinary adapter tests pass. Evidence: `go test ./internal/infra/chatworkapi -run 'TestTransportPolicyIsPinnedAndSingleAttempt|TestMutationCallerDeadlineAfterDispatchIsUnknownWithoutReplay' -count=1` passed.
- [x] Opt-in synthetic observation completes once and prints bounded timings. Evidence: raw output and limits are recorded in `context.md`; the test ran once.
- [x] No test call reaches Chatwork; only localhost fixture counters increment. Evidence: the opt-in server used `httptest.Server`, asserted the synthetic PAT header, and recorded one request per route.
- [x] Shared diff and worktree remain understood. Evidence: this task changed only the policy tests, the build-tagged synthetic observation, the exact timeout clarification, and this retained evidence packet; no production timeout value or retry policy changed.

## Hand off

- [x] Report what synthetic timing demonstrates and what it cannot establish.
- [x] Report the unresolved absence of provider-side timing evidence.
- [x] Keep the evidence packet only for its stated review trigger.
