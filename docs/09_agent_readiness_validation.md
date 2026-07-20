# Agent Readiness Validation

This validation asks whether an agent can translate a user's Chatwork request into an exact `cwk` task, invoke it safely, and understand the task result without guessing or routine external reconstruction. Candidate C (`cwk-context-capsule/1`) is the first stable presentation baseline. A P-derived task projection (`cwk-task-projection/1`) was adopted by an explicit owner compatibility decision after Competition 1 was inconclusive and hardened beyond the frozen candidate, not as its benchmark winner. The current default is its further reviewed headerless subtraction. This document also defines how future candidates are compared before another default change.

## Interaction budgets

- Unknown outcome to complete scoped contract: at most two discovery invocations.
- Known command path to complete invocation: one scoped-help invocation.
- Canonical reference reuse: no discovery or transformation invocation between producer and consumer.
- Supported task reconstruction: zero `jq`, `grep`, custom parsers/joins, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Failure recovery: the next corrective command comes from structured metadata.

Direct extraction of a declared canonical reference or fact is allowed. Rebuilding semantics that `cwk` claims to provide is not.

Provider-call evaluation uses the first-implementation ceilings: one attempt, 20 seconds for metadata/read/non-upload operations, 60 seconds for upload, 8 MiB successful response, 64 KiB provider error, 16 MiB output, 10,000 aggregate list items, five documented 100-item endpoint results, and 5 MiB upload. A transcript fails if it raises a limit, hides a lower provider bound, or treats a bound failure as partial success. Local message `--start-index`/`--count` selection is scored separately from the 100-message `source-limit`; it neither reduces provider response bytes nor creates provider pagination.

## Presentation-independent semantic fixture

Every authoritative presentation comparison requires a synthetic, publishable
fixture whose typed answer key includes:

- room, account, and message canonical identities;
- senders, multiple To recipients, explicit replies, and quotes;
- one resolved relation and one referenced object outside the result bound;
- stable source ordering and duplicate behavior;
- typed send-time ordering, including non-monotonic provider order and equal-time
  ties, when an outcome selects newest messages;
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

Using only root/namespace indexes, exact-command help, and one candidate's output, the agent must:

1. choose the exact room-discovery and bounded-message tasks;
2. identify the exact room reference used;
3. identify each requested sender, recipient, reply, and quote fact;
4. distinguish explicit resolved, explicit unresolved, and absent relationships;
5. state the retrieval bound and whether the result represents complete room history;
6. distinguish candidate count, one-based start index, requested count, actual
   items per page, and next start index from the provider source bound,
   including context added beyond the requested count;
7. select the canonical reference required by a declared next command;
8. select recovery from typed failure metadata.

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

After that experiment, the project owner made a separate compatibility decision to select a P-derived task projection as the default. Frozen candidate P supplied the implementation seed; the integrated projection added semantic hardening and subtraction that were not part of its ineligible score. A later owner review made a second pre-1.0 compatibility decision to remove the repeated `cwk-task-projection/1 task=...` preamble and standalone provider-oriented coverage record. The latest owner decision refines `messages list` into a flat chronological adjacency list with one actor dictionary; it explicitly superseded an indented-tree proposal before implementation. Historical grammars are not preserved as selectable alternatives. The semantic answer, exact canonical-reference identity, bounds/completeness/uncertainty, and external-text trust classification remain required.

The current headerless task projection is subtractive. It starts directly with the result noun and emits only catalog-declared task facts, exact canonical references, task-relevant bounds/completeness/uncertainty, and trust framing for external text. Seven reviewed homogeneous collections declare one fixed schema and trust boundary, then emit provider-order positional records without aliases. `messages list` emits one fixed local schema line, deterministic document-local actor aliases with canonical dictionary entries, and one physical message record per selected typed item in original provider order. Without sender or index selection, output includes every provider item. An active selection adds one record with source count, optional exact senders, candidate count, applied start index, requested count, actual items per page, optional next start index, context, and primary anchors; gapped `#sequence` values retain the original window positions. The provider ceiling is separately named `source-limit`. Typed send time establishes newest-first rank and later provider position breaks equal-time ties, but neither changes physical output order. Direct reply context follows primary index/count selection and may increase displayed count beyond the requested count. The record's second field is the exact canonical message reference accepted unchanged by the next command; fixed message/time/body positions replace repeated labels. Explicit typed reply, To, quote, and unresolved facts remain distinguishable; depth/thread/root/children, absent relation shells, and resolved-default labels are omitted. No presentation derives semantic records from raw Chatwork notation, and declared external text remains visible untrusted data that cannot inject CLI-authored structure.

