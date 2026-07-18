# Agent Readiness Validation

This validation asks whether an agent can translate a user's Chatwork request into an exact `cwk` task, invoke it safely, and understand the task result without guessing or routine external reconstruction. Candidate C (`cwk-context-capsule/1`) is the first stable presentation baseline. The current default is the P-derived task projection (`cwk-task-projection/1`), adopted by an explicit owner compatibility decision after Competition 1 was inconclusive and hardened beyond the frozen candidate, not as its benchmark winner. This document also defines how future candidates are compared before another default change.

## Interaction budgets

- Unknown outcome to complete scoped contract: at most two discovery invocations.
- Known command path to complete invocation: one scoped-help invocation.
- Canonical reference reuse: no discovery or transformation invocation between producer and consumer.
- Supported task reconstruction: zero `jq`, `grep`, custom parsers/joins, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Failure recovery: the next corrective command comes from structured metadata.

Direct extraction of a declared canonical reference or fact is allowed. Rebuilding semantics that `cwk` claims to provide is not.

Provider-call evaluation uses the first-implementation ceilings: one attempt, 20 seconds for metadata/read/non-upload operations, 60 seconds for upload, 8 MiB successful response, 64 KiB provider error, 16 MiB output, 10,000 aggregate list items, five documented 100-item endpoint results, and 5 MiB upload. A transcript fails if it raises a limit, hides a lower provider bound, or treats a bound failure as partial success.

## Presentation-independent semantic fixture

Every authoritative presentation comparison requires a synthetic, publishable
fixture whose typed answer key includes:

- room, account, and message canonical identities;
- senders, multiple To recipients, explicit replies, and quotes;
- one resolved relation and one referenced object outside the result bound;
- stable source ordering and duplicate behavior;
- exact retrieval bounds, partiality, missing context, and uncertainty;
- repeated values that may reward compression;
- hostile text resembling provider notation, presentation structure, JSON, agent instructions, controls, bidi/zero-width formats, line separators, delimiters, and pre-existing escapes.

The answer key contains semantics, not an expected rendering. Candidate
worktrees may not edit it. Competition 1 retained its frozen fixture and key
unchanged, but the audit found that its key omitted one explicit To relation
from a message that also contained a reply. That defect is why the experiment
selected no winner. A future comparison must correct the requirement in a new
versioned corpus before candidate work begins; it must not rewrite the retained
Competition 1 evidence in place.

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

## Stable C baseline, Competition 1 evidence, and current default

The first complete implementation tested candidate C directly against the semantic fixture and exact answers. It preserves canonical references, bounds, unresolved relationships, hostile-text framing, deterministic bytes, and zero external reconstruction. Those results remain the first-stable baseline that later experiments may not silently weaken or rewrite.

Competition 1 was inconclusive: benchmark/oracle defects and recovery-prompt ambiguity made its promotion result non-authoritative. Raw runs, score summaries, audit findings, and known defects remain evidence. They must not be discarded, corrected in place, or relabeled to imply that candidate P won.

After that experiment, the project owner made a separate compatibility decision to select a P-derived task projection as the default. Frozen candidate P supplied the implementation seed; the integrated projection adds semantic kind hardening and further subtraction that were not part of its ineligible score. The decision accepts a breaking migration from `cwk-context-capsule/1` to `cwk-task-projection/1`: old capsule headers, dictionaries, aliases, ordering, and grammar are not preserved. The semantic answer, exact canonical-reference identity, bounds/completeness/uncertainty, and external-text trust classification remain required.

The current task projection is subtractive. It emits only catalog-declared task fields, exact canonical references, task-relevant bounds/completeness/uncertainty, and trust framing for external text. It emits no display aliases, semantic records derived from raw Chatwork notation, provider/wire extras, duplicated coverage prose, or helpful non-contract defaults. Declared message bodies remain visible untrusted data and cannot inject CLI-authored structure.

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

The minimum understanding-quality floor is set before results are viewed. Lower token use cannot compensate for falling below it. A winner, reviewed combination, or another experiment is an acceptable experiment conclusion. If the experiment is inconclusive, it establishes no winner. A separately recorded owner compatibility decision may still supersede the default when it names the evidence limits and explicitly accepts any breaking migration; it must not be presented as a benchmark result.

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

1. identify `CWK_API_TOKEN` from exact scoped help as the sole required
   credential input and `pat` as the sole admitted method;
2. invoke the requested Chatwork task without probing for a login, profile,
   method selector, stored configuration, or credential status command;
3. keep the token out of argv, command literals, stdout, stderr, fixtures,
   diagnostics, and persistent project or user configuration;
4. distinguish missing or invalid token input from a valid account that lacks
   permission;
5. recover `chatwork_token_missing` and `chatwork_token_invalid` through the
   exact scoped-help action declared by the command, not a removed
   authentication namespace;
6. understand that the token selects one account for one command process and
   that `cwk` neither persists nor revokes it.

The synthetic PAT transcript supplies a canary only through the test process
environment, admits one PAT-only session, forwards its ephemeral binding
unchanged, and checks the exact `x-chatworktoken` header at a local server. It
records zero provider task calls for missing/invalid token, binding mismatch,
and permission rejection. It also verifies that an ambient obsolete
`CWK_AUTH_METHOD` value cannot select a different adapter. Live credentials are
never evaluation inputs or retained evidence.

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

These prove bounded discovery, scoped contracts, structured output/error behavior, and exact Chatwork reference reuse without requiring a developer account. Candidate-C evidence validates the first-stable baseline. Current task-projection semantic, subtractive-field, hostile-text, canonical-reference, and golden tests validate `cwk-task-projection/1` as the selected default.

## Review record

Record the natural-language outcome, discovery/task transcript, external-processing count, provider-call bounds, semantic answers, canonical references, recovery choices, per-run token/byte/latency measurements, agent/model versions, worktree/commit, failures, variance, benchmark defects, and the decision authority. Preserve candidate evidence when it loses or the experiment is inconclusive so later thesis revisions can distinguish format failure from model, fixture, prompt, oracle, or scoring drift.
