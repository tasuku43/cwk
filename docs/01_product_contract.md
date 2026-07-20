# Product Contract

This document defines which user outcomes belong in Chatwork CLI (`cwk`) and the representation-independent behavior they must preserve. A concrete presentation is an explicit, replaceable product contract; future replacements require comparative evidence.

## Product statement

`cwk` is a task-oriented Chatwork CLI for developers, operators, and coding agents. It maps user outcomes to exact bounded commands and returns sufficient, trustworthy task context with low agent cognitive and token cost.

It is not an API explorer or a one-to-one endpoint wrapper. Its first complete implementation covers the fixed 2026-07-18 public REST-operation snapshot through reviewed user-task workflows, without making every consumer rebuild routine Chatwork semantics with external tools.

## Primary users

- A developer or operator in Japan delegating Chatwork work to a coding agent.
- An automation author relying on stable command, reference, failure, and semantic-output contracts.
- A human supervising what an agent will read or change.
- A contributor testing presentation ideas without changing product semantics.

## Language and locale contract

The product has one default human locale: Japanese. Human-oriented help,
interactive prompts, fault explanations, recovery reasons, and active public
guides are Japanese. The executable does not infer a locale from the operating
system and does not currently expose a language flag.

Automation remains locale-stable. Exact command paths and flags,
`CWK_API_TOKEN`, allowed-value tokens, effect and role values, fault kind/code,
exit status, JSON keys, schema versions, output field names, capability IDs,
reference kinds, and opaque references remain unchanged ASCII values. The
reviewed success projection also retains its fixed schema tokens and positions;
Japanese documentation explains those tokens instead of translating them.
Provider text is external data and is never translated by `cwk`.

## Supported-outcome promise

For every supported outcome, an agent can:

1. select the exact task from the machine-readable root outcome index and at
   most one scoped contract;
2. supply declared inputs without guessing endpoints, name matches, URLs, or hidden defaults;
3. identify task-relevant facts, bounds, missing context, uncertainty, and canonical references;
4. complete the outcome without routine `jq`, `grep`, custom joins/parsers, raw Chatwork-notation interpretation, source inspection, or exploratory API calls;
5. distinguish success, declared bounded/partial context, and failure;
6. recover through structured executable next actions.

Direct extraction of a declared field or canonical reference is allowed. Reconstructing product semantics from provider fields or multiple undocumented calls is not.

## Public runnable surface

The complete public catalog exposes the local `help`, `doctor`, `version`, and
interactive `config` utilities plus the task-oriented Chatwork workflows
defined below. Public opaque-reference and output contracts are proven by the
Chatwork workflows themselves; the product carries no parallel runnable
resource scaffold.

Human text discovery is hierarchical. Root help shows the directly runnable
single-word utilities and each canonical top-level task namespace once; it does
not repeat the namespace's leaf paths or summaries. `<namespace> --help` (or
`help <namespace>`) then lists only that namespace's exact commands, and an
exact trailing `--help` shows one command contract plus any reviewed structured
recipes whose exact steps contain that command. A recipe is omitted unless all
of its steps exist in the active view. Root and namespace help do not render
recipes. Namespace membership,
counts, section-relative ordering, and selectors derive from `cli.Catalog`;
namespace nodes are not implicit executable commands. Exact human help projects
each declared input's required/repeatable state, source, allowed values,
reference kind, and description from the same command contract. Schema-v4 agent
help uses compact root and namespace indexes containing exact command-level
outcomes and pointers. Only an exact-command selector returns the complete
invocation, output, failure, recovery, authentication, mutation, and workflow
contract.

Human recipes are not included in any schema-v4 agent help. For agents, the
exact leaf summary makes the discovery outcome selectable and exact-command
agent help supplies complete inputs and canonical-reference workflows.

### User-selected command view

