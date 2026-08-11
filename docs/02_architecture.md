# Architecture

Chatwork CLI uses four layers, a task-oriented command catalog, typed operation intent, and one composition root. The purpose is to keep product decisions separate from external-system details while giving side effects a narrow, testable path.

## Dependency direction

```text
internal/cli  ------> internal/app
      |                    |
      |                    v
      +------------> internal/domain <------ internal/infra

internal/domain does not depend on app, infra, or cli.
internal/app does not depend on infra or cli.
internal/infra does not depend on app or cli.
```

`tools/archlint` enforces this direction for production code. Tests may use dedicated helpers, but they must not create a production bypass.

Release automation is a separate supply-chain pipeline rather than a fifth
runtime layer. A tag push owns create-only archive and GitHub Release
publication. A manual dispatch accepts only an existing stable tag and owns
read-only verification of that Release's exact asset/checksum set before it
rejoins the same exact-revision Formula audit and fresh App-token tap publisher.
Neither path imports release policy into `internal/` packages or allows Formula
recovery to mutate published assets.

## Layer responsibilities

### Domain

`internal/domain/` contains pure project vocabulary, values, validation, and invariants. It performs no network, filesystem, process, terminal, clock, or credential I/O.

The base template includes:

- `operation.Effect` with `EffectUnknown`, `EffectRead`, `EffectCreate`, and `EffectWrite`;
- `operation.TargetRef`, a stable description of the object or scope affected;
- `operation.Impact`, explicit cardinality, notification, access-change, and destructive declarations;
- `operation.Intent`, the declared effect, target, and impact presented to the execution boundary;
- `fault.Error`, stable failure kind, code, retryability, and next-action metadata;
- `page` and `apicall` envelopes for opaque cursors and finite call policy;
- secret-free authentication requirement and session metadata, including a private, non-serialized ephemeral binding passed to authenticated task ports;
- `doctor` domain values used by the default task.

Unknown effects and incomplete mutation intent are invalid domain states at an external-execution boundary.

### Application

`internal/app/` contains user-task use cases. A use case:

- interprets task-specific input;
- chooses the order of operations;
- handles deterministic composition and ambiguity rules;
- defines the smallest ports it needs;
- returns task-specific results rather than infrastructure response types.

The application layer depends on domain values and primitive types. It does not import infrastructure, parse CLI arguments, render terminal output, or construct transport requests.

`internal/app/doctorcmd` owns the local utility, while `internal/app/chatworkcmd`
owns the public provider tasks. Reusable application boundaries include
authentication gating, complete-or-no-result pagination, and policy-neutral
mutation invocation. Generic catalog fixtures remain test-local rather than
introducing another application service beside the product workflows.

### Infrastructure

`internal/infra/` implements application or domain-facing contracts for external systems. Examples in a derived project may include HTTP adapters, filesystems, credential stores, clocks, subprocesses, or platform services.

Infrastructure owns protocol-specific validation and conversion. The Chatwork
API token and authorization header never leave this layer. Infrastructure does
not decide which public command should exist, how several adapters form a user
task, or how terminal output is presented.

The Chatwork API adapter is the sole production authentication adapter. It
reads `CWK_API_TOKEN` once from the command environment, retains the token in a
private process-local record, returns only secret-free authentication metadata
across its domain-facing boundary, and resolves that exact record immediately
before provider I/O. No production package owns OAuth protocol, browser,
configuration, or credential-store behavior. [ADR 0003](decisions/0003-chatwork-pat-only.md)
pins this reduced boundary.

`internal/infra/systemdoctor` is the diagnostic adapter. `chatworkapi` owns the
production provider boundary. Generic contract tests use test-local fakes and
do not add an offline production adapter.

`internal/infra/terminalui` is the sole production terminal-mode adapter. It
confines `golang.org/x/term` and platform-scoped `golang.org/x/sys` imports to
infrastructure, requires both stdin and stdout to be terminals, enters raw input
and an alternate screen, reports the bounded terminal size, and restores every
mode it changed. `x/term` supplies terminal detection and state management;
`x/sys/unix` supplies context-responsive descriptor polling/reading, while
`x/sys/windows` supplies synchronous console reading, exact reader-thread
handles, `CancelSynchronousIo`, and VT-output mode management. The Windows read
is confined to one locked OS thread, and cancellation joins that reader before
returning. Cancellation therefore stops the platform read without leaving a
goroutine able to consume a later invocation's input. The adapter does not
interpret selector keys, choose commands, render catalog facts, or decide
whether a preference may be saved; those presentation and product decisions
remain in CLI. Non-terminal streams are rejected before a mutation attempt.

The command-selection adapter owns one bounded, strict JSON preference file:
`${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json` on macOS and
Linux, and `%AppData%\\cwk\\command-selection.json` on Windows. It keeps that
non-secret preference separate from the retired OAuth `cwk/config.json`,
resolves an existing configuration-home alias once to its absolute directory
target, rejects a symbolic-link or special-file `cwk` directory or preference
target, and replaces from a validated same-directory temporary file. Because
load and save continue from the resolved path, a later alias change cannot
redirect that invocation. Unix uses rename and directory sync through the
already-open directory root. Windows requests replace-existing through the
portable API, which does not guarantee atomicity or directory durability. The
adapter implements an application-owned load/save port; it does not decide
which catalog paths may be selected or attach CLI recovery commands.

