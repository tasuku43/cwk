# Work Context: Chatwork mutation transport outcomes

## Current behavior

- Before the change, `internal/app/execution.Invoker` called a mutation action once and preserved any valid structured `fault.Error` after the action returned; raw or invalid errors became `unclassified_mutation_outcome`.
- Before the change, `internal/infra/chatworkapi.Client.Execute` mapped `http.Do` errors and mutation `5xx` responses to non-retryable `chatwork_mutation_outcome_unknown`.
- Before the change, nil response/body, unexpected successful statuses, disallowed `204`, bounded-body read/size failures, and response decoding/mapping failures after a successful `http.Do` returned structured response faults. The invoker preserved them.
- Before the change, `providerFault` mapped known `400`, `401`, `403`, `404`, and `429` responses to stable rejection faults, but also mapped `422` to `chatwork_invalid_request` even though current official Chatwork endpoint contracts list `400` rather than `422`; other unrecognized non-5xx statuses became `chatwork_unexpected_response`.
- After the change, every dispatched mutation transport failure and every unproven response/result failure maps to cause-free, non-retryable `chatwork_mutation_outcome_unknown`. Only the documented rejection set retains provider-specific classifications. The one-attempt policy is unchanged.
- After the change, application-side `chatwork_result_invalid` and `chatwork_result_mismatch` faults observed after the mutation task call begins are collapsed at the CLI mutation boundary to `unclassified_mutation_outcome`, with the existing read-only reconciliation action.
- Mutation catalog entries already declare `chatwork_mutation_outcome_unknown` with a single read-only reconciliation command. Read-side response parsing faults and `chatwork_unavailable` have separate declarations.
- Terra review found two phase-boundary defects: the application service could overwrite a nil-error mutation acknowledgement with later `ctx.Err()`, and generic application cancellation after entering the service was treated like HTTP dispatch even when request construction ended before `Do`.
- After the follow-up fix, mutation services validate and preserve nil-error port results despite later cancellation; read tasks still suppress results. Mutation cancellation is passed to infrastructure, which emits retryable `chatwork_mutation_not_dispatched` only at adapter entry, canceled body-file/request construction, or the final context check immediately before `Do`. Cancellation or failure once `Do` begins remains non-retryable unknown; no httptrace or timeout inference is used.
- Catalog recovery for `chatwork_mutation_not_dispatched` names the same exact command and is retryable only for that typed proof. Generic `operation_canceled` remains scoped-help recovery; `chatwork_mutation_outcome_unknown` retains the exact read-only reconciliation action.
- The production transport policy is pinned to one attempt; no timeout or attempt changes are in scope.

## Relevant structure

- Entry point: `internal/cli/chatwork.go:runChatwork`
- Domain rule: `internal/domain/operation` intent/impact, `internal/domain/fault`
- Application use case: `internal/app/execution.Invoker`; `internal/app/chatworkcmd.Service`
- Infrastructure boundary: `internal/infra/chatworkapi/transport.go:Client.Execute`, `providerFault`, `transportFault`
- CLI catalog or presentation: `internal/cli/chatwork_catalog.go`
- Existing tests and harness checks: `internal/app/execution/invoker_test.go`, `internal/infra/chatworkapi/client_test.go`, `internal/cli/chatwork_adapter_contract_test.go`, `internal/cli/chatwork_runtime_test.go`, `docs/04_harness.md`

## Constraints

- Preserve the single application mutation invocation and one provider attempt.
- Do not infer "not sent" from a generic timeout, cancellation, or HTTP trace.
- Preserve only structured faults that have contractually proven rejection or known pre-I/O semantics; the new `chatwork_mutation_not_dispatched` code is constructed only by infrastructure before `Do`.
- Keep provider bodies and raw causes private.
- The shared worktree has concurrent changes in docs/00–04, docs/09, and the add-capability skill. Coordinate policy edits with the root agent and do not overwrite those changes.

## External facts