The active message probes use the shortest common invocation: omitted
`--window` means the latest bounded `recent` window. The explicit
`--window changes` form is reserved for a task that deliberately requests the
provider differential window. Readiness must prove this default from scoped
help and runtime behavior without an exploratory command.

The active file-collection probe uses one synthetic `files list` result. An
agent selects a named file without external processing, passes position one to
`files show --file` and position two to `files show --room`, preserves provider
order, and recognizes the fourth-position `absent` atom as missing state rather
than a command reference.

The active message-sender-selection probe asks for messages from two exact
accounts while retaining direct reply context. One synthetic bounded window
contains interleaved speakers, branches, a deeper chain, hostile text, and raw
`[To]`/`[rp]` canaries. The agent must identify repeated sender inputs as OR,
distinguish sender-match anchors from added one-hop reply nodes, preserve gapped
source sequences, decline transitive/body-derived expansion, and reuse a
displayed canonical message reference. The budget is one provider task call and
zero external post-processing calls.

The active message-index probe first asks for the newest two primary messages,
then asks for ranks 3 through 5 without repeating the first result, over a
bounded source whose provider order is not timestamp order. The agent must
choose `--count 2` and then `--start-index 3 --count 3` without redundant window
flags; identify sender-before-rank, rank-before-index/count, and selection-before-context
composition; distinguish candidate/start/requested/actual/next metadata from
`source-limit=100`; preserve original source sequences and canonical references;
and understand that explicit direct reply context may make displayed count
exceed the requested count. The budget is two provider task calls using only
documented `force`, no provider cursor or offset, and zero external
post-processing calls. Invalid index/count values and an over-coverage source
must fail before provider I/O or local selection, respectively.

The active message-period probe asks one question whose complete answer lies on
one Tokyo calendar day inside a synthetic maximum-100 source window. The agent
must choose `--on` without calculating or externally filtering timestamps,
interpret effective inclusive-since/exclusive-until bounds, retain only
in-period primary anchors, and understand that an explicitly requested direct
reply neighbor may lie outside that period. A fixed-clock variant proves
`today` and `yesterday` around Tokyo midnight. The budget is one unchanged
provider task call and zero external post-processing; semantic answer quality
must match the full source before token reduction is credited.

The active message-relation-closure probe reproduces T1 with a latest-100
source whose visible Aurora follow-up points to one out-of-window parent, which
points to a second out-of-window parent containing the owner and deadline. The
agent invokes the shortest `messages list` once. Its default-five budget drives
two internal exact reads in breadth-first chain order, and the output reports
limit five, attempts two, both canonical targets, and fetched provenance. The
budget is three provider requests inside one task invocation and zero external
post-processing or explicit `messages show` commands. The answer must not claim
arbitrary history, To/quote/body expansion, or an unbounded thread.

The active message-period-reachability probe reproduces T2 by requesting Tokyo
day 2026-07-08 from a trustworthy latest-100 source whose oldest item is on
2026-07-17. One list invocation must return zero candidates together with
`out-of-reachable-window` and the exact oldest boundary. The agent must stop
without probing adjacent dates, treating the day as truly empty, or issuing an
exact-message read. The budget is one provider request and zero external
post-processing.

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
- malformed notation that preserves the escaped body with unknown relations, versus malformed/oversized wire responses that fail closed;
- bounded results with missing referenced context;
- output write failure with no zero-status partial success;
- future mutation rejection and unclassified post-mutation outcomes.

The rate-limit probe includes four distinct fixtures: a read with a valid
future `x-ratelimit-reset`, a read with missing or malformed timing, a
message/task room-post mutation with the exact documented 10-second error, and
a mutation with only general or unknown timing. The agent must use a displayed
wait only when present, use a valid provider `Date` as the duration baseline
despite local clock skew, say that absent timing is unknown, reject
`Retry-After`-only evidence, and distinguish `retryable: true` read recovery
from `retryable: false` mutation help. It must never interpret the mutation's
known delay as permission to replay or expose the bounded provider body.

Message truthfulness probes separately cover normal `204` zero, partial
`200`, fully restricted `204`, exact restricted `404`, and ordinary not-found
`404`. The agent must not call a differential zero a missing room history or
treat a partial/all restriction as complete. Another fixture places malformed
To/reply/quote/code notation between valid list records; the agent must retain
the terminal-safe body and sibling records, report that relation set as
unknown rather than absent, and avoid inventing a relation from prose or quote
metadata.