### CLI

`internal/cli/` owns:

- `CommandSpec` and `Catalog`;
- `CommandRole` and the structured `AgentContract` for capability, inputs, outputs, complete/paged cursor binding, prerequisites, failures, authentication, and mutation facts;
- argument parsing and task-level validation;
- help and public discovery;
- output and error presentation;
- presentation-only output budgets and atomic delivery planning;
- the composition root that wires use cases to concrete adapters;
- the controlled handoff to side-effect execution.
- derivation of the active command-attention view from the complete catalog and
  the persisted exact-path selection.
- the pure command-selector model, fragmented key-sequence parser, bounded
  viewport, effect badges, and terminal frame rendered over the infrastructure
  terminal session.
- the single Japanese human locale for help, trusted CLI-authored TUI prose,
  public fault messages, recovery reasons, and descriptive agent-contract
  metadata. Stable machine identifiers remain in their existing ASCII forms;
  locale handling never crosses into opaque-reference or provider-text values.

For Chatwork output, including relationship-aware message results and the current headerless task projection, the layers divide responsibility further:

- Domain defines provider-neutral message, participant, recipient, reply, quote, list-selection context, exact-message direct-child context with source sequences/coverage/access limitation, relation-set state, relation-resolution provenance/outcomes, reachability, and unresolved-reference values. It rejects impossible or internally inconsistent graphs but performs no parsing, I/O, or rendering.
- Infrastructure decodes Chatwork wire DTOs, validates the status/header combination that distinguishes normal zero, partial/all access restriction, restricted exact message, and not-found, and parses provider-specific message notation into typed facts. It preserves external text as untrusted data and never invents a reply from To, display names, prose, or temporal proximity. A malformed recognized notation form returns the body with an unknown whole relation set rather than failing its list or retaining partial facts.
- Application use cases select the bounded data required by one outcome,
  resolve only explicit relationships selected by that outcome, and return a
  typed task result with coverage and unresolved facts. Message selection runs
  here after the one provider response: repeated exact senders form one OR
  candidate set; typed send time establishes newest-first rank with later
  provider position breaking equal-time ties; optional one-based start index
  and requested count select primary anchors; the CLI-defaulted reply context
  then adds only direct typed in-window parents/children unless the caller chose
  exact `none`. It then reuses original-source
  parents or spends the finite relation budget on exact same-room reads, one
  attempt per unique recursively reached target. Source sequences and final physical
  order remain those of the unfiltered provider window; supplemental context
  has separate provenance. Application-only sender/period/context/start-index/
  count/relation-budget fields are removed before each infrastructure call.
  For explicit `messages show --context replies`, application first retains the
  exact anchor, then issues one fixed force-recent list request with the same
  context and authentication binding. It joins only typed same-room reply
  edges targeting that anchor, preserves provider order/sequences and list
  coverage/access evidence, and makes this list source available to the
  independent outgoing-parent closure. The fixed list read does not decrement
  the exact-message relation budget. After child selection, application resets
  reply-resolution state and resolves it again against only the visible anchor
  and children; a hidden recent-window target can become resolved only through
  separately emitted parent-resolution evidence.
- CLI presentation projects that same typed result through a release-versioned text contract. The current headerless task projection starts with the result noun and emits only catalog-declared task facts, exact canonical references, task-relevant bounds/completeness/uncertainty, and trust framing for external text. A shared collection prelude emits trust and one fixed schema for the reviewed contacts, rooms, members, personal-task, room-task, file, and contact-request lists; their positional records preserve the application result slice order without aliases. For `messages list`, presentation assigns first-sender-order actor aliases, consumes application-provided original source sequences, and emits one flat record per selected item in unchanged source order. An active selection receives source count, candidate count, optional exact senders, effective period bounds/day/zone, start index, requested count, actual items per page, optional next start index, context, and anchor metadata directly from application. The provider ceiling is rendered separately as `source-limit`; provider restriction is separately `access-limitation=none|partial|all`; oldest reachability and period reachability come directly from domain values. One fixed schema assigns the positional sequence, canonical message reference, actor, send time, and terminal-safe quoted body; only optional typed edges and an affected record's `relation-state=unknown` retain labels. It never traverses or infers a thread: typed resolved source replies become scalar `reply=#N` or ordered compact `reply=[#N,#M]`, supplemental parents use canonical-reference reply edges plus separate `relation-context` provenance, and unavailable targets use typed `relation-gap` records. Unknown relation sets remain distinct from absent and complete multiple relations, and aliases remain document-local. It adds no global version/task preamble, standalone provider coverage record, raw Chatwork notation as semantic structure, provider/wire extras, empty optional shells, or non-contract defaults. Presentation does not define relationship truth, completeness, identity, or task policy. Future candidate renderers must consume the same boundary.

