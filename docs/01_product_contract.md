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

1. select the exact task from root or namespace text help and obtain an
   executable usage with at most one further help lookup;
2. supply declared action inputs without guessing endpoints, silently
   resolving names, converting URLs, or relying on hidden defaults;
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

Public discovery has one text-help route. Root help shows directly runnable
single-word utilities and each canonical top-level task namespace once.
`<namespace> --help` (or `help <namespace>`) lists only that namespace's exact
commands and includes each leaf's complete usage line plus its required flags
and required one-of input groups.
For a namespace with active leaves, root help derives up to three distinct
outcomes from those leaves' catalog Summary values and supplies each shown
leaf's exact --help route. Identical summaries appear once; when more distinct
outcomes remain, root help reports their count and points to exact namespace
help. A disabled-only namespace remains a name-only entry, and disabled leaves
never contribute summaries or routes. An unconfigured view remains limited to
the four local control commands. The link-discovery guide appears only when the
active catalog contains `messages resolve-link` and points to its exact help.
A separate `messages show --help` route appears only when that leaf is active.
In a saved resolver-only view, root help asks a person to enable show through
config without presenting it as executable. Marked static fallback uses the full
catalog but explicitly does not assert active selection. No route includes
unchecked arguments. Root summaries must be
nonempty terminal-safe one-line catalog text of at most 192 encoded bytes; the
default catalog's Japanese prose is checked by the localization contract.
Tests report normal and marked static-fallback root sizes in encoded bytes; no
root-wide hard limit is added. The existing exact-command help limit remains
2,048 bytes.
An exact trailing `--help` contains one usage line, each flag's name, type,
required, required-one-of, or optional state and the complete one-line
description declared in the catalog, plus combination constraints and no more
than two examples. Its encoded output is at most 2,048 bytes. If needed,
optional examples are removed from the end until it fits. The renderer never
shortens descriptions or constraints; catalog validation rejects a command
whose complete no-example projection exceeds the limit.
Namespace membership, counts, section-relative ordering, selectors, usages,
required flags, and examples derive from `cli.Catalog`; namespace nodes are not
implicit executable commands.

`help ... --format agent` is retired and fails with an explicit instruction to
use `--help`; it is never silently ignored. Data commands may still declare
`--format json`, but help has no machine-contract format. Complete
`AgentContract` metadata remains the internal validation source for routing,
effects, output, authentication, failures, recovery, mutation binding, and
reference workflows. Those details are not serialized speculatively into
public help. Runtime failures carry the structured `kind`, `code`,
`next_action`, and applicable usage needed for recovery.

### User-selected command scope

The complete `DefaultCatalog` remains the supported-product, capability-ledger,
API-coverage, and release contract. At runtime, `cwk` derives an ordered
operation scope from an exact-path allowlist of configurable catalog leaves.
The active view is the set of configurable tasks that this `cwk` process will
dispatch. A saved-profile request for an exact disabled canonical path, its
catalog-declared alias, or its exact scoped/trailing help returns the
non-retryable `command_disabled` fault before handler dispatch, PAT resolution,
or provider I/O. The error identifies the canonical path and asks the user to
enable it through `config`. A single exact catalog root namespace is a
catalog-derived help selector equivalent to `cwk help <namespace>` or
`cwk <namespace> --help`; saved profiles show active invocation details and
disabled canonical names only. Other unknown or incomplete paths remain
`unknown_command`.

Root and namespace help may separately name disabled canonical paths. The
disabled section contains names only; it carries no usage, summary, examples,
or workflow. Active help details, recovery projections, reference workflows,
examples, and typo suggestions use only the active view. Exact help for a
disabled leaf is rejected as `command_disabled`, just like execution.

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
advertised or dispatchable through this `cwk` process. Root help says that
`config` is unset, that only control commands can currently be run, and that
selecting only relevant tasks reduces agent token use and selection mistakes. A
known configurable command, exact namespace, trailing help, or scoped help
request fails as non-retryable `command_selection_required` with exact `config`
recovery before PAT resolution or provider I/O. A genuinely unknown or
incomplete path remains `unknown_command`.

