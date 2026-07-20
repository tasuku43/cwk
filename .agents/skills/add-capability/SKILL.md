---
name: add-capability
description: >
  Use when adding or changing a user-visible CLI command, workflow, integration,
  side effect, output contract, or external adapter. Guides thesis alignment,
  layer ownership, tests, documentation, and repository gates.
---

# Add a CLI Capability

Before designing the change, run:

```sh
go run ./tools/projectmeta --field profile
```

If the profile is `template`, stop and use `$bootstrap-derived-cli`. Do not add
a capability while the repository still has the foundry identity or generic
product reasoning.

Read `docs/00_theses.md` before designing the change. A capability is complete
only when its user outcome, safety boundary, discoverability, and verification
are explicit.

## 1. Define the user outcome

Write one sentence describing what the user can accomplish. Prefer task language
over an upstream API resource or implementation name.

Confirm:

- the capability belongs in this CLI;
- an existing command cannot express it naturally;
- input ambiguity has a deterministic resolution or is surfaced to the user;
- stdout, stderr, exit status, and machine-readable output are predictable;
- all side effects and external destinations are named.
- a supported agent outcome can be completed without `jq`, `grep`, a custom
  join/parser, raw provider-notation interpretation, source inspection, or an
  exploratory external call;
- the command's stable capability ID is `public` in `.harness/capabilities.json`, or the upstream capability remains explicitly `internal`, `deferred`, or `excluded` with a reason.

If the thesis does not decide a design trade-off, update the thesis or an
architecture decision before implementation.

For relationship-rich Chatwork reads, define the outcome's typed semantic
result and answer key before presentation. Raw provider JSON plus documented
post-processing does not complete the outcome. Candidate C
(`cwk-context-capsule/1`) remains the first-stable baseline. The P-derived
`cwk-task-projection/1` was selected by an explicit owner compatibility decision
after an inconclusive competition, not as its benchmark winner. The current
default is its further reviewed headerless subtraction. Its renderer starts
directly with the task result and emits only catalog-declared task fields, exact
canonical references, task-relevant bounds/completeness/uncertainty, and
external-text trust framing. The accepted `messages list` specialization emits
one fixed positional schema, one document-local actor dictionary, and flat
provider-order records; the second record field remains the exact canonical
message reference. The reviewed contacts, rooms, members, personal-task,
room-task, file, and contact-request lists likewise emit one trust/schema
prelude and positional provider-order records, but no aliases. Required
canonical positions do not shift; optional suffixes are final, and an absent
file-message reference remains the literal `absent`. Do not add a global
schema/task preamble, standalone provider coverage record, aliases outside the
reviewed message dictionary, raw provider notation as semantic structure, wire
extras, empty optional shells, or non-contract defaults. Record a repeated
external pipeline as thesis evidence.
Compare materially different future presentation candidates in
isolated worktrees under the protocol in
`docs/09_agent_readiness_validation.md` before replacing the current default,
and retain inconclusive raw results and benchmark defects.

Separate discovery from action. Discovery commands may accept ambiguity and
must return stable, opaque IDs. Acting commands accept one opaque ID or another
explicitly unique selector; they must not guess among candidates. Record each
ID producer, its output field, and every consumer so contract tests can prove
that users and agents can pass the value through unchanged.

## 2. Place responsibilities in the four layers

- `internal/domain`: domain types, invariants, and pure policy.
- `internal/app`: a use case and consumer-owned ports.
- `internal/infra`: external systems and concrete port adapters.
- `internal/cli`: argument parsing, composition, help, and rendering.

Domain code imports no outer layer. Application code does not import
infrastructure or CLI code. Infrastructure code does not import application or
CLI code. The CLI is the composition root. `tools/archlint` enforces this
contract from the module path reported by `go list`.

Keep policy out of transport adapters and presentation code. Inject clocks,
filesystems, environment reads, network clients, and side-effect executors at
the narrowest useful boundary.

For relationship-aware output, infrastructure parses reviewed provider
notation into typed facts, domain owns relation and coverage invariants,
application selects and assembles the bounded outcome result, and CLI owns only
presentation. Do not let a renderer infer relations or let an adapter choose
public output. Candidate-specific grammars and helpers must not leak into the
shared semantic layers.

