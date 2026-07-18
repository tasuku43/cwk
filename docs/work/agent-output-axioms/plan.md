# Plan

## Chosen approach

Freeze representation-independent axioms now. Defer presentation selection until a dedicated work packet defines a typed semantic fixture corpus and evaluation protocol. Implement materially different candidates in isolated worktrees, without allowing one candidate's grammar to alter the shared semantics. Compare them on hard correctness/safety constraints and measured agent task quality, token usage, post-processing steps, and command-discovery cost.

## Candidate competition principles

- Every candidate consumes the same typed semantic input and declares the same coverage and canonical references.
- Every candidate is evaluated with the same agent task prompts, answer key, model/version, invocation budget, fixture bytes, and measurement tooling.
- A candidate that violates semantic, identity, coverage, trust, or output-structure constraints is ineligible regardless of token savings.
- Eligible candidates are compared on a Pareto basis: token reduction cannot purchase lower required understanding quality.
- Candidate-specific code and snapshots remain in their worktrees until review selects a winner or requests another iteration.
- Only the selected design is promoted into the public product/output contract.

## Risks

- Optimizing for one model or fixture may overfit the presentation.
- Token counts can drift when tokenizer or model versions change.
- Qualitative judgments can hide regressions unless evaluation questions have an answer key.
- Parallel worktrees can diverge in semantics and make comparisons invalid.
- A compact candidate may look efficient while increasing command mistakes or hallucinated relationships.

## Verification

- Review governing documents for representation-specific commitments.
- Require future competition work packets to pin inputs, agents, measurements, and acceptance rules before coding.
- Run `task check` and `task public:check` for this axiom pass.
