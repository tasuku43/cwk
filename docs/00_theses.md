# Project Theses

This document decides ambiguous product and engineering choices for Chatwork CLI (`cwk`). A thesis states a representation-independent product hypothesis, its consequences, and the evidence that can disprove it. Concrete presentation designs remain replaceable until evaluation promotes one into a public contract.

## North Star

**An agent can translate a user's Chatwork request into one exact `cwk` task, invoke it without guessing, and understand a bounded, trustworthy result with no routine external reconstruction. Among outputs that meet the required understanding and safety quality, `cwk` minimizes token cost.**

The primary user is a developer or operator who delegates Chatwork work to a coding agent from a shell or automation environment. Human usability remains important, but command certainty, agent understanding quality, and token efficiency are the first optimization targets.

The product is not measured by Chatwork API endpoint coverage or by the compactness of one syntax. A smaller output is worse when it causes command mistakes, hides missing context, weakens identity, or makes an agent infer relationships.

## Axiom 1: A supported outcome is operationally closed

When `cwk` claims to support a user outcome, it owns the deterministic selection, joining, interpretation, and task-specific transformation needed to use the result.

### Consequences

- Routine success paths do not require `jq`, `grep`, custom joins/parsers, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Direct extraction of a declared field or opaque reference is allowed; reconstructing product semantics is not.
- Repeated external processing is evidence of a missing or overbroad capability, not a workaround to teach every agent.
- Commands express user outcomes rather than provider endpoints.
- A common deterministic workflow belongs in an application use case, not an agent prompt.

### Enforcement

- Catalog entries declare one outcome and complete input/output semantics.
- Agent transcripts count external processing and fail a supported scenario when the count is nonzero.
- Work packets record repeated pipelines as thesis evidence.

## Axiom 2: An agent reaches an executable task without guessing

An agent that knows the user's desired outcome should reach the exact command contract through bounded, machine-readable discovery.

### Consequences

- Root agent help is a compact outcome index.
- An unknown outcome needs at most the root index and one scoped request; a known path needs one scoped request.
- Scoped help declares inputs, effects, authentication, output semantics, completeness, failures, recovery, and reference workflows.
- Commands do not silently search again, choose a display-name match, or rely on hidden defaults.
- Structured recovery names an exact next command rather than prose that the agent must reinterpret.

### Enforcement

- Catalog, routing, and help derive from `cli.Catalog`.
- Root entries retain the 512-byte per-command budget.
- Agent-readiness tests reject command probing, prose scraping, and undeclared follow-up calls.

## Axiom 3: Semantics precede presentation

Chatwork data is converted into a typed, provider-independent task result before presentation. Presentation may reorganize or encode that result, but cannot invent, strengthen, or silently discard semantics required by the task.

### Consequences

- Provider wire JSON is not the public domain model.
- Explicit To, reply, quote, identity, ordering, coverage, and unresolved-reference facts remain distinguishable when relevant to the outcome.
- To does not become a reply; quoted prose, display names, and time proximity do not create relationships.
- Missing or out-of-bound context remains observable rather than being hidden to make an output look complete.
- Canonical opaque references remain available for declared next actions.
- Domain, application, and infrastructure contracts do not depend on a candidate presentation grammar.

### Enforcement

- Shared semantic fixtures have an answer key independent of any renderer.
- Negative tests reject fabricated relationships and silent completeness claims.
- Candidate presentations are evaluated against the same semantic facts and canonical references.

## Axiom 4: Presentation is an empirical, multi-objective decision

No concrete output syntax is a thesis. Presentation is selected by evidence about agent task quality and resource cost.

### Hard constraints

Every eligible presentation must:

- preserve the semantic answer key and exact canonical references;
- expose task-relevant bounds, missing context, and uncertainty;
- keep external text structurally separate and marked as untrusted data;
- be deterministic for the same typed input;
- require no undocumented parsing convention or external post-processing for the evaluated outcome;
- preserve stdout, stderr, exit, failure, and completeness contracts.

### Optimization objectives

Among eligible candidates, prefer the Pareto frontier across:

- agent answer correctness and relationship understanding;
- correct next-command and reference selection;
- input/output token use;
- additional tool invocations and processing steps;
- serialized bytes, latency, and implementation/maintenance cost;
- human reviewability when it affects safe supervision.

Token count is not optimized below the required understanding-quality floor. Numerical thresholds, tokenizer/model versions, fixtures, and repetitions are chosen in the presentation competition work packet, not invented in this thesis.

## Axiom 5: Presentation candidates compete behind one semantic boundary

