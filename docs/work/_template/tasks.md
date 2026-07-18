# Work Tasks: Short outcome

Use checkboxes for atomic work and add evidence after completion. This file tracks execution; it does not override the goal, context, plan, or governing invariants.

## Understand

- [ ] Read governing theses, product, architecture, and security sections.
- [ ] Reproduce or observe current behavior.
- [ ] Record verified facts and unknowns in `context.md`.
- [ ] Record repeated decisions, friction, and potential thesis workarounds as evidence.
- [ ] Confirm the public outcome and non-goals in `goal.md`.

## Decide

- [ ] Compare credible approaches and record the selected design.
- [ ] Identify public-contract and compatibility impact.
- [ ] Classify utility/discover/act roles and opaque reference flow.
- [ ] Classify the capability as public, internal, deferred, or excluded.
- [ ] Identify effects, target, assets, and trust-boundary changes.
- [ ] Decide authentication, output completeness, timeout, retry, idempotency, and schema-drift contracts when an external API is involved.
- [ ] Create or update an ADR for a durable trade-off.
- [ ] Revise an incomplete thesis before adding a local code exception, then list propagation work.
- [ ] Obtain required design approval.

## Implement

- [ ] Add failing contract or negative-path tests.
- [ ] Implement domain invariants.
- [ ] Implement application use case and owned ports.
- [ ] Implement bounded infrastructure adapters.
- [ ] Register the command in `cli.Catalog` and update presentation.
- [ ] Update the capability ledger and any publishable schema-fixture manifest.
- [ ] Add producer/consumer graph and exact opaque-ID round-trip tests.
- [ ] Add structured output/error, pagination, cancellation, hostile-output, and zero-downstream-call tests in proportion to risk.
- [ ] Add or update harness enforcement.
- [ ] Update durable documentation.

## Verify

- [ ] Focused tests pass. Evidence:
- [ ] `task check` passes. Evidence:
- [ ] `task security` passes when required. Evidence:
- [ ] `task public:check` passes when required. Evidence:
- [ ] `task release:check` passes when required. Evidence:
- [ ] Runtime-only behavior was observed on the required platform. Evidence:
- [ ] The relevant agent-readiness scenario met its discovery-round-trip budget. Evidence:
- [ ] Generated diff and repository status are understood. Evidence:

## Hand off

- [ ] Acceptance criteria have evidence.
- [ ] Durable decisions were promoted out of the work packet.
- [ ] Temporary diagnostics and sensitive artifacts were removed.
- [ ] Follow-up work is explicit and does not block this goal.
- [ ] Pull request or handoff summary explains outcome, why, checks, and risks.