`messages list --max-bytes` stays entirely in CLI request/presentation state.
CLI removes it before constructing the domain request, receives the same fully
selected and relation-resolved result, and plans an atomic rendered prefix. No
domain or application request type and no infrastructure/provider request
contains the byte budget. The omitted option enters the pre-existing renderer
unchanged, preserving its output byte for byte.

Message create/update keeps body-source ownership explicit across the same
layers. CLI validates literal input and selects the 65,536-byte stdin bound;
an infrastructure stream adapter performs the injected descriptor read with
platform cancellation. A `--body-file` path remains typed request
data through domain and application without either layer receiving filesystem
access. Infrastructure alone dereferences it through the existing injected
regular-file reader while building the reviewed form request, stops at 65,536
bytes under the caller context, then validates nonempty UTF-8 and the 65,535-byte body ceiling before network I/O. The application port
therefore remains task-specific and secret-free, and file or stdin failures
cannot become provider calls.

`cmd/cwk/main.go` is a thin executable entry point. It should not contain product logic or construct adapters independently of the CLI composition root.

Production Go packages stay within `cmd/` and the four `internal/` layers. The `cmd/` entrypoint imports only context, operating-system signal handling, and `internal/cli`; process execution, network, filesystem, and third-party dependencies belong behind infrastructure ports. Repository-only programs live under `tools/` and cannot be imported by production packages.

CLI, application, and infrastructure code propagate the caller context instead of creating `context.Background()` or `context.TODO()`. The command entrypoint creates the signal-aware root and calls the context-only CLI boundary; a nil context produces a context-independent contract fault before dispatch. Infrastructure network code uses an explicitly constructed client governed by a finite call policy; package-level default HTTP clients and convenience calls are rejected by architecture lint.

## Chatwork authentication topology

Chatwork has one credential source behind the secret-free gate.

```text
command-process CWK_API_TOKEN
  -> CLI composition constructs the PAT infrastructure adapter
  -> application authentication gate receives a PAT-only secret-free requirement
  -> infrastructure issues one process-local binding for that token record
  -> application passes the binding unchanged to the Chatwork task port
  -> infrastructure resolves the exact record immediately before I/O
  -> x-chatworktoken on the fixed Chatwork API request
```

There is no authentication-method selection, probing, or fallback. A missing or
invalid `CWK_API_TOKEN` fails during composition before a provider task request.
The token determines the one account used by that command process and remains
private to infrastructure. `cwk` does not accept it in argv, persist it, or
expose login, status, logout, callback, or profile commands.

Room creation adds a command-specific identity precondition without adding a
credential selector. CLI clones the catalog requirement and binds its exact
`AccountID` from `--account`. After confirmation, the infrastructure
authenticator creates its private PAT binding, executes `GET /me` through that
same binding, and replaces provisional subject/account metadata only on an
exact canonical match. It removes the provisional binding on failure. The
application gate rechecks the resulting secret-free session before the room
task port can call `POST /rooms`; infrastructure also rechecks the stored
verified account against the room request before request construction, so a
generic binding cannot bypass the gate. The provider request contains no
account or owner field. Identity-probe failures are classified as pre-mutation
outcomes, so they cannot imply an unknown room-create result.

## Catalog as the public source of truth

`cli.Catalog` contains every public `cli.CommandSpec`. Routing, root,
namespace, and exact text help, uniqueness checks, catalog-wide effect tests,
and structured examples derive from it.

`DefaultCatalog` remains the complete public, capability, API-coverage, and
release ledger. Each leaf declares whether it is configurable. Production
loads an exact-path enabled allowlist and derives one catalog-order active view;
it does not construct a second command registry or mutate the complete catalog.
`help`, `doctor`, `version`, and the single exact `config` write are always-on;
the Chatwork task leaves are independently configurable. Missing selection
state derives an unconfigured active view containing only those four control
commands. CLI dispatch consults the complete catalog only to distinguish a
known configurable task or namespace from an actually unknown path, returning
`command_selection_required` before PAT resolution while never routing the
hidden task. A present allowlist is authoritative, so a command added in a later
release remains hidden until explicitly selected; unknown saved paths are
retained as stale upgrade evidence but never become executable. Profiles written by the preceding selector may
contain the formerly configurable exact paths `doctor` and `version`; only
those two known legacy entries are removed before active-view validation and
are omitted on the next confirmed save. This narrow migration does not make
arbitrary always-on paths valid selection entries.

The active view is loaded before trailing-help normalization and command
matching. Root, namespace, exact, and trailing text help; recovery projections;
reference workflows; and routing consume that same view. A disabled command
therefore takes the existing `unknown_command` path
before lazy PAT resolution or provider I/O. View validation is deliberately
narrower than full-catalog validation: a visible terminal producer may have no
visible consumer, but each visible required-reference consumer must have a
reachable visible producer and every visible recovery action must resolve
inside the view. Selection never auto-enables another command to repair an
invalid graph.

Examples are structured metadata attached to a public `CommandSpec`, not
another dispatcher. Exact-command text help scans active catalog metadata and
renders at most two examples whose exact steps exist in the active view. Root
help remains an index; namespace help instead renders each leaf's complete
usage, required flags, and catalog-declared required one-of input groups.

