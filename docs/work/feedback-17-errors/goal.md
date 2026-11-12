# Work Goal: Make runtime error guidance match outcomes

- Status: In progress
- Retention: temporary
- Retention reason: None
- Governing contract: `docs/00_theses.md` Axiom 7; `docs/01_product_contract.md` failure and mutation confirmation contracts; `docs/03_security_model.md` controlled execution and confirmation; `docs/08_external_api_contracts.md` timeout and fault mapping
- Review/delete trigger: Delete after durable contract updates and focused review are complete
- Owner: Codex agent
- Target: Feedback #17
- Related ADRs: None

## Outcome

Chatwork runtime errors distinguish read deadlines from caller cancellation and ordinary I/O failures, mention room membership only as a possibility for room-target permission denials, and state the exact required confirmation value after a catalog-known policy refusal.

## Why now

Feedback #17 reports that read timeouts currently look like caller cancellation, room 403 errors omit a plausible membership cause, and mutation confirmation failures do not reveal whether `access-change` or `destructive` is required.

## Non-goals

- Change cancellation or unknown-outcome taxonomy for dispatched mutations.
- Expose raw provider text, request causes, or caller-supplied confirmation values.
- Change authentication timing, provider permissions, confirmation policy classes, or retry behavior.

## Acceptance criteria

- [ ] A read deadline gets timeout-specific Japanese message text only when `errors.Is(ctx.Err(), context.DeadlineExceeded)` or `errors.Is(err, context.DeadlineExceeded)`; it retains `operation_canceled` kind/code/exit, caller cancellation keeps its cancellation wording, and ordinary I/O failures remain `chatwork_unavailable`.
- [ ] Mutation timeout/cancellation outcomes keep their existing safe phase-dependent taxonomy.
- [ ] A 403 for a room-target task uses a fixed Japanese phrase that says non-membership is a possibility; non-room targets do not say this, and provider text is never exposed.
- [ ] A `mutation_rejected` produced by a real policy refusal for catalog-known `access-change` or `destructive` confirmation states the exact `--confirm=<value>`; it does not reflect the provided value or private cause, keeps exit/kind/code/retryability/recovery, and makes zero authentication/provider calls.
- [ ] Focused tests cover each branch and hostile values.
- [ ] Required durable contract and agent-readiness wording is updated; this temporary packet is removed when complete.

## Governing documents

- Thesis: `docs/00_theses.md`, Axiom 7
- Product contract: `docs/01_product_contract.md`, failure contract and Mutation confirmation policy
- Architecture/security: `docs/02_architecture.md`, `docs/03_security_model.md`, `docs/08_external_api_contracts.md`
- Readiness: `docs/09_agent_readiness_validation.md`

## Completion definition

Acceptance criteria have evidence, focused tests pass, Terra review can inspect the implementation and behavior, and the temporary packet leaves the final tree.