The complete `DefaultCatalog` remains the supported-product, capability-ledger,
API-coverage, and release contract. At runtime, `cwk` may derive a smaller
ordered attention view from an exact-path allowlist of configurable catalog
leaves. Human root, namespace, exact, and trailing help; agent root/namespace
indexes and exact-command scopes; recovery actions; reference workflows; and command routing all consume
that same active view. A disabled path therefore appears as the ordinary
`unknown_command` and is rejected before PAT resolution or provider I/O.

`help`, `doctor`, `version`, and `config` are always-on catalog commands. Every
Chatwork task leaf is configurable; no local always-on operation is stored as a
current selectable path. Bare `config` opens the sole selector on interactive
stdin and stdout terminals. Up and Down move the cursor, Space toggles exactly
one Chatwork leaf, Enter validates and persists the complete exact-path
allowlist, and `q` exits with the prior profile unchanged. The selector treats
ASCII Space and U+3000 full-width space from a terminal input method as the
same toggle without changing Enter's save boundary. Redirected or
otherwise non-terminal input fails with a typed fault rather than selecting a
second grammar.

Every row carries its literal catalog effect badge: `read`, `create`, or
`write`. Cyan/read, yellow/create, and magenta/write may supplement the badge,
but color is not semantic and red is not used to suggest that every write is
destructive. The effect badge occupies a fixed-width field sized to `[create]`,
so every exact command path starts in the same column. The bounded renderer
preserves cursor, checkbox, exact path, and badge before truncating optional
summary text. If the current exact path or an
item row cannot fit, the selector shows resize guidance and accepts only a
non-saving exit; hidden or truncated command identity can never be toggled or
saved.

Before Enter, quitting, input closure, Ctrl-C, or context cancellation performs
zero saves and restores terminal state. Enter first rejects a view that strands
a visible required-reference consumer or points a visible recovery action at a
hidden command, then restores the terminal, then invokes one fixed-target
profile replacement. Restoration failure also performs zero saves, and a
confirmed save is not overwritten by late cancellation.

An absent preference is an explicit unconfigured first-run state. Its active
view contains only `help`, `doctor`, `version`, and `config`; no Chatwork task is
advertised or invocable. Human root help says that `config` is unset, that only
the control commands are currently shown, and that selecting only relevant
commands reduces agent token use and selection mistakes. The root agent index
contains only the same four active commands. A known configurable command,
namespace, trailing help, or scoped help request fails as non-retryable
`command_selection_required` with exact `config` recovery before PAT resolution
or provider I/O. A genuinely unknown path remains `unknown_command`.

`config` starts an absent profile with all current Chatwork commands selected;
Enter explicitly saves that selection, while quitting leaves the installation
unconfigured. Once a preference has been saved, its allowlist is authoritative:
commands added by a later release remain off until selected, and removed paths
remain visible only as stale configuration evidence. A deliberately saved empty
allowlist is configured and does not re-enter first-run guidance. Invalid state
never silently restores the full view or presents an always-on-only root help
page as a normal empty selection. When active-profile loading fails, `doctor`,
`version`, `config`, and scoped help for the always-on local commands remain
reachable; bare root help and Chatwork discovery return the typed load fault
because no truthful active Chatwork view exists. Malformed serialized content
may enter the deliberate `config` repair flow and still writes nothing before
Enter. Unsafe filesystem objects, modes, or inaccessible paths require local
repair instead of an edit loop.

Read-only `doctor`, not `help` or a second config leaf, reconciles an uncertain
profile replacement. Its `command-selection` diagnostic reports source, state,
enabled and disabled counts, and a deterministic `sha256:` fingerprint over
the catalog-ordered enabled exact paths. An unavailable canonical state emits
an unavailable fingerprint rather than fabricating one. The scoped agent error
contract publishes the exact uncertain-message grammar containing expected
`source=saved` and the candidate fingerprint; JSON and text errors follow that
same grammar before routing to `doctor`. Profiles created by
the retired selector may contain `doctor` or `version`; only those two legacy
entries are ignored while deriving the view and removed on the next successful
Enter save. Other always-on entries remain invalid.

