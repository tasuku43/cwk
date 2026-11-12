# Work Context: Chatwork mutation timeout evidence

## Current behavior

- `internal/infra/chatworkapi/transport.go` sets `RequestTimeout` to 20 seconds, `UploadTimeout` to 60 seconds, and `MaxAttempts` to one.
- `CallPolicy` selects the 20-second timeout for ordinary tasks, including messages and room-task mutations, and the 60-second timeout for file upload. `Client.Execute` derives the task context from that policy.
- `productionHTTPClient` sets `http.Transport.ResponseHeaderTimeout` to 20 seconds for every request, `http.Client.Timeout` to 60 seconds, and refuses redirects. The response-header timeout bounds waiting for response headers after the request body has been sent; it also applies to uploads.
- After `http.Do` starts, a mutation transport error is an unknown, non-retryable outcome. This class does not prove whether the server applied the write.
- The existing one-attempt contract and read-only reconciliation for unknown mutation outcomes remain unchanged.
- The supplied feedback attachment motivates reassessment by reporting slow sends ending in `chatwork_mutation_outcome_unknown` and a possible provider-side write delay. Its raw session records and aggregate counts are intentionally omitted from this repository because they derive from private usage history. The attachment's observations do not identify which exact timeout edge ended each historical call or establish what a longer timeout would do.

## Relevant structure

- Infrastructure boundary: `internal/infra/chatworkapi/transport.go`, `Client.Execute`, `CallPolicy`, and `productionHTTPClient`
- Existing adapter tests: `internal/infra/chatworkapi/client_test.go`
- The caller-deadline boundary test uses a controlled test context and waits for a doer-entry handshake before signaling `context.DeadlineExceeded`. This avoids a short wall-clock timer expiring during request validation or construction on a slow CI worker.
- Opt-in synthetic observation: `internal/infra/chatworkapi/timeout_observation_test.go`
- Durable contract: `docs/03_security_model.md`, Chatwork timeout and mutation policy

## Constraints

- No request goes to Chatwork; the observation uses only `httptest.Server` on localhost, a fixed synthetic PAT, and a synthetic upload body.
- Ordinary tests must remain short. The delayed observation is selected only by `-tags=timeout_observation` and has a single fixed ~20.7-second delay.
- The observation starts two synthetic upload requests concurrently in one test run, one with delayed headers and one with immediate headers followed by a delayed body, so the run lasts about 20.7 seconds rather than serializing two waits.
- A successful synthetic delayed-body upload validates only the current local timeout wiring. It cannot establish Chatwork behavior, remote mutation delivery, or a better production timeout.
- A delayed-header mutation error is non-retryable unknown because dispatch has begun. It does not claim the mutation was rejected or not applied.

## Unknowns

- [ ] The supplied aggregate history cannot identify which exact transport edge (overall task context, shared response-header wait, caller deadline, or another event) ended each individual historical call.
- [ ] No representative provider-side latency distribution or provider guarantee establishes that raising the mutation timeout would improve delivery or success rate.

## Thesis evidence

- Repeated design decision or point of agent confusion: A wall-time timeout after a mutation call is not proof of non-delivery; reconcile through read-only evidence before another write.
- User outcome or friction observed in the minimal slice: The supplied feedback reports send attempts ending with uncertain outcomes and subsequent caller-side checks before any replay decision; its private aggregate counts are not retained here.
- Code workaround or exception being considered: None. No timeout extension or retry is introduced.
- Current thesis that resolves it, or proposed thesis revision: Axiom 7 and the mutation boundary already require unknown outcomes to be non-retryable and reconciled with a read-only command.
- Downstream product, architecture, security, Skill, catalog, and harness impact: No product, catalog, mutation recovery, or attempt-count change. Security wording clarifies the already-enforced shared header timeout; boundary tests enforce timeout selection and early-caller-deadline classification.

## Reproduction or observation

Run once, only when explicitly reviewing this evidence:

```sh
go test -tags=timeout_observation ./internal/infra/chatworkapi -run '^TestObserveProductionTimeoutEdges$' -count=1 -v
```

The test sends two concurrent multipart uploads to an in-process localhost server. The server delays one response's headers by 20.7 seconds and flushes the other response headers immediately, then delays its body by 20.7 seconds. It prints the selected policy, observed durations, classified outcomes, and one request count per route. It has a 70-second caller cap above the selected 60-second upload context and 60-second HTTP client timeout. Raw observed values are recorded after the single run below.

Observed once on 2026-11-08 with Go 1.26.10 on Darwin arm64:

```text
selected upload policy=1m0s attempts=1 client_timeout=1m0s shared_response_header_timeout=20s delay=20.7s
delayed-header elapsed=20s outcome=error retryable=false requests=1
delayed-body elapsed=20.7s outcome=confirmed-success requests=1
PASS (test body 20.71s; go test 20.898s)
```

The delayed-header route demonstrates that the shared 20-second response-header limit wins even under the upload task's 60-second context/client ceiling. The delayed-body route demonstrates that an upload response body arriving after 20 seconds but before 60 seconds can complete under this local fixture. This does not test the ordinary mutation's 20-second task context against a delayed body, does not exercise Chatwork, and does not show that a real provider write was delivered or that extending message/task mutation timeouts would help.

## Security and public-boundary notes

- Assets and side effects involved: Synthetic request/response timing only; no external mutation.
- Credentials or confidential data involved: Fixed synthetic PAT used only in the local request header; not printed or persisted.
- New dependencies, destinations, files, processes, or generated content: No dependencies or remote destinations. One build-tagged opt-in test and this evidence packet are retained.
- External schema provenance, publication rights, and drift evidence: Not applicable; fixtures are synthetic and are not published provider schemas.
- Pagination, timeout, retry, idempotency, and cancellation facts: One attempt; 20-second ordinary task context and shared response-header timeout; 60-second upload task/client ceiling; earlier caller deadlines still win. Post-dispatch write timeouts remain unknown and non-retryable.
- Publication and licensing concerns: None.