When a recurring read needs filtering, prefer the smallest finite typed task
input that can state the supported outcome. Apply it once to the typed bounded
result in application, clear local-only fields before the provider port, and
return enough metadata to distinguish matches from added context without
reparsing presentation. Preserve source order/sequence and canonical references;
name matching, raw-body inference, hidden additional calls, and an unbounded
"related" mode are not acceptable shortcuts. For message reply context, test
the exact hop bound and prove To/quote/raw notation cannot expand it.

When the supported outcome begins with an external display name, keep name
matching in a discover command rather than accepting it in a reference-bound
act input. A reviewed local candidate filter may reuse one complete typed
provider collection after clearing the query before the port. It must preserve
source order, return every match with canonical references and explicit
query/source/candidate metadata, visibly escape external names, and never
auto-select even a unique result. Test zero, unique, and ambiguous candidates
plus direct reuse of the chosen reference by the next command.

For the reviewed Chatwork message index selection, declare optional one-based
`--start-index <index>` and maximum `--count <count>` with the inclusive range
1..100; count alone defaults start index to 1. Exact-sender OR matching precedes
typed-time newest-first ranking, later provider position breaks equal-time ties,
then index/count select primary anchors; direct typed reply context follows and
may increase displayed count beyond the requested count. Preserve provider
order and original source sequences in output. Keep candidate count, applied
start index, requested count, actual items per page, and optional next start
index separate from provider `source-limit=100`. This SCIM-derived vocabulary
still makes one provider request using only documented `force`; it is not a
provider query parameter, cursor, offset, or page and does not promise snapshot
stability across calls. Reject invalid values before authentication/I/O and
reject a provider result above declared coverage before local selection can
hide it.

For the reviewed Chatwork message period selection, declare optional inclusive
`--since <RFC3339>`, exclusive `--until <RFC3339>`, or their mutually exclusive
`--on <YYYY-MM-DD|today|yesterday>` shorthand. Exact bounds require an explicit
offset and whole seconds. `--on` uses the fixed `Asia/Tokyo` calendar; resolve
an injected clock once and carry only the effective concrete day/zone and
half-open Unix bounds into the typed request/result. Exact-sender OR and period
membership form the candidate predicate before typed-time rank/index/count;
direct reply context follows and may add an out-of-period record only as
explicit context. Clear every period field before the one `force`-only provider
call, retain `source-limit=100`, and never claim fewer response bytes, provider
date filtering, pagination, or older-history access. Reject ambiguous dates,
offset-free or fractional timestamps, conflicting selectors, and empty or
reversed intervals before authentication/I/O. Use fixed clocks in tests.

For the reviewed Chatwork reply-relation closure, declare optional
`--resolve-relations <count>` in 0..100 with public default five and zero as the
explicit opt-out. After local selection, traverse only typed explicit
same-room reply parents in breadth-first first-reference order. Reuse parents
from the original source without spending a slot; otherwise make at most one
exact read per unique target, and enqueue the same relation kind from attached
context until the budget is consumed. Use a visited set for duplicates and
cycles. Preserve supplemental context outside source sequences, publish fetch
limit/attempts and source/fetched/not-found/restricted/budget-exhausted target
evidence, and bind every exact result to the requested room/message. Retain
only reviewed not-found/restricted target outcomes; abort without partial
success on other faults. Clear the budget before every adapter call and prove
that To, quotes, names, times, prose, and cross-room references never expand it.

For message-period reachability, derive an oldest boundary only from a
nonempty, unrestricted `recent` typed source with valid send times. Classify a
requested period as within, partially outside, wholly outside, or unknown.
Differential, empty, limited, and unprovable sources stay unknown. Never turn a
wholly older period into an ordinary empty-day claim or imply provider
pagination/complete room history.

For the reviewed message-window default, omitted or explicit
`--window recent` selects the latest bounded provider window; only explicit
`--window changes` selects differential retrieval. Keep both modes at one
provider request with the same 100-message source ceiling. Catalog help must
name the default, and runtime tests must cover omission plus both explicit
values without moving this default into infrastructure.

## 3. Declare the operation contract

For every external action, specify:

- effect: read, create, or write; unknown is never executable;
- target, scope, and all generic impact dimensions (cardinality, notification, access change, destructive);
- for a reference-bound create, exactly one required argument/flag opaque `parent_input`, no `target_id_input`, and no other `target_inputs`;
- for a reference-bound write, one required argument/flag opaque `target_id_input` whose reference kind equals `TargetKind`, plus an optional distinct opaque parent role whose input is required when present; `target_inputs` contains only those bound roles;
- for the exceptional command-bound target, one catalog-declared fixed `tool_local` singleton with a stable kind/ID/description, an explicitly empty `target_inputs`, and no `parent_input` or `target_id_input`; never use this for a remote, provider-owned, user-selected, or potentially multiple target;
- validation performed before the external boundary;
- finite timeout, pagination/completeness, maximum attempts, and upstream idempotency behavior;
- whether each flag is repeatable as a structured catalog fact; parsing and
  scoped agent help must consume the same declaration rather than a parallel
  command-specific flag registry;