`config` starts an absent profile with all current Chatwork commands selected;
Enter explicitly saves that selection, while quitting leaves the installation
unconfigured. Once a preference has been saved, its allowlist is authoritative:
commands added by a later release remain off until selected, and removed paths
remain visible only as stale configuration evidence. A deliberately saved empty
allowlist is configured and does not re-enter first-run guidance. Invalid state
never silently enables the full execution view or presents an always-on-only
page as a normal empty selection. When active-profile loading fails, static
root, namespace, and exact-command help is rendered from the immutable complete
catalog, with zero authentication or provider I/O; it documents available
syntax but does not claim that a configurable command is enabled. Root and
namespace fallback output carries one static-catalog notice and directs the
caller to `doctor`; a bare exact namespace follows this same fallback. Exact
command help retains its 2,048-byte invocation-only contract. `doctor`,
`version`, and `config` remain executable, while every non-help Chatwork
invocation returns the typed load fault. Malformed serialized content
may enter the deliberate `config` repair flow and still writes nothing before
Enter. Unsafe filesystem objects, modes, or inaccessible paths require local
repair instead of an edit loop.

Read-only `doctor`, not `help` or a second config leaf, reconciles an uncertain
profile replacement. Its `command-selection` diagnostic reports source, state,
enabled and disabled counts, and a deterministic `sha256:` fingerprint over
the catalog-ordered enabled exact paths. An unavailable canonical state emits
an unavailable fingerprint rather than fabricating one. The catalog's internal
failure contract validates the exact uncertain-message grammar containing
expected `source=saved` and the candidate fingerprint; runtime JSON and text
errors follow that same grammar before routing to `doctor`. Profiles created by
the retired selector may contain `doctor` or `version`; only those two legacy
entries are ignored while deriving the view and removed on the next successful
Enter save. Other always-on entries remain invalid.

A confirmed save is a terminal result rather than a reconciliation record. It
says in natural Japanese how many Chatwork commands are enabled for dispatch,
disabled for dispatch, and changed, and mentions old-setting cleanup only when
it occurred. It does not repeat the fingerprint or expose internal key/value
labels. These counts describe the operation scope enforced by this `cwk`
process. They do not establish human identity or secure the OS account: the
same user can edit or remove the preference, automate the terminal, change the
configuration base, install another executable, or use a PAT outside `cwk`.

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

This operation-scope boundary applies only to dispatch through the current
`cwk` process. It is not adversarial OS security, does not establish human
identity, and does not prevent the same local user from editing or deleting the
file, automating terminal input, changing the configuration base, using another
executable, or calling the provider directly with a PAT. PAT validation,
provider permissions, canonical-reference binding, mutation effects, and the
existing access-change/destructive confirmations apply independently after any
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

It does not publish a global version/task preamble, a standalone provider-oriented coverage record, raw Chatwork notation as semantic structure, undeclared provider/wire fields, duplicated coverage prose, empty optional shells, or helpful non-contract defaults. Collection bounds and completeness sit on the collection record; a message window uses the task vocabulary `recent` or `changes` and names the provider ceiling `source-limit`. `messages list` emits its room, trust classification, and the fixed schema `#sequence message-ref actor sent [reply] [to] [quote] [quote-candidate] [relation-state] "body"` once, then an actor dictionary and one physical record per selected typed message in original provider order. Its counts separately expose `unresolved-replies`, `unresolved-quotes`, and `unknown-relation-sets`; a quote candidate does not reduce unresolved quotes. Without a sender, body-term, period, or index selection, every provider-returned message is selected. An active selection additionally emits one record before the trust declaration; it preserves the source-window count, exact sender and body terms, effective period filters when present, candidate count when body-term, period, or index/count selection is active, one-based start index, requested count, actual items per page, optional next start index, context policy, and primary anchors without adding state to every message. The sequence, canonical message reference, actor, send time, and quoted body are positional; optional typed edges remain labeled. A record with an unproved relation set emits `relation-state=unknown` and omits relationship facts instead of claiming `relations=none`. `#N` is the one-based original provider sequence and may contain gaps after selection; one reply uses `reply=#N` and multiple replies preserve provider notation order as `reply=[#N,#M]`; neither form is command identity. To and reply remain separate, unresolved targets retain an available canonical reference, and depth/thread/root/children/resolved-default records are absent. A complete multiple-reply set is typed rather than marked unknown. A quote candidate appears only when an original bounded source record has the same room, exact author account reference, and exact whole Unix send timestamp as the quote metadata. It carries a canonical `message-ref`, with the source sequence only while that message remains visible, `scope=source-window`, and `identity=unproven`; it never resolves the quote, proves global uniqueness, or creates a reply edge. A partial source may show observed candidates, but zero matches under a partial or unavailable source remain unknown and existing coverage/access limits remain visible. Bounded reply closure is projected separately: `relation-resolution` states the fetch limit and attempts, `relation-context` identifies source or fetched provenance without a source sequence, and `relation-gap` distinguishes not-found, restricted, and budget-exhausted targets. A declared raw message body remains visible as untrusted external text; presentation does not reinterpret it as a reply, recipient, quote, instruction, or other semantic fact.

