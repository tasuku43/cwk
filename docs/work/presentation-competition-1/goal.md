# Work Goal: Select the next agent presentation by evidence

- Status: Complete
- Owner: Project owner
- Target: First post-completion presentation competition
- Related ADRs: None

## Outcome

`cwk` has one reviewed default presentation that preserves every task-relevant
semantic fact and exact canonical reference while removing task-irrelevant
output from the accepted context-capsule baseline. Competition 1 retained all
candidate evidence but selected no eligible challenger. The project owner
therefore chose the simple subtractive task projection as an explicit product
and compatibility decision, using candidate P only as an implementation seed.

The decision, frozen result, and retained evidence are recorded in
[decision.md](decision.md), [evaluation-audit.md](evaluation-audit.md), and
[evidence/manifest.json](evidence/manifest.json). The required repository,
security, and public-boundary gates pass, and the clean experimental worktrees
have been removed without deleting their branches or commits.

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

- [x] Catalog-declared success fields are representable and tested for every
  result variant before compression scores are compared.
- [x] The publishable synthetic corpus covers small and large collections,
  relationships, bounds, explicit zero values, hostile text, and mutation
  outcomes. The frozen key's missing simultaneous To relation is retained and
  audited rather than silently corrected; selected-contract tests cover that
  semantic fact.
- [x] The accepted baseline and materially different challengers consume the
  same typed semantic input in isolated worktrees.
- [x] Static gates verify canonical references, trust framing, bounds,
  uncertainty, output behavior, and deterministic bytes for every renderer;
  active goldens and negative canaries enforce them for the selected default.
- [x] Agent evaluation uses only public `cwk` commands and requires no `jq`,
  `grep`, parser, manual join, source inspection, or undocumented provider
  call. Its oracle and recovery defects are retained as inconclusive evidence.
- [x] The frozen benchmark conclusion is recorded without promoting an
  ineligible challenger, and the separate product compatibility decision is
  explicitly outside the benchmark gate.
- [x] The selected contract, compatibility impact, tests, and migration note
  were reviewed before completion.
- [x] `task check`, `task security`, and `task public:check` pass for the
  integrated implementation and publishable evidence at `b8a522a`.

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
retained, the inconclusive benchmark conclusion and separate owner decision
are recorded, the selected public contract is implemented with compatibility
tests, required gates pass, and temporary credentials, diagnostics, and
candidate worktrees are cleaned up. It does not continue into a competition
rerun or unrelated command, authentication, or API coverage work.