- which derived policy applies at `app/execution.Invoker`; do not make the template assume approval, confirmation, OS authentication, or dry-run;
- audit-safe fields and secret fields;
- allowed network destination.

Route all equivalent effects through one central enforcement boundary. A new
command must not create a second raw transport or bypass validation.

For the fixed Chatwork first implementation, use the checked policy in
`.harness/chatwork_api_v2.json`: one attempt; 20-second metadata/read and
non-upload timeout; 60-second upload timeout; 8 MiB success, 64 KiB provider
error, 16 MiB output, 10,000-item aggregate, documented 100-item endpoint, and
5 MiB upload ceilings. Ordinary exact creates/updates need no extra
confirmation. The reviewed access-changing operation set requires exact
`--confirm=access-change`; the reviewed destructive set requires exact
`--confirm=destructive`. Unknown mutation outcomes are non-retryable and name
only an exact read-only reconciliation task. Runtime code uses typed constants
and tests rather than reading the harness manifest dynamically.

## 4. Update the command catalog

Add the command to the canonical catalog and derive dispatch and help from that
entry. Complete its `AgentContract`: stable capability ID, user outcome,
described inputs and allowed values, formats, fields/types/descriptions,
completeness, non-auth prerequisites, optional secret-free authentication
requirement, stable faults with exact next commands, and mutation contract when
applicable. Nil collections mean unknown and are invalid; use explicit empty
collections for known none.

This product also derives a user-selected attention view from the complete
catalog. For each new leaf, deliberately declare whether it is configurable;
do not infer that choice from namespace, effect, or provider ownership. The
complete `DefaultCatalog` remains the capability/API/release contract, while a
saved exact-path allowlist controls only the active help and routing view. A
saved profile keeps a newly added configurable command off until selected;
missing profile state exposes only the local control plane and returns
`command_selection_required` for a known configurable path until `config` is
saved. In this product keep `help`, `doctor`, `version`, and the single exact
`config` write always-on; only Chatwork task leaves are selectable. Validate the active view
so visible required-reference consumers retain a reachable visible producer
and visible recovery actions resolve inside the view; do not silently
auto-enable a dependency. Human and agent help, trailing-help normalization,
workflows, recoveries, and routing must all consume the same active view.
If persisted selection state is invalid, do not render the always-on control
plane as though it were a deliberate empty root view. Keep config-scoped help
reachable, distinguish repairable serialized content from unsafe or
inaccessible storage, and route the latter to external repair plus read-only
`doctor` diagnostics. Malformed serialized content may enter the explicit
`config` repair selector, but it still writes only after Enter. Existing
profiles may normalize only the two formerly selectable exact paths `doctor`
and `version`; do not generalize that compatibility rule to arbitrary always-on
paths. Cancellation can promise an unchanged profile only before the save
action; after replacement is attempted, use the `doctor` fingerprint for
uncertain-outcome reconciliation, and never overwrite confirmed success with
late cancellation.

Keep confirmed `config` success human-facing: report visible, hidden, and
changed counts in natural Japanese, add cleanup only when nonzero, and do not
repeat internal key/value labels or the reconciliation fingerprint. The
candidate fingerprint belongs to an uncertain-save fault; the actual
source/fingerprint belongs to read-only `doctor`. This split must remain exact
in catalog fields, golden output, and readiness evidence.

Keep the terminal boundary layered. `config` is a fixed-`tool_local` write, not
a read command with a hidden save. CLI owns the pure selector model, key
semantics, catalog-derived rows, viewport, and presentation. Infrastructure
alone imports `golang.org/x/term` and owns terminal detection, raw/alternate
screen modes, sizing, and restoration. Require interactive stdin and stdout;
non-TTY input fails before persistence. Up/Down and Space change only a draft;
accept ASCII Space and fragmented UTF-8 U+3000 full-width space as the same
toggle so a Japanese input method does not make the advertised key inert. On
Enter, validate active-view closure and the fixed-target request, restore the
terminal, then invoke save; a restoration failure must make zero save calls.
Quit, Escape, EOF, terminal closure, and pre-save cancellation leave the last
saved profile unchanged on every graceful path.

