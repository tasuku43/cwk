# Product Contract

This document defines which user outcomes belong in Chatwork CLI (`cwk`) and the representation-independent behavior they must preserve. Concrete presentation becomes public only after evidence-based selection.

## Product statement

`cwk` is a task-oriented Chatwork CLI for developers, operators, and coding agents. It maps user outcomes to exact bounded commands and returns sufficient, trustworthy task context with low agent cognitive and token cost.

It is not an API explorer or a one-to-one endpoint wrapper. It does not make every consumer rebuild routine Chatwork semantics with external tools.

## Primary users

- A developer or operator delegating Chatwork work to a coding agent.
- An automation author relying on stable command, reference, failure, and semantic-output contracts.
- A human supervising what an agent will read or change.
- A contributor testing presentation ideas without changing product semantics.

## Supported-outcome promise

For every supported outcome, an agent can:

1. select the exact task from the root outcome index and at most one scoped contract;
2. supply declared inputs without guessing endpoints, name matches, URLs, or hidden defaults;
3. identify task-relevant facts, bounds, missing context, uncertainty, and canonical references;
4. complete the outcome without routine `jq`, `grep`, custom joins/parsers, raw Chatwork-notation interpretation, source inspection, or exploratory API calls;
5. distinguish success, declared bounded/partial context, and failure;
6. recover through structured executable next actions.

Direct extraction of a declared field or canonical reference is allowed. Reconstructing product semantics from provider fields or multiple undocumented calls is not.

## Current runnable surface

Until the first Chatwork slice replaces scaffold examples, the repository exposes `help`, `doctor`, `sample list`, `sample read`, and `version`. These commands prove catalog, opaque-reference, output, error, and harness behavior. They do not prove a Chatwork outcome or settle its presentation.

## Planned first semantic outcome

The first read-only outcome is room discovery followed by one bounded recent-message result for an exact room reference. Before presentation selection, its provider-independent semantic model must make task-relevant instances of these facts available:

- room, account, and message identity;
- sender and explicit To/reply/quote relationships;
- stable ordering;
- canonical references for declared next actions;
- retrieval bounds, partiality, missing relationships, and uncertainty;
- external text as untrusted data.

Whether and how those facts appear in JSON, a compact text form, a structured transcript, dictionaries, indentation, or another representation is deliberately undecided.

## Public vocabulary

Public command names describe user outcomes. Provider endpoint names and raw notation tags remain infrastructure vocabulary. The semantic vocabulary includes room, participant, message, recipient, reply, quote, context bound, missing reference, and canonical action reference. Presentation candidates may not redefine these meanings.

## Agent-output axioms

Every eligible presentation must:

- derive from the same typed task result rather than reparsing another renderer;
- preserve the outcome's semantic answer key and canonical references;
- expose applicable bounds, partiality, missing context, and uncertainty;
- deterministically separate CLI-authored structure from untrusted provider text;
- preserve success/failure stream, status, completeness, and recovery contracts;
- support the evaluated outcome without undocumented external processing.

No concrete field layout, grammar, aliasing strategy, indentation scheme, or output mode is promised yet.

## Presentation selection lifecycle

Presentation becomes a public contract through a dedicated competition:

1. define one typed semantic fixture corpus and exact answer key;
2. define agent tasks, model/agent versions, prompts, repetitions, invocation budgets, token accounting, and scoring before implementation;
3. implement materially different candidates in isolated worktrees;
4. reject candidates that fail semantic, identity, coverage, trust, determinism, or output-boundary requirements;
5. compare eligible candidates on understanding quality, correct next action/reference, tokens, tool steps, bytes, latency, reviewability, and maintenance cost;
6. select a winner, combination, or another iteration through reviewed evidence;
7. only then stabilize schema/grammar versions, defaults, compatibility promises, and golden tests.

Candidate worktrees are experimental. Their output is not public merely because it runs.

## Filtering and task composition

The product owns deterministic filtering and joining needed by a supported outcome. Whether this is exposed through dedicated commands, finite typed filters, or another interface remains a later command-design decision.

The governing rule is not “never use a query language”; it is “do not shift a recurring supported task back to the agent.” A generic expression facility must earn its place through the same outcome, discovery, safety, and evaluation evidence as any other public capability.

## Discovery, action, and references

Every public command declares `utility`, `discover`, or `act`. Discovery owns ambiguity and returns canonical opaque references. Actions require exact declared references and never perform hidden display-name searches. Presentation-derived shorthand is not command identity unless a future separately typed contract defines its scope and resolution.

## Side effects

Every command declares `read`, `create`, or `write`.

- Reading messages is bounded and non-mutating.
- Sending a message is `create`, binds one room parent, and declares notification impact.
- Editing or deleting is `write`, binds the existing message, and declares notification/destructive impact.
- Unknown or unauthorized mutation intent fails before Chatwork I/O.
- Uncertain post-mutation outcomes are not automatically retried and use read-only reconciliation.

The first Chatwork slice remains read-only.

## Authentication and external-call decisions

The project will initially evaluate a single Chatwork account and API token as the smallest authentication scope. Token acquisition/storage is undecided; normal argv and plaintext project configuration are excluded.

Before live I/O, a capability work packet must decide credential lifecycle, allowed destinations, timeouts, attempts, rate limits, response bounds, cancellation, message-window semantics, and publishable schema fixtures. OAuth and multiple accounts remain deferred until justified by an outcome.

## Compatibility boundary

Before `1.0.0`, contracts may evolve intentionally with tests and migration notes. Once stabilized, compatibility includes command paths, typed inputs, roles, effects, reference kinds, semantic field meanings, bounds/completeness, failures, authentication configuration, and release artifacts.

Presentation grammar, schemas, defaults, and ordering become compatibility promises only after the presentation competition selects and accepts them. Experimental worktree output carries no compatibility promise.

## Explicit non-goals

- Complete Chatwork API coverage or raw transport passthrough.
- Choosing a concrete output syntax as a thesis.
- Silent fuzzy matching, truncation, or relationship inference.
- Default lossy/model-generated summaries.
- Claiming structural escaping makes external text semantically trustworthy.
- OAuth, multiple accounts, administration, deletion, upload, and other mutations in the first slice.

## Completion evidence for a Chatwork capability

A capability is complete only when its outcome, non-goals, command discovery, semantic model, exact references, bounds, failure behavior, authentication, external-call policy, hostile-data tests, and agent transcript are reviewed. Presentation-dependent completion additionally requires the accepted competition evidence and selected format contract. Required repository gates must pass.