`config` is one fixed-`tool_local`-target write and the sole public
command-selection command. CLI builds its checkbox rows from
`Catalog.ConfigurableCommands()` in curated catalog order, parses fragmented
CSI/SS3 arrow input, and keeps scrolling, toggling, and viewport layout in a
pure selector model. Every row retains a textual `[read]`, `[create]`, or
`[write]` badge. Cyan, yellow, and magenta are supplemental cues only; ANSI
bytes are CLI-authored, are excluded from display-width calculation, and never
replace the text badge or encode authorization or destructive impact.
The badge field is right-padded to the visible width of `[create]`, keeping the
following exact command path at one stable display column across all effects.
CLI derives the always-on line and the profile-failure recovery island from
`Catalog.AlwaysCommands()` rather than maintaining another path list. A frame
that cannot show the complete current command identity admits only a
non-saving exit until the terminal is resized.

The selector persists only exact catalog paths. Up/Down moves, ASCII Space or
the UTF-8 encoding of U+3000 full-width space changes the in-memory draft, and
q, Escape, EOF, context cancellation, or terminal closure exits without calling
the store. Enter first validates active-view
reference and recovery closure and constructs the fixed-target mutation
request, then restores the terminal, and only after successful restoration
crosses `execution.Invoker` and calls the save port. A validation or terminal
restoration failure therefore leaves the prior profile unchanged. A raw error
after the save action begins is an uncertain mutation outcome; confirmed
success is not overwritten by later cancellation.

`doctor` supplies the required read-only reconciliation. Its normal diagnostic
result is augmented with command-selection state, source, enabled/disabled and
stale/legacy counts, plus a versioned SHA-256 fingerprint over the ordered
canonical enabled paths. Confirmed `config` success is projected separately as
a short human-readable visible/hidden/change summary with conditional cleanup;
it does not duplicate the fingerprint. An uncertain fault names the expected
`source=saved` as well as its candidate fingerprint; reconciliation succeeds
only when `doctor` reports both. An absent profile instead reports
`state=unconfigured source=missing`, zero enabled Chatwork commands, and the
deterministic empty-selection fingerprint; the interactive selector still
presents an all-selected initial draft. Missing state therefore cannot be
mistaken for a persisted replacement.
The internal catalog failure contract declares the exact runtime message
grammar, and JSON/text error fixtures are tested against it rather than leaving
these dynamic values in unvalidated prose. Malformed
serialized content is
the only load failure that
`config` treats as in-tool repairable, and it still writes only after Enter.
Unsafe objects or modes and unavailable paths refuse the selector; `doctor`
reports the corresponding unsafe or unavailable state, and the user must
repair or restore the filesystem outside `cwk` before running `doctor` or
`config` again. Exact help remains available without treating invalid state as
permission to enable Chatwork commands.

Text help is a hierarchical catalog projection, not another registry.
The root partitions the catalog into directly runnable single-word commands
and first-seen top-level namespaces, renders the direct section before the
namespace section, and preserves curated catalog-relative order within each
section; each namespace appears once with its derived leaf count. Namespace
help selects the same word-boundary prefix and renders every exact command's
complete usage and required flags relative to that prefix. A trailing `--help`
or `-h` is normalized to the
existing local `help <selector>` task only after the catalog proves the
preceding words are an exact command or namespace. Unknown selectors therefore
preserve the normal unknown-command fault, and namespaces never become implicit
executable operations.

Exact help projects `AgentContract.Inputs` rather than parsing usage text or
maintaining flag prose. It emits only one usage line, each flag's name, type,
required, required-one-of, or optional state and one-line description,
combination constraints, and at most two examples. A shared renderer enforces a 2,048-byte encoded
ceiling for every exact command, and catalog validation rejects a declaration
whose full or active-view minimum projection cannot fit it. Complete output, authentication, failure,
mutation, and workflow metadata remains available to internal catalog
validation and runtime structured errors, not as a public help dump.

Because exact text help renders every input name as CLI-authored structure,
catalog validation applies UTF-8, Unicode whitespace, control/format, and line
separator rejection to names from every input source, including environment,
configuration, and stdin. A non-argv source cannot bypass terminal-safe help
structure.

Do not add a second command list for documentation or dispatch. Internal adapters and generated operations are not public merely because they exist. A capability becomes public only when a user-task use case and command specification deliberately expose it.

`members find` is an application-derived read task. The CLI binds an exact room
reference and non-identity display-name query; application clears that query,
executes the existing `members.list` provider port task once, filters the
complete typed account collection with exact substring semantics, changes the
result task back to `members.find`, and attaches source-count/query provenance.
Infrastructure therefore owns no display-name search policy or undocumented
query parameter, while presentation receives only a validated candidate set.

`rooms list --query` is a distinct application-derived discover projection.
CLI treats the query as non-identity text, application clears it before one
ordinary `rooms.list` provider call, then scans rune windows with Unicode simple
case folding, preserving provider order and all matches. It performs no Unicode
or whitespace normalization, trim, ranking, multi-rune case expansion, provider
query, or automatic selection. The result carries query/source/candidate and
completeness provenance with canonical room references. `members find` keeps
its existing exact case-sensitive substring semantics.

At minimum, catalog validation rejects:

- duplicate or empty command paths, and a command path that is also another command's word-boundary namespace prefix;
- missing summaries or handlers;
- `EffectUnknown`;
- `RoleUnknown`;
- malformed, duplicate, role-inconsistent, orphaned, or closed-cycle reference declarations;
- missing capability, input descriptions/allowed values, output field meanings/completeness, prerequisites, or recovery commands;
- argv input metadata whose `Required` value or ordered `AllowedValues` disagree with the small bracket/`a|b` usage grammar; non-argv sources are excluded from this syntax check;
- missing or inconsistent common runtime failures (`operation_canceled`, output `output_write_failed`, standard authentication-gate failures, and standard mutation-invoker contract/policy/unknown-outcome failures), or an unknown-outcome recovery that points to another mutation;
- recovery commands that are only catalog prefixes, contain unchecked argv, use an unknown help selector, or otherwise fall outside the exact-path/`help <path-or-namespace>` grammar;
- read commands with mutation metadata; reference-bound creates without exactly
  one required CLI parent binding; reference-bound writes without a matching
  required CLI existing-target binding; fixed-target mutations with inputs or
  a mismatched target; and mutations without complete impact;
- an exact text-help projection larger than 2,048 encoded bytes, more than two
  examples, or content outside the allowed invocation facts;
- malformed or conflicting deprecated aliases, including an alias without a
  finite compatibility declaration or one that does not bind one canonical
  input;
- command metadata that cannot produce consistent help and routing.

## Command roles and opaque reference flow

`CommandRole` describes where a task sits in a user workflow:

| Role | Responsibility |
|---|---|
| `RoleUtility` | Repository or runtime operation that is neither candidate discovery nor unique-target action |
| `RoleDiscover` | Owns ambiguity and may emit opaque references with candidates |
| `RoleAct` | Operates on required declared opaque references, or on one exact catalog-declared fixed `tool_local` singleton when no selection exists |
| `RoleUnknown` | Invalid for a public command |

Reference kinds live on `AgentContract.Output.Fields`, `AgentContract.Inputs`, and the top-level cursor field owned by `AgentContract.Pagination`. `ProducedRef` and `ConsumedRef` are compatibility projections, not a second declaration. A fixed target instead declares a stable kind, ID, description, and `tool_local` scope directly on the agent contract; it cannot coexist with produced or consumed target references. The catalog derives the reference graph, scoped workflows, next actions, and fixed-target facts from those fields. A shared reference kind is an explicit claim that every value of that kind is interchangeable at each matching input; use distinct kinds when fields or target roles are not interchangeable. Every reference kind needs a producer and consumer, and the required-reference dependency graph must be reachable from at least one command that can run without an unresolved required reference. Optional first-page cursors do not create a dependency. `CommandOutput.Fields` always describe values inside the declared JSON envelope; a public `paged` output owns its separate cursor name, string type, description, and shared opaque kind in `Pagination`. Its typed `completion: "empty_cursor"` rule makes an always-present empty string the sole completion marker. Public help stays concise; internal catalog checks expose and validate the full graph, fixed target, pagination binding, and field meanings.

Public help has no agent schema. Root and namespace text views select a command;
exact text help supplies only the bounded invocation facts above. Structured
inputs still mark repeatable flags and deprecated aliases explicitly, and
argument parsing consumes those same catalog fields rather than maintaining a
second registry. Passing `--format agent` to help produces a typed retirement
error directing the caller to `--help`. Unknown flags produce a deterministic
did-you-mean suggestion from the selected command's declared flags and aliases,
but parsing never rewrites or executes the suggestion automatically. Complete
`AgentContract` I/O, failure, authentication, mutation, and workflow data stays
an internal executable contract and drives runtime structured errors.

Catalog `CommandOutput` metadata is executable compatibility data, not descriptive decoration. Generic CLI contract tests run each built-in JSON renderer and compare its `schema_version`, declared envelope, and every item key with the corresponding catalog declaration. Text-help snapshots separately fix root, namespace, exact, trailing, and retired-format behavior.

## Semantic and presentation boundary

Every presentation candidate consumes one typed semantic result; candidates do not parse one another's rendered output.

```text
Chatwork response
  -> infrastructure validates wire bounds and parses explicit notation
  -> domain values represent participants, messages, and typed relations
  -> application selects one bounded outcome context and declares coverage
  -> interchangeable CLI presentation candidates render the same facts
  -> evaluation compares agent understanding and resource cost
```

Relationship truth has three states:

| State | Meaning | Presentation requirement |
|---|---|---|
| explicit and resolved | Provider notation identifies a relation and the referenced object is available | Preserve it so the evaluation answer remains recoverable |
| explicit and unresolved | Provider notation identifies a relation but the referenced object is outside the bound or unavailable | Preserve the relation and unresolved/coverage state without fabricating the object |
| absent or unsupported | No provider fact establishes the relation | Do not imply that the relation exists |