Every selector row keeps a textual `[read]`, `[create]`, or `[write]` badge.
Cyan, yellow, and magenta may supplement those badges, but color is never
effect truth or authority and red is not used as a generic write cue. Pad the
badge field to the longest visible label so exact command paths align across
effects. Add ANSI only after terminal-safe text projection and width truncation
so hostile text cannot author terminal structure and color bytes do not consume
display width.

Treat this selection as cognitive-surface curation only. It is not
authorization, sandboxing, provider scope, or mutation approval, because a
local actor can edit or delete the preference and restore commands. Enabling a
command must retain its PAT, exact-reference, effect/intent, permission, and
confirmation contracts. Keep command-selection state free of credentials and
provider data and separate from authentication configuration.

For `complete` output, do not declare a pagination binding. For deliberately
`paged` output, declare `AgentContract.Pagination` with the exact optional
cursor argument/flag and exact top-level string cursor output field. The cursor
is emitted beside `schema_version` and the JSON envelope; `CommandOutput.Fields`
describe only values inside the envelope. Both cursor endpoints must use one
dedicated opaque reference kind, and no extra input or output may reuse that
cursor kind. A paged command supports only JSON and defaults to JSON. Pass the
emitted cursor back unchanged. Declare the only supported completion rule,
`completion: "empty_cursor"`, and emit the top-level cursor on every successful
page. An empty string is complete; omission, JSON `null`, and non-string values
are contract failures. Do not use `paged` to make silent truncation or a local
traversal limit look successful.

For a mutation, fill `MutationContract` from required argument or flag inputs rather than
maintaining an informal target description. Treat a missing or unbound role,
optional or non-CLI target, duplicate or extra target input, non-reference input, or target-kind mismatch as
a catalog error. Do not defer that ambiguity to command parsing, policy, or the
adapter.

Do not hand-maintain `ProducedRef` or `ConsumedRef`. Reference compatibility,
workflows, and next actions derive from structured input/output reference kinds.
An act command must require at least one opaque reference unless its exact path
declares the one fixed `tool_local` singleton described above. A fixed-target
act command produces and consumes no target references. Give semantically
different references different kinds; sharing a kind declares them
interchangeable across every matching field/input edge. Ensure required
reference chains lead back to a command that can run without an unresolved
required reference rather than forming a closed cycle.
Verify that root agent help adds only the command's path, namespace, summary,
capability, outcome, effect, and role. Then use an exact-command invocation to
verify the complete contract and touching workflows. A namespace invocation is
a compact index with exact-command pointers, never an aggregate of complete
contracts. Root and namespace agent help must not regain inputs, output detail,
authentication, errors, mutation facts, or workflows as the catalog grows, and
each encoded command entry must remain within the 512-byte catalog budget.
Keep human root help as a catalog-derived navigation projection: directly
runnable single-word commands plus one entry per top-level namespace. Exact
leaf summaries belong in namespace help, and each displayed selector must
round-trip through the same catalog rather than a separate help registry.
If a human workflow genuinely benefits from a short "when this, do that"
recipe, store structured exact command steps on catalog command metadata.
Render it only in exact-command human help when the selected command is one of
its steps and every step exists in the active view. Keep root/namespace human
help as indexes and keep recipes out of every agent-help shape.
Exact human help must derive input requirements, repeatability, source, allowed
values, reference kind, and descriptions from `AgentContract.Inputs`; do not
reconstruct that contract from usage prose. Validate every projected input name
as terminal structure regardless of whether its source is argv, environment,
configuration, or stdin.

Keep every recovery `command` executable under the template's small grammar:
use one exact catalog path, or `help` plus an exact path/canonical namespace.
Do not append flags, values, or guessed selectors. If a derived product needs a
fixed argument-bearing recovery, introduce a typed argument contract and
parser-aware validation before publishing it.

Add bidirectional contract tests so every public catalog entry has a dispatch,
help, and fixture, and no removed or internal entry remains exposed. For JSON,
compare the executable schema version, envelope, and item keys with
`CommandOutput`; do not accept a declaration-only or renderer-only change.

Keep command paths disjoint from their word-boundary namespaces. Match argv
`Required` flags to bracketed versus non-bracketed usage syntax and keep a
written `a|b` list exactly aligned with `AllowedValues`; do not apply this
grammar to stdin, environment, or configuration inputs. Declare the common
cancellation/output failures and, for mutations, every standard invoker
contract or policy failure before exposing the command.