When the caller opts into a message byte budget, one separate delivery record
uses exactly `max-bytes`, `output-complete`, `output-items`, the always-present
`output-total-items`, and, only when truncated, `output-next-start-index` plus
conditional `output-next-count` for an original count-bounded slice.
These fields describe presentation delivery and do not overload provider or
selection completeness.

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

`messages resources --room <room-id> --message <message-ref> --kind files|tasks|all`
is a bounded exact-reference action over the existing room
file and room-task collections. It returns every row whose typed `Message`
reference and room reference both equal the supplied canonical references,
preserving source order and duplicate rows. `all` reads files before room
tasks. Each selected source reports its pre-filter count, the fixed limit 100,
`complete=false`, `absence=unknown`, the number of matches, and every matching
canonical resource reference. `absence=unknown` means that zero matches in a
bounded source do not prove no link exists; the task makes no claim that the
message itself exists. The displayed file/task reference and exact room
reference pass unchanged to `files show` or `room-tasks show`. Message body
notation, names, time, and prose do not establish these links.

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

`rooms list --query <text>` is the supported bridge from an external room name
fragment to canonical room candidates. It performs Unicode simple-case-folded
substring matching over the complete typed result of one existing rooms read,
preserves provider order, and reports every match with its reusable canonical
`room_ref` plus query/source/candidate/completeness metadata. It performs no
Unicode normalization, whitespace normalization, trimming, ranking,
multi-rune case expansion, provider query, or automatic selection. Zero, one,
and multiple candidates are equally valid discover outcomes. An omitted query
retains the complete room-list behavior. This removes the supported workflow's
dependency on `grep` without allowing an action to accept a display name.

`members find --room <room-id> --query <text>` is the supported bridge from
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
sender; up to 100 repeatable `--contains <text>` inputs are nonempty exact
case-sensitive literal substrings of typed `Message.Body` with a maximum of
1,024 UTF-8 bytes each. Contains terms match by OR; sender and period
predicates are combined with them by AND. Terms are not trimmed, normalized,
interpreted as Chatwork notation, or sent as provider search. Empty, invalid
UTF-8, overlong, or excessive terms fail before authentication and I/O;
duplicates and their supplied order are preserved. Optional
`--start-index <index>` and `--count <count>` each accept 1 through 100;
`--start-index` defaults to 1 when `--count` is supplied; and `--context
none|replies` defaults to `replies` whenever a sender, contains, period, or
index selection is active; explicit `none` is the lower-output opt-out. Context
without one of those primary selections is invalid. Inclusive
`--since <RFC3339>` and exclusive `--until <RFC3339>` accept whole-second
timestamps with explicit offsets and may be supplied independently or together.
`--on <YYYY-MM-DD|today|yesterday>` is mutually exclusive with those bounds,
uses the fixed `Asia/Tokyo` calendar, and resolves a command-injected clock once
to one concrete half-open day. Optional `--resolve-relations <count>` accepts
0 through 100, defaults to five, and uses zero as the explicit opt-out. Sender,
contains, and period membership form the primary candidate predicate using
the OR/AND rules above.
For compatibility, `messages list --limit <count>` is a catalog-declared
deprecated alias of `--count` for at least two minor releases. It has identical
validation and selection semantics, succeeds, and emits
`warning: --limit は --count に改名されました` on stderr without changing
stdout.
Unknown flags are never corrected or executed automatically; the invalid-input
fault includes a deterministic did-you-mean suggestion drawn from that
command's declared flags and aliases.
Task-specific input guidance is also bounded by the active catalog and never
changes the invocation. `messages list --query` explains that `--contains`
locally matches the already retrieved, at-most-100-item range (whose default
window is `recent`) and points to `cwk help messages list`. Unsupported
`messages list --around` and `--message` point to `cwk help messages show` only
when that exact path is active; otherwise they point only to the list's own
help. `members list --query` points to `cwk help members find` only while that
path is active, and otherwise points to its own help. The exact unknown path
`messages search` gives the bounded `--contains` guidance only while
`messages list` is active; otherwise it points to generic root help. When
exactly one of the active `room-tasks` or `personal-tasks` namespaces exists,
`help tasks` points to that exact namespace help; when both exist it points to
root help, and when neither exists it gives only generic root-help guidance.
Selection-state handling retains precedence: an unconfigured known task
namespace returns `command_selection_required`, and unsafe static-help
fallback uses generic guidance instead of deriving candidates from the full
catalog. Saved disabled paths never supply hints, and guidance does not echo
an argument value. These human-readable hints do not replace the fault's
catalog-normalized exact recovery action.