Filtering, index selection, and context selection are application outcome
concerns. Provider pagination and notation parsing remain infrastructure
concerns. Presentation owns only representation. CLI request assembly resolves
an omitted `messages list --window` to recent before the application boundary;
explicit `recent` preserves that value and explicit `changes` selects the
differential request. Infrastructure remains a policy-free boolean-to-query
mapping: recent emits `force=1`, while changes omits `force`. In particular,
`messages list` evaluates exact-sender OR and optional half-open period
membership once over the bounded typed provider result, then ranks candidates
by typed send time, applies optional one-based
`--start-index` and maximum `--count` 1..100, and only then expands
direct non-transitive `replies` context. CLI assembly defaults an active
selection's omitted context to `replies`; exact `none` skips this application
expansion, and context without a primary selection fails before authentication.
A timestamp selects membership but
never changes provider-order output; equal timestamps prefer the later provider
position. Omitted targets remain explicit unresolved canonical references
until the bounded resolution phase; they are never guessed. Selection metadata carries source and
candidate counts, concrete period bounds/day, start index, requested count,
actual items per page, optional next start index, original source sequences,
and primary anchors so presentation need not reconstruct policy. Explicit reply
context may make displayed count exceed the requested primary count or include
a direct neighbor outside the primary period.

`messages list --limit` is catalog metadata for one deprecated alias of
`--count`, not a second application input. CLI parsing binds either spelling to
the same typed count, rejects simultaneous/conflicting uses, and emits the
successful migration notice only on stderr. The alias remains for at least two
minor releases. Other unknown flags fail before authentication/I/O and may
suggest a declared spelling without silently changing argv.

Command-path compatibility uses `CommandSpec.DeprecatedAliases`. Catalog
validation rejects invalid, duplicate, canonical-colliding, command-boundary
colliding, and alias-to-alias command/namespace-prefix-colliding spellings.
Dispatch normalizes one exact alias prefix to its canonical path before
resolving the active view, so aliases share the canonical handler, intent,
help, and fail-closed selection policy. No router-owned alias map exists.
Static help recognition is read-only and may use the complete catalog when the
local command-selection store returns the declared invalid/unsafe/unavailable
faults. Root and namespace rendering receives invocation-local fallback state
and adds a static-catalog/`doctor` notice; exact help remains invocation-only.
Non-help invocations keep the original fault.

Leading root-option validation precedes catalog validation and active-view
loading. A typo hint can therefore consult only the fixed global option set and
cannot trigger configuration, authentication, or task I/O. After the active
view is derived, unknown-command guidance ranks only its canonical
`CommandSpec.Path` values; it never consults the disabled portion of
`DefaultCatalog`, deprecated command aliases, or handlers, and it never
normalizes the suggested argv.

`messages list --max-bytes` is a distinct presentation bound, not message
selection, application pagination, or provider pagination. CLI validates its
decimal value in the inclusive range 1..16,777,216 before authentication. The
normal use case still performs its one existing room-message list request and
the same bounded exact-message relation reads; the budget neither reaches nor
changes either request. After the complete result returns, CLI chooses the
longest primary-anchor prefix in application-provided newest-first rank order,
then renders the retained union in provider order.

One delivery unit contains the whole primary message, its selected direct reply
context, and the reachable relation-context and relation-gap closure needed for
those retained records. Shared records may be deduplicated, but no message,
body, actor binding, or relation record may be split. The complete encoded
stdout envelope is measured, including the delivery record and final newline.
The delivery record is the absolute final output line, and its terminating
newline remains inside the hard budget. Its keys are exactly `max-bytes`,
`output-complete`, `output-items`, always-present `output-total-items`, and
optional `output-next-start-index` plus conditional `output-next-count`; the
former is the next omitted anchor's newest-first rank, and the latter replaces
an original count with the remaining anchors so continuation cannot expand the
selected slice. `output-items` counts delivered primary anchors rather than
physical records, while `output-total-items` counts every primary anchor
eligible for delivery before projection. Provider `complete` and application
selection `next-start-index` pass through unchanged and remain semantically
independent from output delivery.
For a budgeted projection, CLI recomputes `relation-resolution targets` from
the relation-context and relation-gap targets it will actually emit after that
summary line. It preserves `fetch-attempts` from the complete application
result because that is the total work already performed before presentation.
The projection changes neither value in the application result itself.
When the fixed envelope or first atomic unit does not fit, CLI returns
`output_budget_too_small` before writing stdout. A budgeted failure therefore
cannot expose a partial prefix.

Public CLI assembly defaults `--resolve-relations` to five and accepts 0..100.
After selection, application walks every explicit same-room reply edge and
collects unique targets from displayed source records in first-reference order. A matching message from
the original source becomes supplemental source context at zero call cost;
otherwise each target consumes one exact `messages show` operation until the
budget is exhausted. Exact success must match both requested room and message.
Not-found and restricted become target states; every transient, cancellation,
malformed, rate, or other permission fault aborts the list task. The admitted
authentication binding and caller context are reused unchanged. Fetched
context is not appended to provider sequences. Every explicit same-room reply
on that context joins the breadth-first queue; duplicate and cyclic targets
are visited once.