A confirmed save is a human terminal result rather than a reconciliation
record. It says in natural Japanese how many Chatwork commands are visible,
hidden, and changed, and mentions old-setting cleanup only when it occurred.
It does not repeat the fingerprint or expose internal key/value labels. The
words visible and hidden describe only the local attention view; they do not
grant or revoke Chatwork authority.

The preference is stored separately from credentials and the retired OAuth
configuration: `${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json`
on macOS and Linux, and `%AppData%\\cwk\\command-selection.json` on Windows.
On macOS and Linux, an existing configuration-home symbolic link is resolved
once to its absolute directory target before `cwk` joins or opens its owned
path. The `cwk` directory and `command-selection.json` remain non-symbolic
owned targets with their strict shape and Unix-mode contracts.
The adapter uses a restricted same-directory temporary file, rename and
directory sync on Unix, and portable replace-existing behavior on Windows;
the Windows API does not supply a cross-platform atomicity or durability
promise.

It is a cognitive-surface preference, not authorization, access control,
sandboxing, or a promise that hidden commands are unavailable to the same local
principal. Editing or deleting the file can restore commands. PAT validation,
provider permissions, canonical-reference binding, mutation effects, and the
existing access-change/destructive confirmations apply unchanged after any
command is enabled.

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

It does not publish a global version/task preamble, a standalone provider-oriented coverage record, raw Chatwork notation as semantic structure, undeclared provider/wire fields, duplicated coverage prose, empty optional shells, or helpful non-contract defaults. Collection bounds and completeness sit on the collection record; a message window uses the task vocabulary `recent` or `changes` and names the provider ceiling `source-limit`. `messages list` emits its room, trust classification, and the fixed schema `#sequence message-ref actor sent [reply] [to] [quote] [relation-state] "body"` once, then an actor dictionary and one physical record per selected typed message in original provider order. Without a sender, period, or index selection, every provider-returned message is selected. An active selection additionally emits one record before the trust declaration; it preserves the source-window count, exact sender and effective period filters when present, candidate count, one-based start index, requested count, actual items per page, optional next start index, context policy, and primary anchors without adding state to every message. The sequence, canonical message reference, actor, send time, and quoted body are positional; optional typed edges remain labeled. A record with an unproved relation set emits `relation-state=unknown` and omits relationship facts instead of claiming `relations=none`. `#N` is the one-based original provider sequence and may contain gaps after selection; one reply uses `reply=#N` and multiple replies preserve provider notation order as `reply=[#N,#M]`; neither form is command identity. To and reply remain separate, unresolved targets retain an available canonical reference, and depth/thread/root/children/resolved-default records are absent. A complete multiple-reply set is typed rather than marked unknown. Bounded reply closure is projected separately: `relation-resolution` states the fetch limit and attempts, `relation-context` identifies source or fetched provenance without a source sequence, and `relation-gap` distinguishes not-found, restricted, and budget-exhausted targets. A declared raw message body remains visible as untrusted external text; presentation does not reinterpret it as a reply, recipient, quote, instruction, or other semantic fact.

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

`members find --room <room-ref> --query <text>` is the supported bridge from
an external display name to canonical account candidates inside one exact
room. It performs exact, case-sensitive substring matching over the complete
typed result of the existing room-members read, preserves provider order, and
reports the query, source-member count, candidate count, completeness, display
name, role, and reusable `account_ref`. It performs no case, whitespace, or
Unicode normalization, ranking, fuzzy search, or automatic selection. Zero,
one, and multiple candidates are equally valid discover outcomes. The query is
application-owned and never becomes a Chatwork request parameter; the selected
canonical reference is passed unchanged to `messages list --sender`.