Chatwork task commands that do not declare `--format` reject it explicitly:
successful task output is text, and JSON error formatting uses the global
`--error-format json` option before the command. The independent
`doctor --format json` output remains supported.
An unknown leading root option fails before command-selection storage,
authentication, or task dispatch. Its deterministic suggestion is limited to
the fixed global surface `--error-format`, `--help`, and `--version`; the typo
`cwk help --hlep` explicitly points to `cwk --help`. Neither path rewrites or
executes the suggestion.
Finite catalog-declared command compatibility paths accept `messages get` as
`messages show` and `rooms members find|list|replace` as the corresponding
top-level `members` task. They select no new handler and never bypass the active
command view; aliases are normalized only after that view is known, so a hidden
canonical path is not disclosed. Help selectors accept canonical or declared
alias paths with either spaces or dots between path segments. `--help` or `-h`
anywhere in an exact-command or `help <selector>` invocation returns that
selected static help without validating the other invocation flags. Root, namespace,
and exact static help remain available when saved command-selection storage is
invalid, unsafe, or temporarily unavailable; ordinary Chatwork tasks still
fail closed on the same state.
An unknown command ranks canonical paths in the already derived active catalog
by the existing deterministic distance rule. It may suggest the single nearest
candidate only when that candidate declares `read` effect. If the nearest
candidate is `create`, `write`, or `unknown`, no suggestion is shown; the
router does not promote a farther read candidate. Deprecated spellings and
disabled/full-catalog-only paths are not candidates, and no suggested handler
is invoked.

`messages show` remains bound to one exact room/message anchor. Omitted
`--context` and exact `--context none` make only that exact read.
`--context replies` adds exactly one declared latest-window list read for the
same room and authentication binding. From its at-most-100 provider-order
records, only messages with a typed explicit same-room reply edge whose target
is the anchor are emitted as `reply-child` records. Their canonical references
and original source sequences are preserved. A preceding `reply-context`
record names the anchor, child count, visible source count, `window=recent`,
`source-limit`, completeness, and `access-limitation=none|partial|all`.
Consequently zero children is only a bounded observation, not proof of no room
replies. To, quote, raw body notation, display names, and time proximity do not
fabricate children. `--resolve-relations 0` disables exact parent-chain reads
but not this fixed child-source read; a positive relation budget remains
separate and follows outgoing typed parent edges. The anchor record is always
first, followed by reply-context/children and then parent-resolution evidence.
`--around N` (0..100) also enables one bounded recent-list read after the exact
anchor read; when `--context replies` is present, both options share exactly
that one read. If the exact anchor occurs in its provider-order source, up to N
immediately preceding and following messages are included with their original
source sequences. A child that also occupies an around position is emitted
once, with its typed child edge retained. Adjacency alone never creates a
reply edge. `around-context` reports anchor state (`in-window`,
`outside-window`, or `unknown`), requested and actual counts per side, side
positions, source count/limit/completeness, and access limitation. An absent
anchor in an unrestricted source is outside the window; restricted access
makes its position unknown. Neither case emits inferred neighbors. Explicit
`--context none` is compatible with `--around`; omitting `--context` keeps its
existing behavior. `--around 0` still requests and reports the source.
Optional `messages list --max-bytes <n>` accepts an encoded stdout budget from
1 through 16,777,216 bytes (16 MiB). It is a CLI/presentation-only input: it is
not part of the domain Chatwork request, never becomes a provider parameter,
and neither narrows nor repeats the existing list request. The complete typed
selection and bounded relation resolution are produced before delivery
planning, so enabling the flag adds no provider call and suppresses none of the
calls the same invocation would otherwise require. Duplicate, non-integer,
zero, negative, and above-16-MiB values fail before authentication or provider
I/O.