For `messages show`, context remains a dedicated typed result rather than
appending children to the exact-result `Messages` variant. `MessageShowContext`
binds the requested anchor to an explicit child slice, original source
sequences, visible source count, bounded latest-window coverage, and access
limitation. Domain validation requires every child to be in the anchor room and
to carry a resolved typed reply edge to the anchor. CLI renders the anchor
first, then one bounded reply-context record and each child canonical record,
then the existing parent-resolution records. Omitted/`none` requests cannot
carry this context.

CLI request assembly parses whole-second offset-bearing RFC3339 bounds and
resolves `--on YYYY-MM-DD|today|yesterday` once with its injected clock and the
fixed `Asia/Tokyo` calendar. Only the effective Unix interval and optional
concrete day/zone cross the application boundary; the relative word and host
local zone do not. Missing clocks and invalid or conflicting period syntax fail
before authentication. Application removes those period fields together with
sender/context/index/relation-budget fields before calling infrastructure.

Application derives oldest reachability from the complete typed source before
local selection. A nonempty, unrestricted `recent` source with positive send
times yields the minimum-time source message; period bounds are classified
against it as within, partially outside, wholly outside, or unknown. A
`changes`, empty, access-limited, or invalid-clock source cannot prove the
boundary and remains unknown.

Chatwork documents no limit, cursor, or offset for the list endpoint.
Infrastructure issues exactly one list request with only the documented
`force` query, then zero through N distinct exact-message requests selected by
application's declared budget. Each infrastructure operation remains one
request and one attempt. The adapter rejects application-only sender/period/
index/context/relation-budget state if it crosses the port. The
provider's maximum-100 coverage is a `source-limit`, not the public requested
count or a provider page size. An over-bound source result fails before
application selection, and invalid public index/count values fail before
authentication or I/O. The catalog result remains one complete bounded task
result with no `AgentContract.Pagination` binding. The SCIM-derived public
vocabulary does not manufacture a provider cursor or snapshot guarantee.

Candidate C (`cwk-context-capsule/1`) is the first stable public presentation baseline and retains that historical evidence. Competition 1 was inconclusive because benchmark defects made its promotion result non-authoritative. A P-derived task projection (`cwk-task-projection/1`) was selected afterward by an explicit owner compatibility decision, then hardened beyond the frozen candidate. The current default further removes its in-band schema/task preamble and standalone provider coverage line through a second explicit pre-1.0 compatibility decision. Neither decision describes P as the benchmark winner. Future alternatives may be developed in isolated worktrees against the same semantic fixtures, answer key, trust rules, canonical references, and output-boundary requirements. Candidate-specific schemas, grammars, ordering, shorthand, or visual hierarchy remain outside domain and application code, and raw experimental evidence remains immutable decision input.

Upstream coverage is separately pinned in `.harness/chatwork_api_v2.json`. That manifest may prove that every fixed official operation has a public task owner, but it cannot dispatch a request or generate a command. `cli.Catalog` remains the only public-command source of truth.

The same manifest pins the first implementation's reviewed resource ceilings and the exact upstream operation IDs in each confirmation class. Production code uses compile-time typed constants with those values; it does not load harness JSON at runtime. `tools/contractlint` detects drift in the independent evidence, while adapter, application, and CLI tests prove enforcement at the transport, source-cardinality, selection, upload, and rendered-output boundaries. Its current `coverage_status` is `complete`, so every one of the fixed 32 operations must retain at least one public capability owner.

The representative public graph is:

```text
rooms list
  RoleDiscover
  produces {kind: chatwork-room, field: room_ref}
       |
       | exact opaque value
       v
messages list --room <room-id>
  RoleAct
  consumes {kind: chatwork-room, argument: --room}
```

The current task projection emits the exact canonical `room_ref` directly and
defines no display alias; only a leading-zero-free positive decimal value of 1
through 32 digits is accepted unchanged by `--room <room-id>`. A display name,
`rid...`, `chatwork-room:...`, or Chatwork browser URL is rejected. Name-first
workflows use `rooms list --query`, whose one application-owned case-insensitive
filter returns every candidate and never invokes an action.
Historical candidate-C aliases were document-local and were never accepted by
commands. Generic reference-graph tests use synthetic test-local catalog specs;
the public Chatwork graph remains the only runnable discover-to-act evidence.

## Operation effect and intent

Effect answers **what class of action occurs**:

| Effect | Meaning |
|---|---|
| `Read` | Observes state without intentionally changing it |
| `Create` | Creates a new object within a declared parent or scope |
| `Write` | Changes or removes an existing declared target |

Intent answers **what this invocation is allowed to affect**. A mutation intent includes a `TargetRef` and an `Impact`. Cardinality and the notification, access-change, and destructive dimensions must all be explicit; their zero values fail closed. Derived projects extend this base with domain-owned detail such as recipients, visibility changes, publication destinations, or workflow triggers.

`MutationContract` connects the public target declaration to runtime intent. For
a reference-bound `Create`, `parent_input` names the single opaque parent or
scope reference and `target_id_input` is absent because the object does not
exist yet. For a reference-bound `Write`, `target_id_input` names an opaque
reference whose kind equals `TargetKind`; an optional, distinct `parent_input`
can bind additional scope. `target_inputs` contains exactly these named roles.
A command-bound singleton instead requires one matching fixed target, an
explicitly empty `target_inputs`, and no input role; the effect determines
whether it is the create scope or existing write target. A free-form, optional,
non-CLI, duplicated, non-reference, remote-as-fixed, or mismatched binding fails
catalog validation rather than being treated as safe.

