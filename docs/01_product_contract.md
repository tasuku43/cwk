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

## Public runnable surface

The public catalog exposes local `help`, `doctor`, and `version` utilities plus the task-oriented Chatwork workflows defined below. The former `sample list` and `sample read` scaffold is retained only as an offline test fixture; it is absent from root help and cannot be invoked through the default catalog. Public opaque-reference and output contracts are proven by the Chatwork workflows themselves.

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

Candidate C (`cwk-context-capsule/1`) is the first stable data-presentation baseline: a versioned context capsule with deterministic headers, a compact local reference dictionary, typed task facts, explicit relationships/bounds, and visibly framed external text. Its local aliases were never command identity.

The current default is the headerless task projection, a further reviewed subtraction of the P-derived `cwk-task-projection/1`. It starts directly with the task result and emits only:

- catalog-declared fields required by the task result;
- exact canonical references; `messages list` additionally uses a deterministic
  document-local actor dictionary to factor repeated sender identity and name,
  without replacing any canonical action reference;
- task-relevant bounds, completeness, and uncertainty;
- structural trust framing for every external-text field.

It does not publish a global version/task preamble, a standalone provider-oriented coverage record, raw Chatwork notation as semantic structure, undeclared provider/wire fields, duplicated coverage prose, empty optional shells, or helpful non-contract defaults. Collection bounds and completeness sit on the collection record; a message window uses the task vocabulary `recent` or `changes`. `messages list` emits its room, trust classification, and the fixed schema `#sequence message-ref actor sent [reply] [to] [quote] "body"` once, then an actor dictionary and one physical record per selected typed message in original provider order. Without a sender filter, every provider-returned message is selected. A filtered result additionally emits one selection record before the trust declaration; it preserves the source-window count, exact sender filter, context policy, and sender-match anchors without adding state to every message. The sequence, canonical message reference, actor, send time, and quoted body are positional; optional typed edges remain labeled. `#N` is the one-based original provider sequence and may contain gaps after selection; `reply=#N` is a local edge, not command identity. To and reply remain separate, unresolved targets retain an available canonical reference, and depth/thread/root/children/resolved-default records are absent. A declared raw message body remains visible as untrusted external text; presentation does not reinterpret it as a reply, recipient, quote, instruction, or other semantic fact.

Seven homogeneous read collections also declare `external-text=untrusted
escaped` and one fixed schema before their provider-order records:

```text
contacts: account-ref room-ref "name" [organization]
rooms: room-ref "name" type role unread mentions tasks
members: account-ref "name" role
personal-tasks: task-ref room-ref assigned-by-ref message-ref "body" status
room-tasks: task-ref room-ref account-ref message-ref "body" status limit-time
files: file-ref room-ref account-ref message-ref "name" size
contact-requests: request-ref account-ref "name" ["message"]
```

The schema line is presentation metadata, not a semantic record. Required
positions never shift. Contact organization is an optional final labeled
suffix, and a contact-request message is an optional final quoted position. A
file's fourth position is its canonical message reference when present or the
literal `absent`; only a canonical value may be reused as command input.

## Future presentation-selection lifecycle

Candidate C was selected for the first complete implementation by explicit product decision. Competition 1 later compared C with alternative projections but was inconclusive because benchmark/oracle and recovery-prompt defects made its promotion result non-authoritative. The owner separately chose a P-derived projection as the new default and explicitly accepted a breaking text-schema migration; the integrated projection adds hardening and subtraction beyond frozen candidate P. The current default is therefore an owner compatibility decision after the competition, not the benchmark winner.

A future replacement becomes a public contract through a dedicated competition and compatibility decision:

1. define one typed semantic fixture corpus and exact answer key;
2. define agent tasks, model/agent versions, prompts, repetitions, invocation budgets, token accounting, and scoring before implementation;
3. implement materially different candidates in isolated worktrees;
4. reject candidates that fail semantic, identity, coverage, trust, determinism, or output-boundary requirements;
5. compare eligible candidates on understanding quality, correct next action/reference, tokens, tool steps, bytes, latency, reviewability, and maintenance cost;
6. select a winner, combination, or another iteration through reviewed evidence;
7. only then make and record the compatibility decision that changes the current schema/grammar version, default, compatibility promises, and golden tests.

Candidate worktrees are experimental. Their output is not public merely because it runs. Raw runs, score summaries, audit findings, and known benchmark defects remain evidence even when the experiment is inconclusive; they must not be rewritten to imply that the subsequently selected format won. The flat `messages list` adjacency refinement is a separate explicit owner compatibility decision, not a retroactive Competition 1 result.

## Filtering and task composition

The product owns deterministic filtering and joining needed by a supported
outcome. `messages list` establishes the first finite typed filter contract:
up to 100 repeatable `--sender <account-ref>` inputs match any listed exact
sender, and
`--context none|replies` defaults to `none`. `replies` adds only direct parents
and children connected to sender matches by typed reply edges inside the one
provider-returned window. It does not traverse transitively, expand To or quote
relations, parse raw body notation, fetch missing parents, or perform another
provider call.

Filtering retains the provider window's original one-based sequence, so
displayed `#N` values may have gaps. A single selection record declares the
source-window count, exact sender set, context mode, and sender-match sequence
anchors; records not listed as anchors are included reply context. Repeating two
senders is a truthful two-sender-focused slice, not a claim that every displayed
message is authored by or directed exclusively between them. Exact canonical
account and message references remain the only command identities.