Optional `messages list --max-body-bytes <n>` independently caps each message
body that is actually rendered. It accepts strict ASCII decimal values from 0
through 65,535. The limit is measured against the original raw UTF-8 body, and
the visible body is the longest prefix at or below that byte count that ends
on a UTF-8 rune boundary. The cap runs after contains, anchor/context
selection, quote-candidate matching, and relation resolution; it never changes
their typed source or result. It applies consistently to selected primary
messages, selected in-window reply context, and emitted `relation-context`
records. It does not render anchors or relation contexts omitted by
`--max-bytes` delivery planning.

Only a body that exceeds the requested cap gets
`body-truncated=true original-body-bytes=<n>` immediately before its quoted
body. `<n>` is the original raw UTF-8 byte length before terminal escaping.
Source `complete`, selection continuation, `output-complete`, and anchor counts
retain their existing meanings; body truncation is independent of all of them.
When combined with `--max-bytes`, the body prefix and metadata are projected
first, terminal escaping follows, and the overall budget measures the final
encoded output including its delivery line and newline. A cap of zero emits an
empty quoted prefix for each nonempty body while retaining the original byte
length. A cap smaller than the next rune emits only the preceding complete
rune prefix. Omitting `--max-body-bytes` preserves existing output byte for
byte.

Optional `messages list --jst` and `messages show --jst` append a
`send-time-jst` value to each rendered message record while preserving the
existing Unix-second `sent`/`send-time` value unchanged. The list schema names
the optional suffix as `[send-time-jst]`; without the flag, the schema and
records remain byte-for-byte unchanged. The value uses a fixed `+09:00`
offset and the four-digit RFC 3339 year range `0000` through `9999`; an epoch
outside that representable local-time range produces the literal
`send-time-jst=unavailable`. This is presentation only and does not change
selection, relations, or requests. It applies to primary messages, in-window
reply/around context, and emitted relation-context message records. It does
not add timestamps to quote candidates or reachability boundaries.

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
count, exact sender set and body terms when present, candidate count when
body-term, period, or index/count selection is active, applied start index,
effective concrete period bounds/day when present, requested count, actual
`items-per-page`, optional `next-start-index`, context, and primary anchor
sequences; records not listed as anchors are included reply context. The
provider's maximum-100 response remains a distinct
`source-limit`, and a response above that declared coverage fails instead of
being hidden by local selection. Repeating two senders is a truthful
two-sender-focused slice, not a claim that every displayed message is authored
by or directed exclusively between them. Exact canonical account and message
references remain the only command identities.

Byte-bounded delivery takes the longest prefix of selected primary anchors in
that same newest-first rank order even though physical message records retain
provider order. Each anchor is atomic with its complete message, its selected
direct reply context, and every reachable relation-context or relation-gap
record required to preserve the already resolved relationship facts. Shared
context is emitted once, but a message, quoted body, actor binding, or relation
record is never split to meet the budget. The encoded stdout, including its
fixed result envelope and final newline, must fit the requested budget. The
delivery record is always the absolute final output line, and its terminating
newline is part of the hard budget.

An opted-in result emits
`delivery max-bytes=<n> output-complete=<true|false> output-items=<m> output-total-items=<N>`.
`output-items` counts delivered primary anchors, not physical context or
relation records. `output-total-items` is always present and counts every
primary anchor that was eligible for delivery in this invocation before the
byte projection. `output-complete=true` means every selected anchor and its
atomic context fit and omits both continuation fields.
`output-complete=false` adds
`output-next-start-index=<rank>`, whose value is the newest-first candidate rank
for the next omitted anchor and may be passed unchanged to `--start-index` with
the same sender/period inputs. When the original invocation used `--count`, it
also adds `output-next-count=<remaining>`; the continuation replaces the
original count with that value so it cannot expand beyond the originally
selected slice. Provider `complete` and the selection record's
existing optional `next-start-index` retain their original values and meanings;
byte delivery never rewrites either one. If the fixed envelope or the first
atomic anchor unit cannot fit, the command returns the typed
`output_budget_too_small` fault and writes zero stdout bytes. Omitting
`--max-bytes` takes the existing renderer path and preserves successful output
byte for byte, including the absence of delivery metadata.