The catalog binding and runtime `TargetRef` are complementary checks: the former proves that an agent can supply an unambiguous target from the command contract, while the latter proves that the concrete invocation presented to policy and infrastructure has the declared target and impact. Neither replaces the other.

Do not infer effect from an HTTP method, command name, or adapter function. A `POST` may be a read-like query; a local file write is still a side effect without HTTP. The product declaration is authoritative and must agree with the concrete operation at runtime.

## Execution flow

```text
argv
  -> CLI derives the active view from DefaultCatalog and local selection state
  -> CLI selects one CommandSpec from that Catalog view
  -> CLI validates role/reference declarations, parses task input, and chooses presentation
  -> application use case interprets the user outcome
  -> domain validates Effect, Intent, TargetRef, Impact, auth, and API envelopes
  -> controlled execution boundary snapshots intent and applies derived policy
  -> infrastructure adapter performs one bounded logical operation
  -> application returns a task result
  -> CLI renders stable output and exit behavior
```

For mutations, validation failure must occur before the external side effect. `app/execution.Invoker` provides the common ordering and has no permissive default policy. Dry-run, human approval, OS authentication, authorization reuse, confirmation, and audit behavior remain derived policy rather than command-local conventions.

The Chatwork policy implementation derives one of three confirmation requirements from typed impact and the fixed operation contract: exact invocation only, exact `--confirm=access-change`, or exact `--confirm=destructive`. CLI parsing supplies the typed confirmation, application policy compares it with the snapshotted intent, and infrastructure never interprets the flag. Every logical provider operation permits one transport attempt. An unclassified post-action result routes only to a catalog-declared read-only reconciliation task.

Invite-link update is a full-replacement use case. CLI/domain enforce the XOR
between exact code and explicit regeneration and require approval plus
nonempty description before authentication. Infrastructure repeats the
invariant before form construction, sends `need_acceptance` and `description`
on every update, and omits `code` only for explicit regeneration. It does not
perform GET/merge or recover a code from the response URL.

## Error ownership

- Domain errors explain invalid values or invariants.
- Application errors explain task failure or ambiguity.
- Infrastructure errors map unstable upstream details into a stable `fault.Error` without leaking secrets.
- CLI maps fault kind, code, retryability, retry-after, and next actions to stable human and machine presentation and exit statuses.

The Chatwork adapter owns rate-limit evidence. It reads only one strict decimal
`x-ratelimit-reset` value, rejects values outside the official five-minute
window relative to both a valid response date and the local response time, and
uses a valid response `Date` as the wait-duration baseline, falling back to the
local response time only when that header is absent or invalid. It does not
interpret `Retry-After` as a Chatwork contract. For message/task room
posting it may privately inspect the bounded error envelope; only the exact
documented single error selects the combined-room 10-second wait. No provider
body crosses the infrastructure boundary. The catalog declares separate read
and mutation rate-limit faults and recovery: reads may retry the same task;
mutations remain non-retryable and route to scoped help. CLI presentation maps
missing timing to text `unknown` and JSON `null`; timing never overrides the
catalog retry decision.

Public fault messages and next-action reasons are Japanese, while kind, code,
retryability, retry-after representation, next command, JSON keys, and exit
status remain locale-neutral contracts. Internal validation strings may remain
English when they are not rendered directly; the CLI must not expose a raw
internal cause as a substitute for a localized public fault.

The stable exit mapping is `0` success; `2` invalid input; `3` internal; `4` authentication; `5` permission; `6` not found; `7` ambiguous; `8` rate limited; `9` unavailable; `10` rejected; `11` canceled; `12` unsupported; and `13` contract violation. Success output is written to stdout, while text or schema-versioned JSON failures are written to stderr. A zero status requires a complete successful write. A failing `doctor` report may be rendered in full before its structured `rejected` failure so callers receive evidence without treating an incomplete result as success.

Do not render inside domain, application, or infrastructure packages. Do not make a use case parse human-facing error strings from an adapter.

## Adding a vertical slice

Add capabilities in this order:

1. User outcome and product-contract decision.
2. Utility, discover, or act role and any opaque reference flow.
3. Domain vocabulary, effect, intent, reference validation, and invariants.
4. Application input, result, use case, and owned ports.
5. Infrastructure adapter satisfying those ports.
6. CLI arguments, rendering, and one `CommandSpec`.
7. Unit tests at each layer and catalog/reference/execution contract tests.
8. Harness and documentation updates for any new invariant.

This order makes product intent reviewable before transport code creates momentum toward the wrong public API.

## Optional capabilities

Concrete schema generators, update checks, credential stores, OAuth libraries, telemetry, package-manager formulas, code signing, and platform-specific authorization are optional modules, not base-layer responsibilities. A derived project adds one only after documenting:

- the user outcome it enables;
- its trust and compatibility boundaries;
- dependency and maintenance cost;
- failure behavior;
- mechanical checks and release impact.