Treat mutation cancellation by phase. Before the action, `operation_canceled`
is retryable because the invoker proves zero attempts. After the action begins,
return a valid structured adapter fault for a known classification and let the
invoker strip its private cause. Never return a raw cancellation as proof that
the write did not happen. Unclassified post-action errors become non-retryable
`unclassified_mutation_outcome`; declare that code in the mutation catalog and
point its next action only to an exact read/discover reconciliation command.

## 5. Add authentication and API boundaries only when needed

Read `docs/07_authentication.md` and `docs/08_external_api_contracts.md` for an
external API capability.

- Keep raw PATs, OAuth tokens, refresh material, token sources, authorization
  headers, and credential-store handles inside `internal/infra`.
- Use `app/authn.Gate` with a secret-free requirement/session and prove auth,
  permission, mismatch, and cancellation failures make zero downstream calls.
- Treat non-nil `AgentContract.Authentication` as a binding to that gate. Declare
  every standard gate fault with its exact code, kind, and retryability, plus
  each provider-specific pass-through fault and a command-valid recovery action;
  catalog validation rejects an incomplete standard set before dispatch.
- Make each authenticated application task port accept `authn.BindingID`.
  Pass the ID from the validated `Session` unchanged; the infrastructure
  adapter resolves that process-local binding and revalidates or refreshes the
  exact private authentication record immediately before I/O. Never pass an
  OAuth client, token source, PAT wrapper, authenticated transport, provider
  SDK type, or credential-store handle into application code.
- Call `authn.NewBindingID` only from production infrastructure. Architecture
  lint rejects issuance in domain, application, CLI, and command packages, so
  argv or configuration values cannot be promoted into authentication bindings.
- Treat gate expiry as admission metadata rather than a lease. Keep refresh
  thresholds, reuse, cache/storage, account selection, and approval policy in
  the derived security model, while making stale or mismatched bindings fail
  before a provider task request.
- Do not implement OAuth protocol machinery. Add a reviewed OAuth library only
  in a derived project whose accepted security model selects OAuth.
- For this Chatwork implementation, follow the PAT-only decision that
  supersedes `docs/decisions/0002-chatwork-oauth-public-client.md`.
  `CWK_API_TOKEN` is the sole command-process credential source. Do not accept
  it through argv, persist it, add a credential-method selector, or retain an
  OAuth fallback.
- Resolve the PAT into one private infrastructure record, issue only a
  secret-free process-local binding, and revalidate that exact record before
  attaching `x-chatworktoken` to the fixed API destination. Missing or invalid
  token input makes zero provider task requests.
- Make one-page adapters return an opaque cursor envelope. Use bounded
  complete traversal, or declare a paged public result with the catalog-bound
  optional cursor input and next-cursor output.
- Keep wire DTOs in infrastructure. Add publishable fixtures under `testdata`
  and bind their path, digest, provenance, and license in
  `.harness/schemas.json`.
- Map provider failures once into stable `fault.Error` values. Never expose a
  raw response body or cause.

## 6. Test at each boundary

Add the smallest set that proves the capability:

- domain tests for invariants and edge cases;
- application tests with fake ports for ordering and failure behavior;
- adapter contract tests for exact requests and bounded responses;
- CLI tests from argv through stdout, stderr, exit code, and captured effects;
- rejection tests proving invalid input causes zero external calls;
- catalog tests rejecting missing, extra, duplicate, optional, non-CLI, non-opaque, and reference-kind-mismatched mutation bindings;
- catalog tests rejecting optional act references and closed required-reference cycles;
- authentication/policy/cancellation tests proving zero downstream mutation;
- mutation outcome tests proving structured deadline/cancellation causes retain their typed classification, unstructured post-action errors are non-retryable, confirmed success is not overwritten, and reconciliation cannot point to a mutation;
- authentication-binding tests with simultaneous accounts/authorities,
  missing, stale, wrong-account, and cross-session IDs, typed-nil task ports,
  expiry races, refresh identity mismatch/failure, and zero unintended provider
  task requests;
- Chatwork PAT tests proving that `CWK_API_TOKEN` is the sole source, that no
  method selector or fallback exists, and that missing, malformed, canceled,
  or mismatched bindings make zero provider task requests;
- authentication secret-canary tests that include PAT values, authorization
  headers, provider bodies, stdout, stderr, structured faults, logs, snapshots,
  fixtures, and test diagnostics;
