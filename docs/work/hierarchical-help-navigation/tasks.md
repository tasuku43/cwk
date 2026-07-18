# Work Tasks: Hierarchical human help navigation

## Understand

- [x] Read governing theses, product, architecture, security, and harness.
- [x] Reproduce root, namespace-selector, namespace-suffix, exact-suffix, and
  agent-root behavior.
- [x] Record verified facts and resolved questions in `context.md`.
- [x] Record the flat-root and rejected-natural-suffix friction as thesis
  evidence.
- [x] Fix the public outcome and non-goals in `goal.md`.

## Decide

- [x] Compare flat, described-namespace, and split human/machine approaches.
- [x] Identify the human text compatibility change and machine compatibility
  boundary.
- [x] Preserve the existing `help` utility role and all task reference flows.
- [x] Confirm no capability-ledger classification changes.
- [x] Confirm no operation effect, target, asset, or trust-boundary changes.
- [x] Confirm authentication, pagination, retry, idempotency, and external
  schema contracts are unaffected.
- [x] Conclude that no ADR is needed because the catalog architecture already
  owns multiple projections.
- [x] Plan thesis/product/architecture/harness propagation instead of a local
  exception.
- [x] Treat the user's explicit implementation request as design approval.

## Implement

- [x] Add failing root, namespace, alias, and agent-compatibility tests.
- [x] Implement catalog-derived hierarchical human help.
- [x] Implement valid trailing namespace/exact help normalization.
- [x] Preserve credential-free help behavior with negative-path coverage.
- [x] Update durable documentation and README.

## Verify

- [x] Focused tests pass. Evidence: `go test ./internal/cli -count=1` succeeds,
  including hierarchy, alias equivalence, input projection, hostile catalog,
  and credential-laziness cases.
- [x] `task check` passes. Evidence: the Go 1.26.5 full gate succeeds with
  hygiene, architecture, contract, unit, security, vulnerability, release,
  public-boundary, and reproducibility checks.
- [x] `task security` passes when required. Evidence: no separate invocation
  was required; the full gate's security guard and vulnerability scan pass.
- [x] `task public:check` passes when required. Evidence: no publication action
  was requested; the full gate's public-boundary guard passes.
- [x] `task release:check` passes when required. Evidence: no release action
  was requested; the full gate's release lint and reproducibility run pass.
- [x] Runtime-only behavior was observed on the required platform. Evidence:
  macOS with Go 1.26.5 produced the hierarchical root, namespace, and exact
  views; unknown selectors retained `unknown_command`; agent JSON remained
  schema v3.
- [x] The relevant agent-readiness scenario met its discovery-round-trip
  budget. Evidence: an unknown human task reaches a leaf through root,
  namespace, and exact help; the machine root retains exact outcomes and its
  existing root-plus-one-scoped-request bound.
- [x] Generated diff and repository status are understood. Evidence: the diff
  is limited to catalog-derived help routing/presentation, contract tests,
  governing documentation, the capability skill, README, and this work packet.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions were promoted out of the work packet.
- [x] Temporary diagnostics and sensitive artifacts were removed.
- [x] Follow-up work is explicit and non-blocking: none is required for this
  closed capability change.
- [x] Commit summary explains the navigation outcome, rationale, and checks.
