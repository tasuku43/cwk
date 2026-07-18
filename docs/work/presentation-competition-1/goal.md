# Work Goal: Select the next agent presentation by evidence

- Status: Proposed
- Owner: Project owner
- Target: First post-completion presentation competition
- Related ADRs: None

## Outcome

`cwk` has one reviewed default presentation that preserves every task-relevant
semantic fact and exact canonical reference while reducing agent task cost
relative to the accepted context-capsule baseline. The decision is supported
by reproducible, per-run evidence from isolated candidate worktrees rather
than by visual preference.

## Why now

The complete Chatwork API surface now runs against a real test account. Live
use shows that relationship parsing and bounds are useful, but the generic
renderer repeats scope, identity, empty states, and provider notation and also
omits several catalog-declared mutation-result fields. Optimizing bytes before
repairing those meaning gaps would reward an incomplete contract.

## Non-goals

- Changing Chatwork task semantics, command paths, effects, authentication,
  provider-call policy, or opaque-reference identity.
- Adding OAuth, token persistence, profiles, or multiple accounts.
- Using lossy summaries, inferred relationships, or model-generated output.
- Selecting a candidate from screenshots, intuition, or token count alone.
- Shipping every experimental renderer or adding a public format selector.
- Retaining live credentials, personal data, live IDs, or private Chatwork
  history as evaluation evidence.

## Acceptance criteria

- [ ] Catalog-declared success fields are representable and tested for every
  result variant before compression scores are compared.
- [ ] One publishable synthetic fixture corpus and semantic answer key cover
  small and large collections, relationships, bounds, explicit zero values,
  hostile text, and mutation outcomes.
- [ ] The accepted baseline and materially different challengers consume the
  same typed semantic input in isolated worktrees.
- [ ] Every candidate preserves exact canonical references, trust framing,
  bounds, uncertainty, stdout/stderr/exit behavior, and deterministic bytes.
- [ ] Agent evaluation requires no `jq`, `grep`, parser, manual join, source
  inspection, or undocumented provider call.
- [ ] A candidate is promoted only by the predeclared quality and resource
  gates in `protocol.md`; inconclusive evidence leaves the baseline unchanged.
- [ ] The selected contract, compatibility impact, tests, and migration note
  are reviewed before its implementation is merged.
- [ ] `task check` passes; `task public:check` also passes for publishable
  fixture or public-document changes.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1, 3, 4, 5, and 8
- Product contract: `docs/01_product_contract.md`, Agent-output axioms and
  Future presentation-selection lifecycle
- Architecture: `docs/02_architecture.md`, Semantic and presentation boundary
- Security: `docs/03_security_model.md`, Output and terminal safety
- Validation: `docs/09_agent_readiness_validation.md`

## Completion definition

This work is complete only when the shared contract gaps are repaired, the
protocol and fixtures are frozen, all candidate runs and raw measurements are
retained, one result is explicitly accepted or the baseline is explicitly
retained, the accepted public contract is implemented with compatibility
tests, required gates pass, and temporary credentials and diagnostics are
removed. It does not continue into unrelated command, authentication, or API
coverage work.
