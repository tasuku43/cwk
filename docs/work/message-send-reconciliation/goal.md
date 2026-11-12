# Work Goal: Reconcile an uncertain message send safely

> Historical design: the former all-author candidate scope was superseded by `docs/work/reconcile-own-messages/` and the durable current-account contract.

- Status: In progress (Terra review pending)
- Retention: temporary
- Retention reason: None
- Governing contract: `docs/00_theses.md` Axiom 1; `docs/01_product_contract.md` mutation unknown-outcome contract; `docs/03_security_model.md` body-file boundary; `docs/08_external_api_contracts.md` single-attempt mutation and read-only reconciliation
- Review/delete trigger: Terra verifies the boundary and focused tests; remove after that review and before the final tree
- Owner: Codex agent
- Target: Feedback #4
- Related ADRs: None

## Outcome

When a message send has an unknown provider outcome, the agent can invoke one declared read-only task with the same room, body source, recipients, and reply target to inspect every exact expanded-body match in the latest bounded window without deciding that the original send succeeded or failed.

## Why now

Feedback #4 reports repeated external grep pipelines after `chatwork_mutation_outcome_unknown`. Chatwork may have applied a write whose response was lost, so retrying can create a duplicate.

## Non-goals

- Retry or change any mutation.
- Prove which invocation, person, or account created a matching message.
- Search older history, infer a send from an absent match, or expose the submitted body.
- Add a caller-side notation parser or a second transport path.

## Acceptance criteria

- [x] `messages send` unknown outcomes recover through exact catalog path `messages reconcile-send`.
- [x] Reconciliation validates `--body`/`--body-file`, ordered `--to`, and optional `--reply-to` with the existing send contracts, then makes at most one exact reply-target GET and exactly one latest message-list GET, using the same binding and zero POSTs.
- [x] The exact send expanded-body builder produces one opaque, request- and binding-bound matcher; body-file content is read by the bounded infrastructure reader and the matcher is discarded and zeroed on all paths. Existing typed request/list-result bodies still cross the application task boundary, while the reconciliation-specific file reader returns no content and the matcher returns only booleans.
- [x] The result returns every canonical exact match with author and time, count, source limit, completeness, access limitation, and explicit unknown outcome. It never returns body or file path and never calls no-match proof of non-send.
- [x] Tests cover zero/one/many matches, Unicode and line-ending bytes, hostile output, restricted/out-of-window evidence, invalid-input zero reads, no POST, binding, one-use and cleanup, canonical reference reuse, and no external post-processing.
- [x] Durable product, architecture, security, authentication, external API, harness, skill, and readiness contracts agree; focused tests pass.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1 and 7
- Product contract: `docs/01_product_contract.md`, mutation outcome recovery and message send
- Architecture: `docs/02_architecture.md`, app-owned task orchestration and infrastructure boundary
- Security: `docs/03_security_model.md`, send-body source and mutation outcome
- Authentication/API: `docs/07_authentication.md`, `docs/08_external_api_contracts.md`
- Harness/readiness: `docs/04_harness.md`, `docs/09_agent_readiness_validation.md`

## Completion definition

All acceptance criteria have concrete test or document evidence, repository changes preserve one-attempt writes and exact-reference contracts, the required checks selected by the parent pass, and this temporary packet is removed.