Omitting `messages list --window` selects the latest bounded `recent` window,
which is the normal conversation-understanding outcome. Explicit
`--window recent` is equivalent. Provider differential retrieval remains
available as explicit `--window changes`; it is not the implicit behavior of
the shortest command. Both modes remain incomplete bounded room context with
`source-limit=100`, not complete room history.

The product owns deterministic filtering and joining needed by a supported
outcome. `messages list` establishes the first finite typed selection contract:
up to 100 repeatable `--sender <account-ref>` inputs match any listed exact
sender; optional `--start-index <index>` and `--count <count>` each accept 1
through 100; `--start-index` defaults to 1 when `--count` is supplied; and
`--context none|replies` defaults to `none`. Inclusive
`--since <RFC3339>` and exclusive `--until <RFC3339>` accept whole-second
timestamps with explicit offsets and may be supplied independently or together.
`--on <YYYY-MM-DD|today|yesterday>` is mutually exclusive with those bounds,
uses the fixed `Asia/Tokyo` calendar, and resolves a command-injected clock once
to one concrete half-open day. Optional `--resolve-relations <count>` accepts
0 through 100, defaults to five, and uses zero as the explicit opt-out. Sender
OR and period membership form the primary
candidate predicate.
Typed `send_time` ranks candidates newest first, with later provider position
winning equal-time ties. The one-based start index chooses the first rank and
count caps how many primary anchors follow, so `--start-index 11 --count 20`
means ranks 11 through 30 rather than ranks 11 through 20. `replies` runs last and adds
only direct parents and children connected to the selected anchors by typed
reply edges inside the one provider-returned window. Context may increase the
displayed count beyond the requested count and may add a direct neighbor outside
the requested primary period; it does not traverse transitively, expand To or
quote relations, or parse raw body notation.

After selection, every unique explicit same-room reply target referenced by the
displayed result is considered in breadth-first first-reference/provider order. A target in
the original source is attached as supplemental context without consuming a
fetch slot. An absent parent consumes at most one exact-message request; the
default public invocation therefore performs at most one list call plus five
exact reads, and an explicit value N permits at most one list call plus N exact
reads. Every explicit same-room reply of a supplemental message joins the same
queue. Duplicate and cyclic targets are visited once. A fetched parent is context rather than a
source message, candidate, ranked item, anchor, or proof of older-history
coverage. Exact not-found and restricted outcomes remain per-target gaps;
rate-limit, unavailable, cancellation, malformed, and other permission faults
fail the command without partial success.

Selection retains the provider window's original one-based sequence and
physical order, so displayed `#N` values may have gaps even though timestamps
chose the anchor set. A single selection record declares the source-window
count, exact sender set when present, candidate count, applied start index,
effective concrete period bounds/day when present, requested count, actual
`items-per-page`, optional `next-start-index`, context, and primary anchor
sequences; records not listed as anchors are included reply context. The
provider's maximum-100 response remains a distinct
`source-limit`, and a response above that declared coverage fails instead of
being hidden by local selection. Repeating two senders is a truthful
two-sender-focused slice, not a claim that every displayed message is authored
by or directed exclusively between them. Exact canonical account and message
references remain the only command identities.

The Chatwork endpoint documents only its `force` window query. Index selection
is application-owned and reduces rendered/token cost
when it actually removes primary records; it never reduces provider response
bytes. Its vocabulary and one-based/count semantics derive from SCIM index
pagination, but it supplies no provider cursor or offset and is not a SCIM or
provider pagination implementation. Separate invocations are stateless; source
changes can move ranks between them. Invalid, duplicate, non-integer, zero,
negative, above-100, ambiguous date, offset-free timestamp, fractional-second,
empty/reversed interval, or conflicting day/bound values fail before
authentication or provider I/O. Period selection changes rendered/model input
only; it does not reduce provider response bytes or retrieve older history.