Materially different presentation hypotheses are implemented in isolated worktrees and evaluated under comparable conditions before one is stabilized.

### Consequences

- Each candidate consumes the same typed semantic input and provider-independent fixtures.
- Candidate worktrees cannot change semantics, coverage, or answer keys to improve their score.
- The evaluation pins task prompts, model/agent versions, invocation budgets, token accounting, repetitions, and scoring.
- Candidate-specific grammars, dictionaries, aliases, indentation, schemas, and output modes remain hypotheses in their worktrees.
- Only a reviewed winner, or an explicitly selected combination, is promoted into the public output contract and main implementation.
- Inconclusive evidence results in another experiment, not an arbitrary format commitment.

### Enforcement

- A presentation-competition work packet defines candidates and measurement before implementation begins.
- Comparison reports identify each worktree/commit and record raw results, not only a winner summary.
- The selected design receives golden/contract tests only after the decision is accepted.

## Axiom 6: Discovery owns ambiguity; actions use exact references

Room, account, and message discovery may return candidates. A read or mutation acts on declared opaque references passed unchanged from an invocable producer.

### Consequences

- Every public command is `utility`, `discover`, or `act`.
- Display names, positions, browser URLs, and presentation-derived shorthand are not authorization identities.
- An action never case-folds, decodes, trims, reconstructs, or substitutes a Chatwork identifier.
- Required-reference chains lead back to an invocable producer.

### Enforcement

- Reference kinds live on structured catalog inputs and outputs.
- Whole-catalog graph tests prove producer/consumer reachability.
- Round-trip and negative tests preserve exact identity and reject alternate forms before adapter access.

## Axiom 7: Effects and uncertain outcomes are visible before repetition

An agent must know what a command can affect and whether a previous mutation may have happened before repeating or recovering.

### Consequences

- Every public operation declares `read`, `create`, or `write`; unknown effects fail closed.
- Message creation binds its room parent and declares notification impact.
- Message update/deletion binds the existing message and declares destructive/notification impact.
- Authentication, permission, validation, and policy rejection cause zero downstream calls.
- An unclassified post-mutation outcome is non-retryable and points to read-only reconciliation.
- Credentials and unsafe provider causes remain inside infrastructure.

### Enforcement

- Domain intent and catalog mutation validation reject incomplete targets or impact.
- Fake-adapter tests prove pre-I/O rejection and safe post-I/O classification.
- Secret-canary and public-boundary checks cover output, errors, logs, and fixtures.

## Axiom 8: Claims remain executable

Command certainty, operational closure, semantic fidelity, understanding quality, token efficiency, output safety, and reference flow require repeatable evidence.

### Consequences and enforcement

- `scripts/check.sh` remains the canonical gate and `cli.Catalog` the public-command source of truth.
- Agent evaluations record discovery calls, external-processing steps, canonical references, semantic answers, recovery, tokens, and latency.
- Public examples use synthetic Chatwork-like data.
- `task check` decides implementation completion; higher-risk changes add the named security/public/release profiles.

## First testable slice

The first slice should test the axioms rather than prematurely stabilize presentation:

1. define a typed semantic fixture for room discovery and a bounded recent-message result;
2. include explicit and missing relationships, repeated values, canonical references, hostile text, and partial coverage;
3. define agent questions and exact answers for command selection, relationship understanding, completeness, and next-reference use;
4. implement materially different presentation candidates in isolated worktrees against the same input;
5. compare eligibility, understanding quality, token use, tool steps, and maintenance cost;
6. promote a presentation contract only after reviewing the evidence.

Authentication storage, exact public command names, provider call budgets, presentation candidates, and numerical acceptance thresholds remain decisions for subsequent work packets.

## Explicit non-goals

- Mirroring every Chatwork API endpoint.
- Raw routes, arbitrary headers/bodies, or transport passthrough.
- Selecting JSON, a custom compact grammar, dictionaries, aliases, indentation, or another concrete representation as an axiom.
- Hidden fuzzy target selection.
- Silent truncation or fabricated relationships.
- Default lossy or model-generated summaries.
- Claiming structural escaping prevents semantic prompt injection.
- Enabling writes before authentication, impact, retry, and reconciliation policies are concrete.

## Thesis lifecycle

Record agent confusion, repeated pipelines, extra discovery calls, answer errors, token regressions, unsafe identifier conversions, and comparison bias in the active work packet. Revise an axiom before normalizing a workaround. Presentation evidence updates the selected contract; it does not retroactively turn one implementation pattern into an axiom.
