# Agent Readiness Validation

This validation asks whether an agent can translate a user's Chatwork request into an exact `cwk` task, invoke it safely, and understand the task result without guessing or routine external reconstruction. Candidate C (`cwk-context-capsule/1`) is the first stable presentation baseline. A P-derived task projection (`cwk-task-projection/1`) was adopted by an explicit owner compatibility decision after Competition 1 was inconclusive and hardened beyond the frozen candidate, not as its benchmark winner. The current default is its further reviewed headerless subtraction. This document also defines how future candidates are compared before another default change.

## Interaction budgets

- Unknown outcome to complete scoped contract: at most two discovery invocations.
- Known command path to complete invocation: one scoped-help invocation.
- Canonical reference reuse: no discovery or transformation invocation between producer and consumer.
- Supported task reconstruction: zero `jq`, `grep`, custom parsers/joins, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Failure recovery: the next corrective command comes from structured metadata.

Direct extraction of a declared canonical reference or fact is allowed. Rebuilding semantics that `cwk` claims to provide is not.

Provider-call evaluation uses the first-implementation ceilings: one attempt, 20 seconds for metadata/read/non-upload operations, 60 seconds for upload, 8 MiB successful response, 64 KiB provider error, 16 MiB output, 10,000 aggregate list items, five documented 100-item endpoint results, and 5 MiB upload. A transcript fails if it raises a limit, hides a lower provider bound, or treats a bound failure as partial success. A local message `--limit` is scored separately from the 100-message `source-limit`; it neither reduces provider response bytes nor creates pagination.

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

Using only root/scoped help and one candidate's output, the agent must:

1. choose the exact room-discovery and bounded-message tasks;
2. identify the exact room reference used;
3. identify each requested sender, recipient, reply, and quote fact;
4. distinguish explicit resolved, explicit unresolved, and absent relationships;
5. state the retrieval bound and whether the result represents complete room history;
6. distinguish an optional primary-message selection limit and candidate count
   from the provider source bound, including context added beyond that limit;
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

The current headerless task projection is subtractive. It starts directly with the result noun and emits only catalog-declared task facts, exact canonical references, task-relevant bounds/completeness/uncertainty, and trust framing for external text. Seven reviewed homogeneous collections declare one fixed schema and trust boundary, then emit provider-order positional records without aliases. `messages list` emits one fixed local schema line, deterministic document-local actor aliases with canonical dictionary entries, and one physical message record per selected typed item in original provider order. Without sender or count selection, output includes every provider item. An active selection adds one record with source count, optional exact senders, candidate count and requested primary limit when count limiting is active, context unless it is the limit-only default `none`, and primary anchors; gapped `#sequence` values retain the original window positions. The provider ceiling is separately named `source-limit`. Typed send time selects newest-N membership and later provider position breaks equal-time ties, but neither changes physical output order. Direct reply context follows the primary limit and may increase displayed count beyond N. The record's second field is the exact canonical message reference accepted unchanged by the next command; fixed message/time/body positions replace repeated labels. Explicit typed reply, To, quote, and unresolved facts remain distinguishable; depth/thread/root/children, absent relation shells, and resolved-default labels are omitted. No presentation derives semantic records from raw Chatwork notation, and declared external text remains visible untrusted data that cannot inject CLI-authored structure.

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

The active message-limit probe asks for the newest two primary messages in one
bounded source whose provider order is not timestamp order. The agent must
choose `--limit 2`, identify sender-before-limit and
limit-before-context composition, distinguish the requested limit and candidate
count from `source-limit=100`, preserve original source sequences and canonical
references, and understand that explicit direct reply context may make displayed
count exceed two. The budget is one provider task call using only documented
`force`, no cursor or offset, and zero external post-processing calls. Invalid
limit values and an over-coverage source must fail before provider I/O or local
selection, respectively.

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

## Command-attention probe

The active command-selection scenario uses a temporary synthetic config home
and fake PAT/provider counters; it never reads or changes a developer's real
preference. Starting with no preference file, the agent observes the complete
current help view. It then runs the line-oriented `config edit`, disables
`contact-requests list`, `contact-requests accept`, and `contact-requests
reject`, and explicitly saves.

The scenario must prove all of the following without inspecting source or
editing JSON directly:

1. root and namespace human help, exact and trailing help, root and scoped agent
   help, recovery actions, and reference workflows expose none of the disabled
   paths; the empty contact-request namespace is not advertised;
2. invoking a disabled path returns the ordinary `unknown_command` result with
   zero PAT-resolution and zero provider calls;
3. `config show` identifies all three exact paths as disabled and states
   `security-boundary=false` while `help`, `config show`, and `config edit`
   remain visible;
4. a second `config edit` can re-enable the three paths from its complete
   catalog-derived selector, after which normal and agent help expose them
   again;
5. re-enabled contact-request commands still enforce their original PAT,
   canonical-reference, provider-permission, and `--confirm=access-change` or
   `--confirm=destructive` contracts; selection grants no authority; and
6. removing the saved preference restores the documented all-current-commands
   view, demonstrating why this feature is attention curation rather than a
   security control.

The saved-profile variant injects one synthetic catalog addition and verifies
that it remains off until selected. The invalid-state variant proves ordinary
commands and root help do not silently fall back to all enabled or a false
empty view. Config-scoped help remains available; malformed content can enter
`config edit`, while unsafe or inaccessible filesystem state requires local
repair and `config show`. The selector writes nothing before an explicit
`save`; cancellation, EOF, and a blocked-input context interruption before that
action preserve the prior bytes. A separate fixture cancels after a confirmed
save and proves that success is not reclassified as retryable cancellation.

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
go run ./cmd/cwk config show
go run ./cmd/cwk config edit --help
go test ./internal/cli -run 'TestChatwork|TestAgent|TestRootTextHelp|TestTrailingHelp|TestProductionHelp'
go test ./tools/presentationeval -run 'TestActive(FileCollection|MessageAdjacency|MessageSenderSelection|MessageLimit)'
```

These prove the bounded human root-to-namespace-to-command navigation, the
direct machine root-to-scope path, scoped contracts, structured output/error
behavior, and exact Chatwork reference reuse without requiring a developer
account. Candidate-C evidence validates the first-stable baseline. Current headerless task-projection semantic, subtractive-field, hostile-text, canonical-reference, all-route, and golden tests validate the selected default. The active flat-message scenario additionally proves provider order, branch/interleaving recognition, unresolved-parent handling, one-line hostile-text framing, and reuse of a displayed canonical message reference as the next exact command input. The active sender-selection scenario proves exact sender OR semantics, direct typed reply context, stable gapped source sequences, anchor/context distinction, one bounded provider call, and zero external post-processing. The active file-collection scenario proves the fixed six-position schema, canonical file/room reuse, explicit missing-message state, hostile filename containment, and zero external post-processing.
The broader message-limit tests prove exact 1..100 validation, deterministic
equal-time ties, a `force`-only provider request, and no pagination. The active
scenario proves newest-N selection by typed send time, unchanged provider-order
output, source/candidate/requested-limit distinction, context beyond N, one
provider call, canonical-reference reuse, and zero external post-processing.

## Review record

Record the natural-language outcome, discovery/task transcript, external-processing count, provider-call bounds, semantic answers, canonical references, recovery choices, per-run token/byte/latency measurements, agent/model versions, worktree/commit, failures, variance, benchmark defects, and the decision authority. Preserve candidate evidence when it loses or the experiment is inconclusive so later thesis revisions can distinguish format failure from model, fixture, prompt, oracle, or scoring drift.
