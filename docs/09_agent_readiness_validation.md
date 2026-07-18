# Agent Readiness Validation

This validation asks whether an agent can translate a user's Chatwork request into an exact `cwk` task, invoke it safely, and understand the task result without guessing or routine external reconstruction. It also defines how presentation candidates are compared before one becomes public.

## Interaction budgets

- Unknown outcome to complete scoped contract: at most two discovery invocations.
- Known command path to complete invocation: one scoped-help invocation.
- Canonical reference reuse: no discovery or transformation invocation between producer and consumer.
- Supported task reconstruction: zero `jq`, `grep`, custom parsers/joins, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Failure recovery: the next corrective command comes from structured metadata.

Direct extraction of a declared canonical reference or fact is allowed. Rebuilding semantics that `cwk` claims to provide is not.

## Presentation-independent semantic fixture

The first Chatwork fixture is synthetic and publishable. Its typed answer key includes:

- room, account, and message canonical identities;
- senders, multiple To recipients, explicit replies, and quotes;
- one resolved relation and one referenced object outside the result bound;
- stable source ordering and duplicate behavior;
- exact retrieval bounds, partiality, missing context, and uncertainty;
- repeated values that may reward compression;
- hostile text resembling provider notation, presentation structure, JSON, agent instructions, controls, bidi/zero-width formats, line separators, delimiters, and pre-existing escapes.

The answer key contains semantics, not an expected rendering. Candidate worktrees may not edit it.

## Agent tasks and exact answers

Using only root/scoped help and one candidate's output, the agent must:

1. choose the exact room-discovery and bounded-message tasks;
2. identify the exact room reference used;
3. identify each requested sender, recipient, reply, and quote fact;
4. distinguish explicit resolved, explicit unresolved, and absent relationships;
5. state the retrieval bound and whether the result represents complete room history;
6. select the canonical reference required by a declared next command;
7. select recovery from typed failure metadata.

Scoring compares answers with the shared key. Presentation-specific explanations are not accepted as substitutes for semantic correctness.

## Candidate eligibility

A candidate is ineligible regardless of token savings when it:

- changes or hides a required semantic answer;
- implies a relationship absent from the typed input;
- loses, transforms, or substitutes a canonical reference;
- hides bounds, partiality, missing context, or uncertainty relevant to the task;
- lets external text inject candidate-authored structure;
- produces nondeterministic bytes for identical typed input;
- requires undocumented parsing or a nonzero external-reconstruction count;
- violates stdout, stderr, exit, failure, completeness, or untrusted-data contracts.

## Parallel-worktree presentation competition

Before implementation, the competition work packet pins:

- fixture corpus and exact semantic answer keys;
- candidate hypotheses and the boundaries they may change;
- target agent/model and tool versions;
- prompts, context supplied, repetitions, temperature/randomness policy, and timeout;
- discovery and tool-invocation budgets;
- tokenizer or authoritative token accounting source;
- correctness scoring, minimum quality floor, and tie/variance handling;
- byte, latency, human-reviewability, and maintenance-cost measurement;
- worktree/commit naming and result storage.

Materially different candidates are implemented in isolated worktrees against the same semantic interface. One candidate's output or helper code must not become another candidate's hidden input. Each report retains raw runs and failures, not just aggregate scores.

## Selection rule

First reject ineligible candidates. Compare the remaining candidates on a Pareto basis across:

- semantic-answer correctness and consistency;
- correct next-command/reference selection;
- input/output and total task tokens;
- extra tool invocations and processing steps;
- serialized bytes and latency;
- human reviewability for safe supervision;
- implementation and maintenance cost.

The minimum understanding-quality floor is set before results are viewed. Lower token use cannot compensate for falling below it. A winner, reviewed combination, or another experiment is an acceptable decision; arbitrary selection is not.

## Recovery probes

Each eligible candidate must preserve the same recovery decisions for:

- no matching room and ambiguous room discovery;
- missing authentication versus insufficient permission;
- provider rate limits and temporary unavailability;
- malformed or oversized notation;
- bounded results with missing referenced context;
- output write failure with no zero-status partial success;
- future mutation rejection and unclassified post-mutation outcomes.

## No-post-processing audit

The transcript fails when a supported task contains an external parser, manual identifier join, raw notation parsing, guessed command/endpoint/cursor, or undeclared provider call. When this happens, decide whether the capability is incomplete, the outcome is too broad, or the presentation candidate failed. Do not patch the agent prompt with the workaround.

## Runnable scaffold probes

Before Chatwork commands and competition fixtures exist, retain:

```sh
go run ./cmd/cwk help --format agent
go run ./cmd/cwk help sample --format agent
go run ./cmd/cwk sample list --format json
go run ./cmd/cwk sample read --id smp_2f4a6c8e0b1d --format json
go run ./cmd/cwk --error-format json sample read --id smp_000000000000
```

These prove bounded discovery, scoped contracts, structured output/error behavior, and exact reference reuse. They do not select or validate a Chatwork presentation.

## Review record

Record the natural-language outcome, discovery/task transcript, external-processing count, provider-call bounds, semantic answers, canonical references, recovery choices, per-run token/byte/latency measurements, agent/model versions, worktree/commit, failures, and variance. Preserve candidate evidence even when it loses so later thesis revisions can distinguish format failure from model or fixture drift.