Mutation probes also require the agent to distinguish the three typed policies without guessing: ordinary exact invocation, `--confirm=access-change`, and `--confirm=destructive`. The access-change fixture covers membership/link/contact exposure; the destructive fixture covers room leave/delete, message deletion, invite-link deletion, and request rejection. Missing/wrong confirmation must make zero provider calls, and an uncertain outcome must select the declared read-only reconciliation task rather than repeat the mutation.

The room-create probe starts from `account show`, passes its exact account
reference through `--account`, and verifies one same-binding `/me` check before
one room POST. A second fixture returns a different synthetic `/me` account and
requires `authentication_context_mismatch`, zero POSTs, no owner claim, and no
identity/token leakage. Cancellation, rate limiting, and unavailability during
that preflight must remain known zero-mutation outcomes rather than unknown
room creation. A generic or mismatched binding, an overlong name, and an
unknown icon preset must also fail before the room POST.

The invite-link probe rejects empty, code-only, approval-only, description-
missing, invalid-code, and code-plus-regeneration invocations with zero
authentication/provider calls. Successful fixtures cover explicit code and
explicit regeneration, inspect the full form, and require the resulting URL,
approval, and description to be understandable without treating omitted
provider fields as preserved. The agent must explain that description clear is
unsupported because the official empty/omission transition is not established.

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

## Command-attention probe

The active command-selection scenario uses a temporary synthetic config home
plus a synthetic interactive terminal and fake PAT/provider counters; it never
reads or changes a developer's real preference and needs no live Chatwork data.
Starting with no preference file, the agent observes the complete current help
view and the four always-on commands: `help`, `doctor`, `version`, and `config`.
It opens the single `config` selector, uses Up and Down to move and Space to
disable `contact-requests list`, `contact-requests accept`, and
`contact-requests reject`, then presses Enter to save. The selector displays
catalog-derived `[read]`, `[create]`, and `[write]` labels; optional color may
reinforce an effect but never replaces its textual label.

The scenario must prove all of the following without inspecting source or
editing JSON directly:

1. root and namespace human help, exact and trailing help, root and namespace
   agent indexes, exact-command agent help, recovery actions, and reference
   workflows expose none of the disabled paths; the empty contact-request
   namespace is not advertised;
2. invoking a disabled path returns the ordinary `unknown_command` result with
   zero PAT-resolution and zero provider calls;
3. the confirmed save result reports visible/hidden/change counts in the
   reviewed natural-Japanese transcript and conditionally reports only nonzero
   cleanup, without internal key/value labels or a fingerprint; always-on
   `doctor` reports valid state, source, enabled/disabled/stale/legacy counts,
   and the actual fingerprint without performing a second mutation; an
   uncertain-save JSON fixture carries the candidate fingerprint, follows the
   exact message grammar published by exact-command agent help, and distinguishes
   `source=saved` from an identical `source=default` fingerprint;
4. a second `config` run marks the three paths off and can re-enable them from
   its complete catalog-derived selector, after which normal and agent help
   expose them again;
5. re-enabled contact-request commands still enforce their original PAT,
   canonical-reference, provider-permission, and `--confirm=access-change` or
   `--confirm=destructive` contracts; selection grants no authority; and
6. removing the saved preference restores the documented all-current-commands
   view, demonstrating why this feature is attention curation rather than a
   security control.

The saved-profile variant injects one synthetic catalog addition and verifies
that it remains off until selected. The invalid-state variant proves ordinary
commands and root help do not silently fall back to all enabled or a false
empty view. Config help remains available; malformed serialized content may
enter the `config` repair selector, while unsafe or inaccessible filesystem
state requires local repair followed by always-on `doctor`. A legacy profile
containing formerly selectable `doctor` or `version` paths loads the remaining
Chatwork selection without offering either local command as a toggle, reports
the legacy count through doctor, and removes those entries only after Enter
saves the normalized replacement. That confirmed save must emit one combined
`古い設定を2件整理しました。` cleanup line rather than separate legacy or
stale key/value fields.

The invalid-view selector stays open after Enter, renders the exact actionable
dependency/recovery diagnostic, and writes nothing until the draft is repaired
and Enter is pressed again. q, Escape, Ctrl-C, EOF, input failure, and blocked
input context cancellation preserve the last saved bytes. Every path that
leaves the selector restores the alternate screen, cursor, Windows output mode
when applicable, and raw input mode exactly once. Restoration occurs before
the save boundary; a restoration failure prevents persistence, while a late
cancellation after confirmed save cannot reclassify success. If either stdin
or stdout is not a TTY, `config` emits the typed
`interactive_terminal_required` failure, no selector/ANSI transcript, and no
write.