The budgeted projection keeps `relation-resolution fetch-attempts` equal to the
total exact-message attempts already performed by the invocation. Its
`targets` value is instead projected to the number of relation targets whose
relation-context or relation-gap records are actually emitted after that line.
This may make `targets` smaller than the work already represented by
`fetch-attempts`; it never implies that an omitted target was not fetched and
never changes the executed relation-resolution plan.

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

The reviewed official Chatwork API v2 operation and OAuth-scope indexes expose
neither a global full-text message-search operation nor reaction-read data.
Those outcomes remain explicit excluded capabilities rather than commands that
silently scan every room or scrape a browser. The supported alternative starts
with `rooms list --query`, then applies `messages list --room <room-id>` sender,
day, or bounded-period selection inside the provider's maximum-100 room window.
Help must state that boundary so an absent global result is never presented as
an exhaustive search result.

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

For room inputs, exact means a leading-zero-free positive decimal value of 1
through 32 digits emitted as `room_ref`, passed unchanged through
`--room <room-id>`. Names, `rid442804132`, `chatwork-room:442804132`, and
Chatwork browser URLs are rejected rather than normalized. Callers starting
from a name use `rooms list --query` and then pass one returned decimal
reference unchanged. `members find` remains exact case-sensitive substring
matching and is not altered by the room-query case-folding decision.

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
| Message create/update body | 65,535 bytes |
| File upload | 5 MiB |

The provider documents a 100-item maximum for `GET /my/tasks`, room message, room task, room file, and incoming-request lists. Those results preserve that 100-item provider bound rather than claiming the 10,000-item aggregate ceiling. A user-selected message count or byte budget cannot raise or replace that source bound, and an oversized provider result fails before selection. The optional message byte budget may lower the 16 MiB stdout ceiling for one presentation but cannot change transport, source, selection, or relation-resolution bounds. The active contracts fix cancellation, message-window semantics, provider rate-limit behavior, mutation policy, PAT failure-before-I/O behavior, and publishable schema fixtures before live I/O is enabled. Multiple accounts remain deferred.

For a read, timeout-specific Japanese wording is selected only when
`errors.Is(ctx.Err(), context.DeadlineExceeded)` or
`errors.Is(err, context.DeadlineExceeded)` is true. It retains the existing
`operation_canceled` kind, code, retryability, and exit status; actual caller
cancellation retains cancellation wording and ordinary I/O failure remains
`chatwork_unavailable`. A 403 mentions possible non-membership only when the
task has a selected room target or scope. The explanation is fixed Japanese
text; provider response text stays private.

Message create/update requires exactly one body source: literal `--body`,
injected stdin selected by `--body -`, or an infrastructure-owned read selected
by `--body-file`. Every source resolves to nonempty valid UTF-8 with no NUL and
at most 65,535 bytes before provider HTTP. `messages send` additionally accepts
up to 100 unique exact `account-ref` values through repeatable `--to`, preserving
input order, and one optional exact `message-ref` through `--reply-to`. The
reply target is read once in the selected `--room` before sending so its exact
sender can be encoded. Infrastructure emits one `[To:{account_id}]` tag per
typed recipient, then an optional `[rp aid={account_id} to={room_id}-{message_id}]`
tag, then one separating ASCII space and the original body. The original body
is not parsed or rewritten: hand-written notation remains visible and may
duplicate the typed tags. The byte bound applies to the complete expanded body,
including generated tags. Newlines, Chatwork notation, and shell metacharacters
remain opaque message text. Invalid or unreadable input
has a typed non-retryable `invalid_input` outcome and does not expose the body,
path, or private read failure. The production stdin reader delegates inherited
descriptors to one infrastructure adapter: Unix waits with bounded readiness
polling, while Windows cancels the exact synchronous reader thread and joins it.
Cancellation neither closes the caller-owned descriptor nor abandons a reader
that could consume later input. The file path must resolve to a regular file
and its dedicated read stops at 65,536 bytes while carrying the caller context.

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
relation absence, and quote author/time never reconstruct or prove a message
identity. The application compares them only with the same original bounded
source window; exact author and whole timestamp matches remain unresolved,
source-window-scoped candidates and never cause another provider read. For
`messages show`, exact-only reads have no candidates; `--context replies` and
`--around` use their existing recent source. Filtered and byte-limited outputs
retain canonical candidate references but drop an optional source sequence
when its target is hidden.

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

