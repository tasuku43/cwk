# Agent Readiness Validation

This validation asks whether an agent can translate a user's Chatwork request into an exact `cwk` task, invoke it safely, and understand the task result without guessing or routine external reconstruction. Candidate C is the first accepted context-capsule presentation; this document also defines how future candidates are compared before replacing it.

## Interaction budgets

- Unknown outcome to complete scoped contract: at most two discovery invocations.
- Known command path to complete invocation: one scoped-help invocation.
- Canonical reference reuse: no discovery or transformation invocation between producer and consumer.
- Supported task reconstruction: zero `jq`, `grep`, custom parsers/joins, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Failure recovery: the next corrective command comes from structured metadata.

Direct extraction of a declared canonical reference or fact is allowed. Rebuilding semantics that `cwk` claims to provide is not.

Provider-call evaluation uses the first-implementation ceilings: one attempt, 20 seconds for metadata/read/non-upload operations, 60 seconds for upload, 8 MiB successful response, 64 KiB provider error, 16 MiB output, 10,000 aggregate list items, five documented 100-item endpoint results, and 5 MiB upload. A transcript fails if it raises a limit, hides a lower provider bound, or treats a bound failure as partial success.

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

## Candidate-C baseline and future parallel-worktree competition

The first complete implementation tests candidate C directly against the semantic fixture and exact answers. It must preserve canonical references, bounds, unresolved relationships, hostile-text framing, deterministic bytes, and zero external reconstruction. These tests form the baseline that a later experiment may not silently weaken.

For a future replacement, before experimental implementation the competition work packet pins:

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

Mutation probes also require the agent to distinguish the three typed policies without guessing: ordinary exact invocation, `--confirm=access-change`, and `--confirm=destructive`. The access-change fixture covers membership/link/contact exposure; the destructive fixture covers room leave/delete, message deletion, invite-link deletion, and request rejection. Missing/wrong confirmation must make zero provider calls, and an uncertain outcome must select the declared read-only reconciliation task rather than repeat the mutation.

Authentication probes require the agent to:

1. select `auth login` directly for the declared single local authentication
   target, supply a public client ID only on first login, and use no synthetic
   profile discovery or selector;
2. distinguish an authoritative environment method from the exact persisted
   OAuth selection without inferring a preference from available credentials;
3. keep PATs, callbacks, codes, PKCE verifiers, access/refresh tokens,
   and credential-store keys out of argv and successful output;
4. distinguish unconfigured/expired OAuth state from insufficient API
   scope and from a credential-store access fault;
5. recover an uncertain login/logout store outcome through exact read-only
   `auth status`, not by repeating the mutation;
6. understand that logout removes local credentials and does not prove remote
   token revocation.

The synthetic OAuth transcript uses a fixed public client and redirect,
successful/failing browser opener, state mismatch case, PKCE failure case, full
callback supplied through stdin, strict fake user configuration, fake OS
credential store, expiry/refresh race, and secret canaries. It
records zero provider task calls for selection, callback, store, identity,
scope, and refresh rejection. Live credentials and browser history are never
evaluation inputs or retained evidence.

## No-post-processing audit

The transcript fails when a supported task contains an external parser, manual identifier join, raw notation parsing, guessed command/endpoint/cursor, or undeclared provider call. When this happens, decide whether the capability is incomplete, the outcome is too broad, or the presentation candidate failed. Do not patch the agent prompt with the workaround.

## Runnable public probes

Use the public Chatwork catalog and synthetic authentication/adapter fixtures:

```sh
go run ./cmd/cwk help --format agent
go run ./cmd/cwk help rooms --format agent
go run ./cmd/cwk help messages list --format agent
go test ./internal/cli -run 'TestChatwork|TestAgent'
```

These prove bounded discovery, scoped contracts, structured output/error behavior, and exact Chatwork reference reuse without requiring a developer account. The candidate-C semantic and golden tests separately validate the selected presentation.

## Review record

Record the natural-language outcome, discovery/task transcript, external-processing count, provider-call bounds, semantic answers, canonical references, recovery choices, per-run token/byte/latency measurements, agent/model versions, worktree/commit, failures, and variance. Preserve candidate evidence even when it loses so later thesis revisions can distinguish format failure from model or fixture drift.