The terminal fixture also checks deterministic viewport rendering, fragmented
CSI/SS3 arrow sequences, and textual effect labels that remain after ANSI color
spans are removed. Infrastructure tests cover `x/term` raw mode and setup
rollback; supported Unix and Windows cross-builds prevent platform-specific
mode code from becoming a native-only success.
Catalog-mutation fixtures prove the always-on line and failure-recovery island
change with catalog metadata. Narrow-width and low-height fixtures prove Space
and Enter cannot act until the complete current command identity is visible.

## No-post-processing audit

The transcript fails when a supported task contains an external parser, manual identifier join, raw notation parsing, guessed command/endpoint/cursor, or undeclared provider call. When this happens, decide whether the capability is incomplete, the outcome is too broad, or the presentation candidate failed. Do not patch the agent prompt with the workaround.

## Runnable public probes

Use the public Chatwork catalog and synthetic authentication/adapter fixtures:

```sh
go run ./cmd/cwk --help
go run ./cmd/cwk rooms --help
go run ./cmd/cwk rooms list --help
go run ./cmd/cwk help --format agent
go run ./cmd/cwk help rooms --format agent
go run ./cmd/cwk help messages list --format agent
go run ./cmd/cwk help files list --format agent
go run ./cmd/cwk config --help
go test ./internal/cli -run 'TestChatwork|TestAgent|TestRootTextHelp|TestTrailingHelp|TestProductionHelp|TestConfig|TestCommandSelection'
go test ./internal/infra/terminalui
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
go test ./tools/presentationeval -run 'TestActive(FileCollection|MessageAdjacency|MessageSenderSelection|MessageIndex|MessagePeriod|MessageRelationClosure|MessageReachability)'
```

These prove the bounded human root-to-namespace-to-command navigation, the
direct machine root-to-scope path, scoped contracts, structured output/error
behavior, and exact Chatwork reference reuse without requiring a developer
account. The config and terminal probes use only synthetic streams, preference
state, and adapters; the public selector itself is exercised manually only with
an attached stdin/stdout TTY and a temporary config home. Candidate-C evidence
validates the first-stable baseline. Current headerless task-projection semantic, subtractive-field, hostile-text, canonical-reference, all-route, and golden tests validate the selected default. The active flat-message scenario additionally proves provider order, branch/interleaving recognition, unresolved-parent handling, one-line hostile-text framing, and reuse of a displayed canonical message reference as the next exact command input. The active sender-selection scenario proves exact sender OR semantics, direct typed reply context, stable gapped source sequences, anchor/context distinction, one bounded provider call, and zero external post-processing. The active file-collection scenario proves the fixed six-position schema, canonical file/room reuse, explicit missing-message state, hostile filename containment, and zero external post-processing.
The broader message-index tests prove exact 1..100 validation, deterministic
equal-time ties, a `force`-only provider request, and no provider pagination. The active
scenario proves typed-time rank selection, unchanged provider-order output,
source/candidate/start/requested-count/items-per-page/next-start distinction,
continuation without repeated ranks, context beyond requested count, two
provider calls, the omitted-window recent default, canonical-reference reuse,
and zero external post-processing. A separate runtime fixture preserves exact
`--window changes` differential behavior.
The active message-period scenario proves sender-and-period composition,
fixed-clock Tokyo day resolution, concrete effective bounds, unchanged source
coverage/provider calls, context outside the primary period only through typed
reply edges, semantic-answer retention, and zero external post-processing.
The active relation-closure scenario proves the default-five public contract,
recursive same-room parent discovery, two bounded internal exact reads,
canonical supplemental context, and no external command or parser. The active
reachability scenario proves that a wholly older requested day is reported as
unreachable after one list call rather than as empty or as a prompt for date
probing.

## Review record

Record the natural-language outcome, discovery/task transcript, external-processing count, provider-call bounds, semantic answers, canonical references, recovery choices, per-run token/byte/latency measurements, agent/model versions, worktree/commit, failures, variance, benchmark defects, and the decision authority. Preserve candidate evidence when it loses or the experiment is inconclusive so later thesis revisions can distinguish format failure from model, fixture, prompt, oracle, or scoring drift.