When a required catalog-known value is missing or wrong, the policy rejection
states the exact `--confirm=access-change` or `--confirm=destructive` value.
It preserves `mutation_rejected`, its non-retryable exit/recovery contract, and
the zero-authentication/provider-call guarantee. It never repeats the supplied
value or a private cause.

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

For an unknown `messages send` outcome, the exact recovery task is
`messages reconcile-send`. It accepts the same canonical room, body source,
ordered `--to` references, and optional `--reply-to` reference. After body
validation and any exact reply read, it obtains the current authenticated
account once and reads one latest bounded message window under the same binding
and context. It compares the complete expanded body byte-for-byte and returns
only exact matches authored by that account, with an always-present canonical `account_ref` even for zero matches and each
candidate's canonical message/sender references, sender name and send time.
Every candidate `sender_ref` equals `account_ref`. Source count still covers the whole fetched window. Missing,
invalid or failed current-account lookup fails the entire task before the
window read. This is an intentional pre-1.0 change from the earlier all-author
candidate set; selecting the recovery task does not enable another CLI command.

The original invocation's account is not established by a later lookup. The
outcome remains unknown: an earlier identical post, a different original
account, propagation delay, bounded history, restricted access, edits/deletions,
a changed body file, or a changed reply author may affect observations. A
consumed stdin stream must be supplied again. No match authorizes a retry. The
task never echoes body or file path and makes no POST.

Infrastructure may report retryable `chatwork_mutation_not_dispatched` only
when cancellation is observed before the mutation reaches `http.Do`; its
recovery points to the same exact command because that request was not sent.
Generic cancellation, timeout, or transport failure after dispatch remains an
unknown outcome and cannot recommend replay. A validated mutation success is
preserved if cancellation arrives after the provider result is returned.

## Compatibility boundary

Before `1.0.0`, contracts may evolve intentionally with tests and migration notes. Once stabilized, compatibility includes command paths, typed inputs, roles, effects, reference kinds, semantic field meanings, bounds/completeness, failures, authentication configuration, and release artifacts.

Selecting Japanese as the default human language is an intentional pre-1.0
text-contract change. Human prose is not retained in English as an alias.
Machine classification and automation remain compatible through the unchanged
identifiers listed in the language contract.

Candidate C's versioned grammar, schemas, defaults, and ordering were the compatibility promises of the first complete implementation. The P-derived `cwk-task-projection/1` deliberately broke that contract; the headerless projection made a second pre-1.0 break by removing its repeated schema/task preamble and standalone coverage record. The flat chronological `messages list` adjacency contract is a third explicit pre-1.0 refinement and superseded an unimplemented indented-tree proposal. Applying fixed positional records to the seven reviewed homogeneous collections is a fourth explicit refinement. Clients must not expect historical headers, reference dictionaries, aliases, field ordering, or grammar. Current compatibility is identified out of band by the release and enforced by catalog fields, documentation, all-route tests, and goldens. Semantic field meanings, exact canonical references, bounds/completeness, failures, and trust classifications remain governed independently of a text migration. A future replacement changes the current promises only through reviewed evidence and an explicit compatibility decision. Experimental worktree output carries no compatibility promise.

Hierarchical text help is another explicit pre-1.0 text-contract change: the
former flat root leaf list moved to catalog-derived namespace views, natural
namespace `--help` became valid, namespace command names became relative, and
exact help gained catalog-derived input facts and navigation links. The later
measured-help decision is an intentional pre-1.0 machine-contract break: public
schema-v4 agent help is removed because its complete-contract payloads did not
reduce command errors and frequently exceeded the consumer's readable output
budget. Namespace text help now includes each leaf usage and required flags;
exact text help is capped at 2,048 bytes and restricted to invocation facts,
constraints, and two examples. Runtime structured errors own recovery detail,
while complete `AgentContract` metadata remains internal validation evidence.
`help ... --format agent` now fails with migration guidance. Non-help task
contracts, exact command paths, and command execution contracts do not change.