- pagination tests for empty, one, many, repeated-cursor, budget, and mid-page failure paths;
- catalog tests rejecting missing, extra, required, non-string, non-opaque, and
  reference-kind-mismatched public cursor bindings;
- hostile-output tests for ESC/newline, bidi and zero-width format characters, U+2028/U+2029, pre-existing backslashes, JSON-looking and prompt-like printable data, oversized content, and writer failure;
- tests proving structural escaping does not claim to filter semantic instructions and does not change an opaque reference;
- regression fixtures for stable TSV/JSON output and structured error output.
- presentation-independent semantic fixtures and exact answer keys for
  To/reply/quote relations, missing references, coverage, canonical action
  references, and hostile text;
- negative relationship tests proving that To, quotes, display names, time
  proximity, and indentation-looking text do not fabricate reply edges;
- canonical-reference round trips that reject presentation-derived shorthand
  unless a separate typed contract explicitly defines it;
- reviewed collection tests for schema/trust on empty results, provider order,
  one physical line per item, hostile quoted text, stable required positions,
  final optional suffixes, and the `files list` to `files show` round trip;
- a no-post-processing agent transcript, the historical candidate-C baseline,
  and the current headerless task-projection contract, including its fixed
  positional message schema and subtractive-field rules; any future replacement
  uses a presentation competition that pins candidates,
  agent/model versions, prompts, repetitions, answer scoring, token accounting,
  quality floor, latency, benchmark-defect reporting, and raw result retention;
- finite-filter tests proving exact canonical inputs, OR/AND semantics,
  invalid combinations before I/O, provider-call count, selection bounds,
  typed-time ranking and deterministic ties, start/count continuation without
  repeated ranks, source-order preservation,
  source/candidate/start/requested-count/items-per-page/next-start distinction,
  context allowed beyond requested primary count, context-hop limits,
  over-coverage source rejection,
  match/context distinction, and no inference from presentation or external
  text;
- retained baseline fixtures for the candidate-C first contract and active
  golden fixtures for the current task projection; future replacements receive
  active compatibility fixtures only after reviewed evidence and an explicit
  compatibility decision;
- command-attention tests proving a new configurable leaf stays off in a saved
  profile, appears in missing-state defaults, participates in active
  reference/recovery closure, and cannot leak through any human/agent help or
  routing projection while disabled. Prove disabled execution performs zero
  PAT/provider calls and re-enabling retains the leaf's original security
  policy. For this product also prove single-command routing, always-on
  `help`/`doctor`/`version`/`config`, exact legacy normalization, non-TTY
  rejection, fragmented arrow input, bounded hostile-text-safe frames, textual
  effect badges independent of color, validate-restore-save ordering, zero-save
  restoration failure, and read-only doctor fingerprint reconciliation.

Tests must use temporary directories, fixed clocks, fake credentials, and local
test servers. They must not require a developer account or live network.

## 7. Keep claims enforceable

Update the claim-to-enforcement table in `docs/04_harness.md` when a safety claim
changes. Update architecture and operating documentation when a boundary
changes. Do not rely on prose alone when a lint, type, test, or workflow can
enforce the rule.

Run:

```sh
./scripts/check.sh fast
./scripts/check.sh full
./scripts/check.sh security
```

Before public release, also confirm `./scripts/check.sh public` passes with
`profile: ready` in `.harness/project.json`.

## 8. Validate the agent journey

Replay the relevant scenario from `docs/09_agent_readiness_validation.md`.
Record how many invocations were needed to discover the task, where each input
came from, whether output passed unchanged into the next task, and whether each
failure selected a next command without prose interpretation. Extra command
guesses are thesis/product evidence, not an agent workaround to document.
Also record the external post-processing count, provider-call count and bounds,
semantic answer accuracy, canonical references, candidate worktree/commit,
agent/model versions, repetitions, and per-run byte, token, tool-step, and
latency measurements. A supported scenario with a nonzero external
post-processing count is incomplete, claims an outcome that is too broad, or
has an ineligible presentation candidate.

## 9. Feed implementation learning back into the thesis

Implementation is an iterative design probe. When code or tests reveal a new
constraint, do not leave the decision only in a local comment. Revisit the
thesis, refine it when the lesson is general, then propagate that decision into
architecture documentation, the command catalog, typed contracts, tests,
linters, and this skill. The repository should become less ambiguous after
each capability is added.