For a nonempty `recent` source with no access limitation and valid typed send
times, the result publishes the minimum-time source message as the oldest
reachable boundary. A requested period is typed as within that boundary,
partly before it, wholly before it, or unknown. `changes`, empty, limited, and
unprovable sources use unknown. A wholly older period is not reported as an
ordinary empty day, and the boundary never claims complete room history or
provider pagination.

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

`rooms create` requires `--account <account-ref>` as a credential-bound
creation scope, not an owner. Chatwork's official room-create request has no
owner field. After input and exact access-change confirmation validate, the
adapter uses the same private PAT binding for `GET /me`; only an exact
`account_id` match permits `POST /rooms`. The transport rechecks that verified
session account against the request immediately before construction, so a
generic or mismatched binding cannot bypass the CLI gate. A mismatch,
cancellation, or failed
identity check produces no room-create request and never publishes the actual
identity or token. `--owner` is removed rather than retained as a misleading
compatibility alias. The account reference comes from `account show`, but that
earlier result alone is not current credential proof. The official 1--255
character room-name bound and finite icon preset set also fail before
authentication.

Primary sources: Chatwork's
[room-create reference](https://developer.chatwork.com/reference/post-rooms)
and [`GET /me` reference](https://developer.chatwork.com/reference/get-me).

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

The provider documents a 100-item maximum for `GET /my/tasks`, room message, room task, room file, and incoming-request lists. Those results preserve that 100-item provider bound rather than claiming the 10,000-item aggregate ceiling. A user-selected message limit cannot raise or replace that source bound, and an oversized provider result fails before selection. The active contracts fix cancellation, message-window semantics, provider rate-limit behavior, mutation policy, PAT failure-before-I/O behavior, and publishable schema fixtures before live I/O is enabled. Multiple accounts remain deferred.

For message retrieval, the 100-message source bound and provider access
limitation are separate facts. A list `204` without
`chatwork-message-limitation` is a true zero for that invocation; it does not
prove that the room has no history. `200` plus the sole official value `true`
means some messages in the requested window were restricted, and `204` plus
`true` means every would-be result in that window was restricted. Exact
message `404` plus `true` is `chatwork_message_restricted`; `404` without that
header is `chatwork_not_found`. The limitation summary remains private
provider prose. A false, duplicate, contradictory, or status-incompatible
limitation signal fails as `chatwork_message_limitation_invalid` instead of
being presented as complete or empty.

Message notation is untrusted enrichment over a separately preserved body.
Only the official complete To, reply, and quote forms and the reviewed complete
code-delimited region contribute parser control. If a recognized form is
malformed, unclosed, contradictory, or ambiguous, that one message retains its
terminal-safe body but publishes `relation_state: unknown` and no partial
typed relations. Other records in the list remain available. Unknown is not
relation absence, and quote author/time never reconstruct a message identity.

Primary sources: Chatwork's
[message-list reference](https://developer.chatwork.com/reference/get-rooms-room_id-messages),
[exact-message reference](https://developer.chatwork.com/reference/get-rooms-room_id-messages-message_id),
[limitation change notice](https://developer.chatwork.com/changelog/2022-09-06-notice),
and [message-notation guide](https://developer.chatwork.com/docs/message-notation).

Chatwork's official general rate limit is 300 requests per five minutes. A
`429` timing is known only when one `x-ratelimit-reset` header contains
strict decimal Unix seconds in the future and no more than five minutes from
the response and local clocks. A valid HTTP `Date` is the duration baseline;
the local clock is used only when `Date` is absent or invalid. `Retry-After`,
missing, duplicate, malformed, expired, or implausibly distant values do not
establish timing. The two
documented room-posting operations, message creation and task creation, have a
combined 10-request/10-second room limit. Only their exact documented bounded
JSON error, `Rate limit for message posting per room exceeded.`, establishes a
10-second wait; its provider body remains private. Text errors render absent
timing as `retry_after: unknown`, and JSON uses `null`.

A rate-limited read is `retryable: true` and may point back to the exact read
task. A rate-limited mutation is `retryable: false`, routes to scoped help, and
is never automatically retried. A known wait on that mutation is advisory
rate-limit evidence, not permission to repeat a change. The one-attempt
transport policy remains unchanged.

## Mutation confirmation policy

An exact invocation with validated canonical references and complete typed intent is sufficient for ordinary creates and updates, including message/task creation, room metadata changes, read-state changes, message edits, and task status changes. This is explicit command intent, not a general authorization grant.

Mutations that change membership, contact access, or link exposure additionally require the exact `--confirm=access-change` value. In the fixed operation snapshot these are room creation, room-member replacement, invite-link creation/update, and incoming-contact-request acceptance. Destructive operations additionally require the exact `--confirm=destructive` value: room leave/delete, message deletion, invite-link deletion, and incoming-contact-request rejection. Confirmation is invocation-local, is never inferred from a TTY or agent identity, and is not reused.

`invite-link update` is a complete replacement over every public provider
input, not a partial patch. It requires exactly one of a validated `--code`
(1--50 ASCII letters, digits, `_`, or `-`) and `--regenerate-code`, plus an
explicit `--approval` and nonempty `--description`. Only the regeneration flag
permits omission of `code`; that omission intentionally invokes the documented
random generation behavior. Empty and partial replacements fail before
authentication. `description` is also available on create. Because the
official reference does not define description omission or empty-string clear
semantics, cwk neither merges a prior value nor exposes a speculative clear
operation. The provider-confirmed result renders the resulting URL, approval,
and nonempty description.

Primary source: Chatwork's
[invite-link update reference](https://developer.chatwork.com/reference/put-rooms-room_id-link).

Every provider operation has one transport attempt. An uncertain mutation result is non-retryable and names an exact read-only catalog task for reconciliation before another mutation; it never recommends repeating the write.

## Compatibility boundary

Before `1.0.0`, contracts may evolve intentionally with tests and migration notes. Once stabilized, compatibility includes command paths, typed inputs, roles, effects, reference kinds, semantic field meanings, bounds/completeness, failures, authentication configuration, and release artifacts.

Selecting Japanese as the default human language is an intentional pre-1.0
text-contract change. Human prose is not retained in English as an alias.
Machine classification and automation remain compatible through the unchanged
identifiers listed in the language contract.

Candidate C's versioned grammar, schemas, defaults, and ordering were the compatibility promises of the first complete implementation. The P-derived `cwk-task-projection/1` deliberately broke that contract; the headerless projection made a second pre-1.0 break by removing its repeated schema/task preamble and standalone coverage record. The flat chronological `messages list` adjacency contract is a third explicit pre-1.0 refinement and superseded an unimplemented indented-tree proposal. Applying fixed positional records to the seven reviewed homogeneous collections is a fourth explicit refinement. Clients must not expect historical headers, reference dictionaries, aliases, field ordering, or grammar. Current compatibility is identified out of band by the release and enforced by catalog fields, documentation, all-route tests, and goldens. Semantic field meanings, exact canonical references, bounds/completeness, failures, and trust classifications remain governed independently of a text migration. A future replacement changes the current promises only through reviewed evidence and an explicit compatibility decision. Experimental worktree output carries no compatibility promise.

Hierarchical human help is another explicit pre-1.0 text-contract change: the
former flat root leaf list moved to catalog-derived namespace views, natural
namespace `--help` became valid, namespace command names became relative, and
exact help gained catalog-derived input facts and navigation links. The later
schema-v4 agent-help decision is an intentional pre-1.0 machine-contract break:
root and namespace selectors are compact indexes that point only to exact
command paths, while complete contracts and touching workflows are returned
only for one exact command. It replaces schema-v3 namespace aggregation, which
could emit hundreds of kilobytes of unrelated complete contracts. Non-help task
contracts, exact command paths, and command execution contracts do not change.
The scoped `help` task contract
changed only by correcting its stale invalid-selector recovery wording to name
namespaces as valid selectors.

Bounded message selection was introduced as an additive pre-1.0 command-input
change. The later index-selection decision is an intentional pre-1.0 breaking
change: message `--limit` is replaced by `--start-index` and `--count` so an
argument cannot be mistaken for an inclusive end rank. The independent room
task deadline `--limit` is unchanged. Catalog `source_limit`/text
`source-limit=...` remains the fixed 100-message provider retrieval ceiling;
message record positions and canonical references do not change.

The later recent-window-default decision is an intentional pre-1.0 behavioral
change for `messages list` invocations that omit `--window`. Omission now sends
the documented `force=1` query and reports `window=recent`; callers that require
provider differential semantics must pass exact `--window changes`. Both
explicit values keep their prior meaning, and no persisted migration is needed.

The bounded relation-closure decision is an intentional pre-1.0 behavioral and
text-contract change for `messages list`. Omission now permits up to five
recursive exact same-room reply-parent reads and publishes resolution evidence;
`--resolve-relations 0` restores zero additional calls. The same change adds an
oldest-reachable boundary and period-reachability classification where those
facts can be proved. It does not change the list source ceiling, create provider
pagination, follow cross-room references, or persist configuration.

Persistent command selection is an additive pre-1.0 local workflow. It does
not remove capabilities from `DefaultCatalog`; it derives the help and routing
view for one installation from a saved exact-path allowlist. The later
single-selector refinement deliberately removed public `config show` and
`config edit`, made `help`/`doctor`/`version`/`config` always on, adopted one
terminal Up/Down/Space/Enter/`q` interaction, and moved read-only uncertain-save
reconciliation to `doctor`. The profile schema and location did not change;
legacy `doctor`/`version` entries are normalized on the next save.

The subsequent first-run refinement is an intentional pre-1.0 behavioral
change: missing state now exposes only the four control commands and requires an
explicit Enter-confirmed `config` save before any Chatwork task or scoped task
help is available. It adds `command_selection_required`, an explanatory root
help notice, and `state=unconfigured source=missing` diagnostics. Existing
saved profiles and deliberately saved empty selections are unchanged.

The final pre-`v0.1.0` selector presentation deliberately replaced the
machine-shaped confirmed-save line with Concept A's natural-Japanese result
suffix. Scoped agent output fields changed from
`enabled`/`disabled`/`stale_removed`/`legacy_removed`/`fingerprint` to
`visible`/`hidden`/`cleaned`, while retaining `status` and `changed`. Cleanup
now combines stale and legacy entries and appears only when nonzero. The
fingerprint remains exclusively on the uncertain-save and read-only `doctor`
reconciliation paths; no released version used the superseded contract.

## Explicit non-goals

- Tracking future Chatwork additions automatically, mechanically mirroring endpoints, or exposing raw transport passthrough.
- Treating candidate C or the current task projection as a thesis, or allowing presentation shorthand to become action identity.
- Silent fuzzy matching, truncation, or relationship inference.
- Default lossy/model-generated summaries.
- Claiming structural escaping makes external text semantically trustworthy.
- OAuth grants and lifecycle commands; token persistence; multiple accounts or credential profiles; administration/private APIs; webhooks; GUI work; and release publication in the first complete implementation. Presentation experiments and token optimization remain separate from API-capability completeness.

## Completion evidence for a Chatwork capability

A capability is complete only when its outcome, non-goals, command discovery, semantic model, exact references, bounds, failure behavior, authentication, external-call policy, hostile-data tests, and agent transcript are reviewed. Candidate C remains the first-stable baseline evidence. Presentation-dependent completion under the current default requires the headerless task-projection contract, its subtractive-field and golden evidence, and the recorded breaking compatibility decisions; no benchmark-win claim is required or permitted for the inconclusive Competition 1. Required repository gates must pass.
