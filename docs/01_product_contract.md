# Product Contract

This document defines which user outcomes belong in Chatwork CLI (`cwk`) and the representation-independent behavior they must preserve. A concrete presentation is an explicit, replaceable product contract; future replacements require comparative evidence.

## Product statement

`cwk` is a task-oriented Chatwork CLI for developers, operators, and coding agents. It maps user outcomes to exact bounded commands and returns sufficient, trustworthy task context with low agent cognitive and token cost.

It is not an API explorer or a one-to-one endpoint wrapper. Its first complete implementation covers the fixed 2026-07-18 public REST-operation snapshot through reviewed user-task workflows, without making every consumer rebuild routine Chatwork semantics with external tools.

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

## Required first complete surface

Room discovery followed by one bounded recent-message result for an exact room reference is the anchor outcome. Its provider-independent semantic model makes task-relevant instances of these facts available:

- room, account, and message identity;
- sender and explicit To/reply/quote relationships;
- stable ordering;
- canonical references for declared next actions;
- retrieval bounds, partiality, missing relationships, and uncertainty;
- external text as untrusted data.

The same complete implementation covers the 32 REST operations in the official 2026-07-18 documentation snapshot across account/status, contacts, rooms, members, messages, tasks, files, invite links, and incoming contact requests. A checked operation-to-capability mapping proves coverage while the catalog remains the only public-command registry.

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

The first stable data presentation is candidate C: a versioned context capsule with deterministic headers, a compact local reference dictionary, typed task facts, explicit relationships/bounds, and visibly framed external text. Local aliases reduce repetition but are never accepted as command identity; the same capsule exposes canonical references for exact reuse. The grammar is a compatibility contract, not a product axiom.

## Future presentation-selection lifecycle

Candidate C is selected for the first complete implementation by explicit product decision. A future replacement becomes a public contract through a dedicated competition:

1. define one typed semantic fixture corpus and exact answer key;
2. define agent tasks, model/agent versions, prompts, repetitions, invocation budgets, token accounting, and scoring before implementation;
3. implement materially different candidates in isolated worktrees;
4. reject candidates that fail semantic, identity, coverage, trust, determinism, or output-boundary requirements;
5. compare eligible candidates on understanding quality, correct next action/reference, tokens, tool steps, bytes, latency, reviewability, and maintenance cost;
6. select a winner, combination, or another iteration through reviewed evidence;
7. only then change candidate C's schema/grammar versions, defaults, compatibility promises, and golden tests.

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

The room-discovery/message anchor lands as a read slice, but the first complete implementation also includes the fixed mutation surface under the confirmation policy below.

## Authentication and external-call decisions

The first implementation supports one Chatwork account through either PAT or
OAuth 2.0. Every API task requires the exact secret-free selector
`CWK_AUTH_METHOD=pat|oauth2`; there is no implicit preference or fallback when
both credentials exist.

PAT reads `CWK_API_TOKEN` only from the command-process environment and never
persists it. OAuth uses the provider's Authorization Code Grant for one public
client with state and PKCE S256. `CWK_OAUTH_CLIENT_ID` and
`CWK_OAUTH_REDIRECT_URI` are non-secret registration configuration; the
redirect must be an exact registered non-HTTP custom URI. Login shows one
transient authorization URL on stderr, reads the complete callback URL from
stdin, validates the exact redirect/state before exchange, and stores access
and refresh material only in the operating-system credential store. Codes,
verifiers, tokens, credential-store keys, and credential-bearing errors never
enter argv or stdout.

`auth profiles` discovers the one opaque public-client profile reference.
`auth login`, `auth status`, and `auth logout` consume that reference unchanged.
Login refuses to replace an existing credential; status exposes only method,
state, and provider-advertised expiry; logout removes the local credential and
does not claim provider-side revocation. API tasks using either method converge
on the same secret-free session/binding contract before infrastructure sends a
request to `https://api.chatwork.com/v2`; public destination overrides remain
forbidden.

The first implementation fixes these ceilings; command-line or environment overrides cannot raise them:

| Boundary | Ceiling |
|---|---:|
| Metadata and ordinary read/write request timeout | 20 seconds |
| File-upload request timeout | 60 seconds |
| Transport attempts per logical operation | 1 |
| Successful provider response body | 8 MiB |
| Provider error body read for classification | 64 KiB |
| Complete stdout result | 16 MiB |
| Aggregated list result | 10,000 items |
| File upload | 5 MiB |

The provider documents a 100-item maximum for `GET /my/tasks`, room message, room task, room file, and incoming-request lists. Those results preserve that 100-item provider bound rather than claiming the 10,000-item aggregate ceiling. The active capability work packet fixes cancellation, message-window semantics, provider rate-limit behavior, mutation policy, OAuth lifecycle, and publishable schema fixtures before live I/O is enabled. Multiple accounts remain deferred.

## Mutation confirmation policy

An exact invocation with validated canonical references and complete typed intent is sufficient for ordinary creates and updates, including message/task creation, room metadata changes, read-state changes, message edits, and task status changes. This is explicit command intent, not a general authorization grant.

Mutations that change membership, contact access, or link exposure additionally require the exact `--confirm=access-change` value. In the fixed operation snapshot these are room creation, room-member replacement, invite-link creation/update, and incoming-contact-request acceptance. Destructive operations additionally require the exact `--confirm=destructive` value: room leave/delete, message deletion, invite-link deletion, and incoming-contact-request rejection. Confirmation is invocation-local, is never inferred from a TTY or agent identity, and is not reused.

Every provider operation has one transport attempt. An uncertain mutation result is non-retryable and names an exact read-only catalog task for reconciliation before another mutation; it never recommends repeating the write.

## Compatibility boundary

Before `1.0.0`, contracts may evolve intentionally with tests and migration notes. Once stabilized, compatibility includes command paths, typed inputs, roles, effects, reference kinds, semantic field meanings, bounds/completeness, failures, authentication configuration, and release artifacts.

Candidate C's versioned grammar, schemas, defaults, and ordering are compatibility promises for the first complete implementation because the product has selected it explicitly. A future replacement changes those promises only after the presentation competition accepts it. Experimental worktree output carries no compatibility promise.

## Explicit non-goals

- Tracking future Chatwork additions automatically, mechanically mirroring endpoints, or exposing raw transport passthrough.
- Treating candidate C as a thesis or allowing its aliases to become action identity.
- Silent fuzzy matching, truncation, or relationship inference.
- Default lossy/model-generated summaries.
- Claiming structural escaping makes external text semantically trustworthy.
- Confidential-client, loopback-HTTP, and device OAuth grants; multiple accounts or OAuth profiles; administration/private APIs; webhooks; GUI work; alternative presentations; token optimization; and release publication in the first complete implementation.

## Completion evidence for a Chatwork capability

A capability is complete only when its outcome, non-goals, command discovery, semantic model, exact references, bounds, failure behavior, authentication, external-call policy, hostile-data tests, and agent transcript are reviewed. Presentation-dependent completion requires the accepted candidate-C format contract and its baseline evidence; a future replacement additionally requires accepted competition evidence. Required repository gates must pass.