- Chatwork, “エンドポイントについて” (`https://developer.chatwork.com/docs/endpoints`), checked 2026-11-07: the API documents `429` for exceeding the general request limit and for the combined room-posting limit; the exact combined-room error text is already pinned by repository policy.
- Chatwork endpoint references checked 2026-11-07 include documented `400` invalid-input, `401` authentication, `403` permission, `404` missing-target, and `429` rate-limit responses. Examples: `https://developer.chatwork.com/reference/put-rooms-room_id-messages-message_id` and `https://developer.chatwork.com/ja/reference/delete-incoming_requests-request_id`.
- No external response content is copied into fixtures or output.

## Unknowns

- [x] Confirm with the root agent that shared catalog/policy edits require coordination with the concurrent documentation pass.
- [x] Confirm that application-side mutation result-validation faults are unproven after dispatch and route them to unknown-outcome handling.

## Thesis evidence

- Repeated design decision or point of agent confusion: A valid structured fault is not sufficient evidence that a mutation was rejected; after dispatch, the adapter's classification must prove that semantic distinction.
- User outcome or friction observed in the minimal slice: A 2xx write response whose body is malformed/oversized can currently produce a normal structured contract fault whose recovery points to help, leaving repetition safety unclear.
- Code workaround or exception being considered: None; fix the shared Chatwork adapter response classifier.
- Current thesis that resolves it, or proposed thesis revision: Axiom 7 already states that uncertain post-mutation outcomes are non-retryable and reconcile read-only. Clarify its implementation wording rather than changing the thesis.
- Downstream product, architecture, security, Skill, catalog, and harness impact: Infrastructure mapping and mutation outcome tests; contract wording in docs/08, security/architecture/harness as coordinated; catalog fault code already exists.

## Reproduction or observation

```sh
go test ./internal/infra/chatworkapi ./internal/cli -run 'TestMutation|TestRunChatworkUnknownTransportOutcome|TestRunChatworkRejectsWrongResultVariant' -count=1
```

The original focused response-outcome tests proved malformed, oversized, truncated, unexpected-status, invalid-response, wrong-target, generic failure, and post-Do cancellation outcomes are unknown; provider rejections remain stable; only one provider mutation request occurs; exact bindings are used; mismatched bindings make zero requests; and raw body/cause canaries do not reach public faults. Terra review then reproduced the exact P1 race in `chatworkcmd.Service.executeProvider` and the P2 pre-Do misclassification. Follow-up tests now cover a cancellation racing after a valid nil-error mutation result, adapter-entry and body-file cancellation with zero HTTP `Do` calls, read cancellation preservation, CLI typed-fault mapping for body-file/adapter-entry scenarios, generic cancellation after a called port remaining unknown, and no cause leakage. The first full focused run after concurrent catalog edits found an unrelated stale `messages list` expected description plus the existing mutation-recovery test's overly broad no-replay assertion; the latter was updated to permit same-command recovery only for the new proven pre-dispatch code. Rerun after search_impl's catalog edits settle.

## Security and public-boundary notes

- Assets and side effects involved: Remote Chatwork mutation and certainty about whether it completed.
- Credentials or confidential data involved: Synthetic PAT only in local server tests; provider body/cause must stay private.
- New dependencies, destinations, files, processes, or generated content: None.
- External schema provenance, publication rights, and drift evidence: No new fixture; synthetic local HTTP bodies only.
- Pagination, timeout, retry, idempotency, and cancellation facts: One attempt; fixed current timeout; no automatic transport retries; post-dispatch generic transport failure is unknown; same-command recovery is allowed only for the infrastructure-minted pre-Do cancellation fault.
- Publication and licensing concerns: None.

## Glossary

- **Dispatch:** The adapter calls its injected HTTP doer for a mutation request. Once that call begins, a generic failure cannot prove the provider received nothing.
- **Authoritative rejection:** A documented provider response whose semantics prove that the requested mutation was rejected.
- **Unknown outcome:** The adapter cannot establish whether the provider applied the mutation; the fault is non-retryable and points to read-only reconciliation.