Bounded message selection was introduced as an additive pre-1.0 command-input
change. The later index-selection decision introduced `--start-index` and
`--count` so an argument cannot be mistaken for an inclusive end rank. Measured
compatibility evidence then restored message `--limit` as a catalog-declared
deprecated alias of `--count` for at least two minor releases, with successful
execution and a stderr migration warning. The independent room-task deadline
`--limit` is unchanged. Catalog `source_limit`/text
`source-limit=...` remains the fixed 100-message provider retrieval ceiling;
message record positions and canonical references do not change.

`room-tasks create --limit` continues to accept the existing positive decimal
Unix-seconds input unchanged and also accepts strict whole-second RFC3339 with
an explicit `Z` or numeric `±HH:MM` zone. CLI converts the latter once to the
same positive Unix-second value before constructing the domain request. It
rejects fractions, offset-free or unknown-offset (`-00:00`) values, invalid
calendar/offset values, nonpositive instants, and integer overflow before
authentication or provider I/O. `--limit-type date|time` keeps its existing
independent meaning; the provider still receives integer Unix seconds and the
same limit-type value. This is the room-task deadline input, not the deprecated
`messages list --limit` alias.

The optional `messages list --max-bytes` input is an additive pre-1.0
presentation contract. It emits delivery metadata only when selected and
leaves every omitted-option byte unchanged. It does not add `--compact` or a
local response cache. The separate measured-use compatibility decision changes
active message selection's omitted context from `none` to `replies`; explicit
`--context none` retains the lower-output behavior.

The repeatable `messages list --contains` input is an additive pre-1.0 local
selection contract over the existing bounded typed result; it adds no provider
search or read. Selection metadata now preserves the ordered exact terms, and
active selections declare `context=none` as well as `context=replies` so an
explicit opt-out remains distinguishable from context expansion. This changes
the text projection for previously filtered sender, period, or index results
whose context is `none`: the selection record gains the `context=none` field.
This update affects active selection records only; unfiltered results continue
to omit selection metadata, and message record lines keep their current grammar.

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

The exact-message reply-context decision is an intentional pre-1.0 behavioral
change only for explicit `messages show --context replies`. That invocation now
performs one additional bounded recent-list read and emits direct typed child
records with source/access limitations. Omitted context and explicit `none`
remain one exact request; the existing relation-fetch budget still counts only
additional exact parent reads.

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

The saved-profile dispatch-boundary refinement adds non-retryable
`command_disabled` for exact disabled canonical paths, catalog-declared aliases,
and exact scoped/trailing help. It adds a separate name-only disabled section to
namespace help and keeps disabled-only namespaces in root navigation. A bare
one-token catalog namespace selects that same namespace help; active usage,
examples, workflows, and typo suggestions remain active-view-only. Unknown or
incomplete non-namespace paths, unconfigured profiles, and unsafe static-help
fallback retain their separate existing outcomes. This contract bounds dispatch through
the current `cwk` process; it does not claim OS-level security or human identity.

The final pre-`v0.1.0` selector presentation deliberately replaced the
machine-shaped confirmed-save line with Concept A's natural-Japanese result
suffix. The structured output field proposal changed from
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

## Local message-link discovery

`messages resolve-link --url <message-link>` is a configurable local read and
`RoleDiscover`. It accepts only the literal ASCII form
`https://www.chatwork.com/#!rid<R>-<M>` with each reference a canonical
leading-zero-free 1..32 digit positive decimal; the maximum input is 95 bytes.
It neither normalizes nor decodes the input and rejects other forms. It emits
unchanged `room_ref` and `message_ref` values and states that existence and
access were not checked. These values can be supplied unchanged to the active
`messages show --room ... --message ...` command. The URL is never accepted
as an act input, echoed in an error, used as a recovery argument, or fetched.
This local operation needs no PAT and does not add a provider API operation.
When both leaves are active, the root and namespace Japanese help make the two-step route discoverable; exact help remains invocation-only. If only the resolver is active, root help describes reference extraction and tells the agent to ask a person to enable show with config; it never presents the disabled show as executable or changes selection. Exact help remains limited to its invocation contract.