The governing rule is not “never use a query language”; it is “do not shift a recurring supported task back to the agent.” A generic expression facility must earn its place through the same outcome, discovery, safety, and evaluation evidence as any other public capability.

## Discovery, action, and references

Every public command declares `utility`, `discover`, or `act`. Discovery owns
ambiguity and returns canonical opaque references. Actions require exact
declared references and never perform hidden display-name searches. The sole
reference-free action form binds one catalog-declared fixed `tool_local` target
when the product owns exactly one instance and offers no selection. It cannot
identify a remote or potentially multiple object. Presentation-derived
shorthand is not command identity unless a future separately typed contract
defines its scope and resolution.

## Side effects

Every command declares `read`, `create`, or `write`.

- Reading messages is bounded and non-mutating.
- Sending a message is `create`, binds one room parent, and declares notification impact.
- Editing or deleting is `write`, binds the existing message, and declares notification/destructive impact.
- Unknown or unauthorized mutation intent fails before Chatwork I/O.
- Uncertain post-mutation outcomes are not automatically retried and use read-only reconciliation.

The room-discovery/message anchor lands as a read slice, but the first complete implementation also includes the fixed mutation surface under the confirmation policy below.

## Authentication and external-call decisions

The first implementation supports one Chatwork account per command process
through the API token in `CWK_API_TOKEN`. That environment value is the sole
credential input and PAT is therefore the only method admitted by every
Chatwork task. There is no method selector, credential probing, login/status/
logout command, stored account choice, or fallback path.

Infrastructure reads the token once from the command environment, validates
its bounded shape, keeps it behind a secret-free ephemeral binding, and sends
it only as `x-chatworktoken` to `https://api.chatwork.com/v2`. `cwk` never
accepts the token in argv, writes it to project or user configuration, stores
it in an operating-system credential service, or renders it. Missing or
invalid token input fails before a provider task request; public destination
overrides remain forbidden.

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

The provider documents a 100-item maximum for `GET /my/tasks`, room message, room task, room file, and incoming-request lists. Those results preserve that 100-item provider bound rather than claiming the 10,000-item aggregate ceiling. The active contracts fix cancellation, message-window semantics, provider rate-limit behavior, mutation policy, PAT failure-before-I/O behavior, and publishable schema fixtures before live I/O is enabled. Multiple accounts remain deferred.

## Mutation confirmation policy

An exact invocation with validated canonical references and complete typed intent is sufficient for ordinary creates and updates, including message/task creation, room metadata changes, read-state changes, message edits, and task status changes. This is explicit command intent, not a general authorization grant.

Mutations that change membership, contact access, or link exposure additionally require the exact `--confirm=access-change` value. In the fixed operation snapshot these are room creation, room-member replacement, invite-link creation/update, and incoming-contact-request acceptance. Destructive operations additionally require the exact `--confirm=destructive` value: room leave/delete, message deletion, invite-link deletion, and incoming-contact-request rejection. Confirmation is invocation-local, is never inferred from a TTY or agent identity, and is not reused.

Every provider operation has one transport attempt. An uncertain mutation result is non-retryable and names an exact read-only catalog task for reconciliation before another mutation; it never recommends repeating the write.

## Compatibility boundary

Before `1.0.0`, contracts may evolve intentionally with tests and migration notes. Once stabilized, compatibility includes command paths, typed inputs, roles, effects, reference kinds, semantic field meanings, bounds/completeness, failures, authentication configuration, and release artifacts.

Candidate C's versioned grammar, schemas, defaults, and ordering were the compatibility promises of the first complete implementation. The P-derived `cwk-task-projection/1` deliberately broke that contract; the headerless projection made a second pre-1.0 break by removing its repeated schema/task preamble and standalone coverage record. The flat chronological `messages list` adjacency contract is a third explicit pre-1.0 refinement and superseded an unimplemented indented-tree proposal. Applying fixed positional records to the seven reviewed homogeneous collections is a fourth explicit refinement. Clients must not expect historical headers, reference dictionaries, aliases, field ordering, or grammar. Current compatibility is identified out of band by the release and enforced by catalog fields, documentation, all-route tests, and goldens. Semantic field meanings, exact canonical references, bounds/completeness, failures, and trust classifications remain governed independently of a text migration. A future replacement changes the current promises only through reviewed evidence and an explicit compatibility decision. Experimental worktree output carries no compatibility promise.

## Explicit non-goals

- Tracking future Chatwork additions automatically, mechanically mirroring endpoints, or exposing raw transport passthrough.
- Treating candidate C or the current task projection as a thesis, or allowing presentation shorthand to become action identity.
- Silent fuzzy matching, truncation, or relationship inference.
- Default lossy/model-generated summaries.
- Claiming structural escaping makes external text semantically trustworthy.
- OAuth grants and lifecycle commands; token persistence; multiple accounts or credential profiles; administration/private APIs; webhooks; GUI work; and release publication in the first complete implementation. Presentation experiments and token optimization remain separate from API-capability completeness.

## Completion evidence for a Chatwork capability

A capability is complete only when its outcome, non-goals, command discovery, semantic model, exact references, bounds, failure behavior, authentication, external-call policy, hostile-data tests, and agent transcript are reviewed. Candidate C remains the first-stable baseline evidence. Presentation-dependent completion under the current default requires the headerless task-projection contract, its subtractive-field and golden evidence, and the recorded breaking compatibility decisions; no benchmark-win claim is required or permitted for the inconclusive Competition 1. Required repository gates must pass.
