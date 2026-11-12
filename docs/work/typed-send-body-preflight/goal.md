# Work Goal: Preflight reply-send body files before provider reads

- Status: Complete, awaiting Terra review
- Retention: temporary
- Retention reason: Review packet for the typed-send body-file preflight P2 follow-up
- Governing contract: `docs/03_security_model.md` bounded filesystem and mutation pre-dispatch invariants
- Review/delete trigger: Delete after Terra review and the parent change completes
- Owner: Codex agent
- Target: Typed message send
- Related ADRs: None

## Outcome

For `messages send --reply-to --body-file`, infrastructure reads and validates the bounded regular file once before the reply GET, keeps the exact bytes in an opaque one-use handle, and later consumes or securely clears that handle without rereading the path. Invalid body input makes zero provider calls.

## Why now

Terra review found that reply lookup GET occurred before infrastructure validated the body file, violating the existing zero-provider-call contract for invalid files.

## Non-goals

- Change the `messages send` public syntax or provider notation contract.
- Give application or domain code filesystem access or body bytes.
- Change one-attempt mutation handling or permit automatic retries.

## Acceptance criteria

- [x] Invalid, missing, unreadable, empty, invalid-UTF-8, and oversized body files produce zero GET and POST calls.
- [x] Successful reply-send reads the file once, uses the same context and binding, preserves exact body suffix bytes, and sends once.
- [x] Body and path are hidden by handle formatting and public faults; handle bytes, source path, binding, and owner are cleared on consume/discard.
- [x] Lost POST response remains non-retryable unknown outcome.
- [x] Focused application, infrastructure, CLI, and domain suites pass; root owns the repository gate.

## Governing documents

- Thesis: `docs/00_theses.md`, controlled effect boundary and mutation outcome
- Product contract: `docs/01_product_contract.md`, message body source and typed send
- Architecture/security: `docs/02_architecture.md`, `docs/03_security_model.md`
- API/harness/readiness: `docs/08_external_api_contracts.md`, `docs/04_harness.md`, `docs/09_agent_readiness_validation.md`

## Completion definition

The focused tests pass, the durable security and API contracts state the preflight order, Terra has reviewed the fix, and this temporary packet is removed with the parent change.
