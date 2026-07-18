# Work Goal: Establish agent-output axioms

> Follow-up decision (2026-07-18): this work deliberately avoided selecting a
> representation as an axiom. The subsequent complete-implementation goal may,
> and does, select candidate C as a replaceable public contract.

- Status: Complete
- Owner: Project owner
- Target: Product-thesis pass before presentation implementation
- Related ADRs: None

## Outcome

Chatwork CLI has representation-independent axioms for agent command discovery and output. The axioms define correctness, sufficient context, boundedness, trust, no mandatory external post-processing, token efficiency, and agent understanding quality without selecting JSON normalization, dictionaries, aliases, indentation, a custom grammar, or another concrete presentation.

## Why now

Early exploration produced several concrete output patterns, including a relationship-oriented compact form. The project owner clarified that these patterns are hypotheses, not governing truths. Presentation candidates will be implemented later in parallel worktrees and compared against the same fixtures and agent tasks before one becomes a public contract.

## Non-goals

- Selecting or implementing a presentation winner.
- Stabilizing an `agent` format, grammar, output mode, alias scheme, or indentation rule.
- Implementing Chatwork API capabilities.
- Choosing authentication storage or numerical optimization thresholds without evidence.

## Acceptance criteria

- [x] Governing theses contain only representation-independent output axioms.
- [x] Concrete presentation mechanisms are explicitly deferred to a controlled competition.
- [x] All candidates share one typed semantic input, fixture corpus, hard correctness constraints, and comparable measurements.
- [x] Token reduction is optimized subject to agent understanding quality and safety, not as a standalone objective.
- [x] Architecture keeps presentation hypotheses out of domain, application, and infrastructure contracts.
- [x] Agent evaluation can compare candidates without external `jq`, custom joins, or source inspection.
- [x] `task check` and `task public:check` pass.

## Governing documents

- Thesis: `docs/00_theses.md`
- Product contract: `docs/01_product_contract.md`
- Architecture and security: `docs/02_architecture.md`, `docs/03_security_model.md`
- Evaluation: `docs/09_agent_readiness_validation.md`

## Completion definition

The work is complete when the abstract axioms and future competition method agree across durable documentation and skills, concrete output choices are clearly uncommitted, and required gates pass.
