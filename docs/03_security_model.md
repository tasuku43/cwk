# Security Model

This document defines the minimum security reasoning expected from Chatwork CLI and every derived project. It does not claim that the base scaffold makes project-specific integrations secure. Its purpose is to make assets, effects, trust boundaries, and enforcement visible before implementation spreads them across commands.

## Security objective

The CLI must not perform a side effect broader than the validated user task and declared `operation.Intent` and `operation.Impact`. Invalid, unknown, or inconsistent operation state fails before external I/O. Credentials, confidential identifiers, and private repository history must not enter public source or artifacts.

## Assets in the base template

- Integrity of the user's local system and files.
- Confidentiality of arguments, environment values, and future credentials.
- Integrity of command selection, effect classification, and target binding.
- Integrity of source, dependencies, generated files, and release artifacts.
- Confidentiality of information excluded by the public-repository policy.
- Availability of a deterministic local verification path.

A derived project must add every remote account, object type, local state file, credential, cache, log, subprocess, network service, and published artifact it introduces.

## Actors and assumptions

The model distinguishes:

- **User:** invokes the CLI and may provide malformed input accidentally.
- **Coding agent:** can edit files and execute commands within its granted environment; it is not proof of human authorization.
- **Contributor:** can propose source changes but cannot redefine policy without review.
- **External system:** returns untrusted data and may fail, drift, rate-limit, or be compromised.
- **Attacker:** may control input, remote content, a dependency, a copied file, or a proposed patch.
- **Release owner:** is trusted to follow the reviewed release procedure, not to bypass it manually.

The base model does not assume that a TTY, parent process, environment variable, or successful process launch proves human intent.

## Trust boundaries

```text
untrusted argv / environment / files
                |
                v
       CLI parsing and validation
                |
                v
      application task boundary
                |
                v
 Effect + Intent + TargetRef + Impact validation
                |
                v
     controlled side-effect boundary
                |
      +---------+----------+
      |         |          |
 filesystem  process    network / platform service
```

Remote responses cross the boundary in the opposite direction. They remain untrusted data after parsing, bounding, and safe structural rendering; those steps do not turn remote prose into trusted instructions.

## Effect and intent policy

| Effect | Required declaration | Base policy |
|---|---|---|
| `Read` | Stable task and bounded source | May execute without mutation authorization; still validates input, destination, and output bounds |
| `Create` | Parent or creation scope in `TargetRef`; complete base `Impact` | Fails closed without complete intent and impact; derived project decides confirmation or human authorization |
| `Write` | Existing target in `TargetRef`; complete base `Impact` | Fails closed without complete intent and impact; product-specific consequences require additional typed detail |
| `Unknown` | None | Never executable |

HTTP methods, SDK names, or command verbs do not determine the effect. A query sent with `POST` may be a read; writing a local output file is a side effect even when no network request occurs.

The base `Impact` always declares cardinality and whether the operation sends notifications, changes access, or is destructive. When those dimensions cannot explain policy, extend intent with a typed domain model before exposing the command. Examples include:

- multiple targets;
- irreversible deletion;
- access or ownership changes;
- messages or notifications sent to other people;
- execution of user-provided code;
- publication outside the user's account or machine.

Do not encode these distinctions only in a human-readable confirmation string.

## Opaque reference boundary

The local message-link discover task treats the entire supplied link as
untrusted. It accepts only the exact documented ASCII form with two canonical
references and at most 95 bytes, performs no URL normalization, decoding,
network access, or credential lookup, and emits only those validated references
and an explicit not-verified state. It does not echo the raw link or put it in
recovery instructions. A parsed link does not prove a target exists or that an
account may access it. Act tasks still reject URLs as reference inputs.


Discovery output may contain labels controlled by an external system, but action identity comes from the declared opaque reference field. Pass the emitted ID unchanged into the consuming argument.

- Validate the documented shape and length before adapter execution.
- Do not authorize against a display name, row position, copied URL, or reconstructed resource path.
- Do not case-fold, decode, unescape, trim, or otherwise normalize an opaque ID unless its domain contract explicitly defines that operation.
- Bind mutation `TargetRef` to the same validated ID accepted by the action command.
- Keep remote labels out of authorization identity and sanitize them separately for display.
- Reject control/format runes and Unicode line or paragraph separators at opaque transport boundaries where they have no valid protocol role. Validation rejects; it never silently rewrites an ID, cursor, target part, or idempotency key.

The only reference-free action target is a catalog-declared fixed
`tool_local` singleton with a stable kind, ID, and description. It cannot be a
provider object, cannot be selected from credential presence, and cannot be
used once multiple possible instances exist. Catalog tests keep this path
disjoint from opaque-reference flows and bind its mutation target explicitly.

Public Chatwork commands apply the exact-byte rule to typed canonical
references. Domain tests reject alternate and unsafe forms, catalog tests prove
producer-to-consumer binding, and a CLI runtime test passes a discovered
account reference unchanged into exact sender selection. Generic catalog
fixtures remain test-local and do not introduce another resource identity.

A room action accepts only a leading-zero-free positive decimal canonical room
reference of 1 through 32 digits through `--room <room-id>`. A display name,
`rid...`, kind-prefixed text, or Chatwork browser URL is rejected before
authentication or adapter access rather than decoded into authority. The
separate `rooms list --query` discover path may case-fold external room names,
but it returns every match and never forwards one automatically to an action.

### Presentation-derived identity

A presentation candidate may introduce shorthand, positions, labels, grouping, or other derived display values. None is authorization identity by default. Public action inputs, policy, logs, and infrastructure use validated canonical references. If a later presentation proposes reusable shorthand, it requires a separate typed contract defining scope, lifetime, collision handling, and exact resolution; the presentation experiment itself cannot grant that meaning.

## Chatwork notation and relationship trust

Chatwork notation is untrusted provider data with a documented syntax, not executable instruction. Infrastructure parses only reviewed bounded forms into typed facts.

- To establishes recipient identity but not a reply edge.
- Every complete reply tag establishes one provider-declared room/message relation after identifier validation; multiple tags remain separate ordered facts.
- Quote metadata remains unresolved; missing message identity is never reconstructed from author, timestamp, text, or an observed candidate. The application may expose source-window candidates only after exact same-room, canonical quote-author, and whole Unix timestamp equality. Every candidate stays `identity=unproven`, is not a reply edge or fetch target, and preserves a canonical message reference. Partial-source matches describe only observed records; zero matches under partial or unavailable access are unknown. Existing source coverage/access remains explicit, and no quote adds provider I/O.
- Malformed, nested, contradictory, or unsupported recognized notation keeps the bounded external body, drops every partial relation fact for that message, and marks the whole relation set unknown; malformed wire identity, UTF-8, JSON, or response bounds still fail closed. One malformed body never discards otherwise valid list records.
- Message access-limitation headers are untrusted protocol evidence. Only the official `true` value on its documented list/exact-message status is promoted; malformed or contradictory combinations fail closed, and the provider summary is neither exposed nor parsed into policy.
- Display names embedded beside tags cannot override account identifiers.
- Proximity, layout-looking text, prompt-like prose, and copied tags inside quoted/code-like content do not create authorization or mutation targets.

The typed task result distinguishes explicit resolved, explicit unresolved, and absent relations before presentation. Every candidate must preserve that answer; a clearer or smaller display is not allowed to strengthen a relation. Quote candidate states are separate evidence, never relation resolution: `messages list` reports unresolved replies, unresolved quotes, and unknown relation sets separately, and a unique source-window candidate still counts as an unresolved quote.

`messages list --max-body-bytes` limits only the displayed prefix of external
body text. It counts original UTF-8 bytes and keeps only complete rune
boundaries before the existing structural terminal escaping. The truncation
marker and original byte count are CLI-authored metadata; printable body
meaning remains untrusted and is not filtered. Matching and typed relation
inference always use the full validated body before presentation, and a
truncated display never implies source or result incompleteness.

`messages list` may use a validated same-room reply target as the exact
destination of an additional read. It never derives that destination from To,
quote metadata, names, timestamps, prose, or rendered text. The public default
permits five exact reads and `--resolve-relations` is constrained to 0..100.
Breadth-first first-reference ordering, one visited set, and one attempt per
unique target bound recursive chains and cycles. Every exact result must match
the requested room and message. Not-found and exact-message restriction may be
retained as typed target evidence; cancellation, rate limits, unavailability,
malformed results, and other permission faults fail without partial success.

`messages show --context replies` performs one additional bounded list read in
the already validated anchor room under the same ephemeral authentication
binding and caller context. It does not derive or broaden a room target. A
message becomes a child only when infrastructure supplied a valid typed reply
edge whose room and target exactly match the anchor; body text, copied tags,
To/quote facts, names, and timestamps are not consulted. The result retains
latest-window and access-limitation evidence so restricted or out-of-window
children are never reported absent. This fixed read is declared independently
from the finite exact-parent fetch budget.

`messages show --around N` uses that same bounded read in the already validated
anchor room, with the unchanged caller context and authentication binding. It
adds no room lookup, identity selection, or request derived from adjacent
message content. The exact read remains first. If a restricted source does not
prove the anchor's position, the result publishes `unknown` and no neighbors;
neither message identifiers nor timestamps are used to manufacture positions
or reply relationships.

`messages resources` trusts only typed file/task `message_id` references after
infrastructure has validated them. Application compares each reference and
room exactly with the caller's inputs. It does not inspect message bodies,
`[info]`/`[dtext]`/`[download]` notation, names, timestamps, or prose to create
an attachment/task event. Every selected collection remains bounded and
incomplete, so no matches are reported as `absence=unknown`; any source fault
suppresses the combined output rather than exposing partial results.

## Controlled execution boundary

All filesystem writes, subprocess execution, credential access, network calls, and platform services must have a bounded construction path. A command or use case must not receive an unrestricted client merely because it is convenient.

Message create/update accepts exactly one body source. Literal argv is
validated directly; `--body -` reads at most 65,536 bytes from the CLI's
injected stdin, and `--body-file` carries only a path through domain and
application layers before infrastructure uses its injected, caller-context,
regular-file reader. Production pipe/terminal stdin goes through an
infrastructure descriptor adapter: Unix uses bounded readiness polling and
Windows targets the exact synchronous reader thread for cancellation. The
Windows worker is joined, and neither platform closes the caller-owned
descriptor or abandons a reader that could consume later input. The body-file
reader rejects non-regular paths and stops at 65,536 bytes.
Every source must produce nonempty valid UTF-8 of at most 65,535 bytes
with no NUL before provider HTTP begins. Typed `invalid_input` faults expose
neither the body, path, nor private read error, and tests prove zero provider
calls for unreadable or invalid sources.

For message send, `--to` accepts at most 100 unique exact account references;
duplicates fail before stdin consumption, authentication, or provider I/O.
`--reply-to` accepts one exact message reference and cannot select a different
room. Its preflight performs at most one exact read in the already selected
room using the same context and secret-free authentication binding as the
subsequent POST. The application requires the returned message and room to
match the requested refs and requires a valid account ref for its sender.
Typed To/reply inputs stay separate from the `--room` create target. Generated
provider notation is prefixed in deterministic order without parsing or
rewriting caller text, so callers who also hand-write a tag may intentionally
produce duplicate notation. Final UTF-8 byte validation occurs after
prefix-generation and before the POST. For a reply send using `--body-file`,
infrastructure reads and validates the source once before the reply GET and
passes an opaque one-use handle through the application back to the same
adapter. The application receives neither filesystem access nor the prepared
bytes, and the adapter consumes the handle once without rereading the path.
Missing, unreadable, empty, malformed, or oversized body-file input therefore
makes zero provider calls. If only the final generated-prefix expansion is
oversized, the reply GET may already have succeeded but the POST still remains
zero. A failed read, invalid/mismatched result, cancellation, or invalid final
body sends no POST; only faults that prove this pre-dispatch state may guide a
same-command retry. Once POST dispatch may have begun, the existing
non-retryable unknown-outcome and read-only reconciliation rules remain in
force.

For a mutation, the boundary performs this order:

1. Snapshot and validate the operation input.
2. Validate `Effect`, `Intent`, `TargetRef`, `Impact`, and concrete destination consistency.
3. Apply project-specific policy such as dry-run, human authorization, or confirmation.
4. Execute the side effect once.
5. Return a bounded result and audit event if the product requires one.

A rejection at steps 1–3 must result in zero mutation attempts. Tests use fakes or counters to prove this order.

Cancellation is interpreted relative to that boundary. Before step 4, the invoker can prove zero attempts and may expose retryable `operation_canceled`. After step 4 begins, a valid structured adapter fault remains authoritative and is copied without its private cause. A raw error or cancellation cannot prove rollback; it becomes non-retryable `unclassified_mutation_outcome`, whose catalog recovery must be a read-only reconciliation command. A confirmed nil result is not replaced by a later context cancellation.

Application ports treat both a nil interface and an interface containing a typed nil pointer as missing configuration. The shared port check prevents a dependency-wiring mistake from becoming a panic after validation or policy approval.

### Chatwork mutation policy

The first Chatwork implementation treats a fully specified exact invocation as sufficient confirmation for ordinary creates and updates. It does not add a generic “confirm every write” prompt that an agent could satisfy mechanically without understanding the operation.

Two higher-impact classes fail before provider I/O unless argv contains the exact typed confirmation:

- `--confirm=access-change` for room creation, room-member replacement, invite-link creation/update, and incoming-contact-request acceptance;
- `--confirm=destructive` for room leave/delete, message deletion, invite-link deletion, and incoming-contact-request rejection.

The catalog binds each confirmation to the corresponding operation impact; a confirmation for one class does not satisfy the other, persist across invocations, or prove human approval. Missing, duplicated, misspelled, or inapplicable confirmation is invalid input or policy rejection with zero mutation attempts.

For a missing or wrong required value, the public policy refusal may state the
exact catalog-known `--confirm=access-change` or `--confirm=destructive` input.
The message is chosen from those fixed values, never from the supplied argv or
private cause. The public outcome remains non-retryable
`rejected/mutation_rejected` with catalog recovery and zero authentication or
provider calls.

Room creation never treats caller-provided `--account` as authentication
evidence or room ownership. The same private PAT binding must first return that
exact account from `/me`; mismatch and identity-probe failure remove the
provisional binding and make zero room-create calls. Neither the requested nor
actual identity is copied into a public mismatch detail. The provider room
form contains no owner/account field. Infrastructure revalidates the stored
verified account immediately before request construction, so a generic or
mismatched binding cannot become an alternate authorization path. Official
room-name and icon-preset validation also completes before authentication.

Invite-link update never sends an empty or partial form. Exact code versus
explicit regeneration, approval, and nonempty description are validated before
authentication and again at form construction. This prevents omission from
silently regenerating a code, applying the documented approval default, or
depending on the undocumented description-omission behavior. No URL parsing,
prior-value merge, or empty-description clear is used to invent provider
semantics.

No Chatwork operation is retried automatically in this implementation. After an uncertain mutation result, the only recovery is the exact read-only reconciliation command declared by that mutation's catalog contract. A reconciliation may report absence or ambiguity; it never converts uncertainty into permission to repeat the write.

One cancellation fault proves a narrower phase: infrastructure emits
`chatwork_mutation_not_dispatched` only while it can still prove that the
mutation has not reached `http.Do` (adapter entry, cancellation during local
request/body-file construction, or the final pre-dispatch context check). It
is retryable and points to the same exact command because no mutation request
was sent. A generic timeout, cancellation from `http.Do`, trace event, or
service-level observation cannot make that claim; once `Do` may have begun,
the result is non-retryable `chatwork_mutation_outcome_unknown` with the
catalog-declared read-only reconciliation action. After a port returns a
validated mutation result with no error, later cancellation does not replace
that confirmed success. Read tasks retain their cancellation behavior.

## Credentials and secrets

The core contains a secret-free authentication contract, an ephemeral
non-serialized session binding, and an application gate. The current Chatwork
adapter makes the PAT source concrete below. Any future credential method or
storage mechanism must document:

- credential issuer and scope;
- acquisition and any refresh flow;
- storage location and operating-system protection;
- how secrets are kept out of process arguments, logs, errors, history, and generated files;
- revocation and expiration behavior;
- tests and scans that fail on unsafe handling.

Secrets must not cross from infrastructure into domain or application values and must not be accepted through command-line arguments when a safer channel is available. Do not persist tokens in plaintext configuration or use real credentials in ordinary CI gates. The separately opted-in scheduled live smoke check is limited to a dedicated test-account PAT, fixed read-only CLI tasks, default-branch execution, and no provider-output logging or artifacts. It does not run on pull requests or when unconfigured. Read [Authentication](07_authentication.md) before changing the current PAT boundary; a future OAuth proposal must also start from [ADR 0001](decisions/0001-oauth-library-boundary.md) and a new accepted product/security decision.

The first Chatwork implementation supports one account per command process
through the API token in `CWK_API_TOKEN`. There is no method selector or second
credential source. Missing or invalid token input fails before a provider task
request.

Environment delivery is an explicit non-persistent automation trade-off, not a
claim that environment variables are a secure credential store. The CLI does
not accept the token in argv, configuration, stdin data payloads, or output;
infrastructure reads it once, keeps it in memory behind an ephemeral binding,
and redacts it from every error and test diagnostic. The tool exposes no
credential login, status, logout, callback, profile, or persistence surface.

The token may be sent only to the exact HTTPS origin
`https://api.chatwork.com` under the `/v2` base path as `x-chatworktoken`.
Redirect following is disabled for credential-bearing requests. A different
base URL can be injected only through internal test construction with synthetic
tokens and a local server; there is no public flag or environment override for
the production destination.

## Filesystem, process, and network policy

### Filesystem

- Validate output paths and overwrite behavior before writing.
- Use restrictive permissions for confidential data.
- Define atomicity, symbolic-link, and partial-write behavior where relevant.
- Keep caches, state, configuration, and audit data separate.

#### Command-selection dispatch boundary

The persisted exact-path allowlist defines the operation scope that the
current `cwk` process dispatches. A saved valid profile makes a configurable
task outside that scope non-invocable through this process. An exact disabled
canonical path, catalog-declared alias, or exact scoped/trailing help request
returns non-retryable `command_disabled` before handler dispatch, PAT
resolution, or provider I/O. The fault names the canonical command and asks in
Japanese for the user to enable it through `config`; its next action is the
exact `config` command. An exact bare catalog namespace is a local help
selector. With a saved valid profile it renders active details and any
disabled names only; with unsafe selection storage it follows the marked
static-catalog fallback. It never initializes a provider or disguises disabled
details as active. Other unknown or incomplete paths remain `unknown_command`,
and an absent profile retains `command_selection_required` for known
configurable namespaces and tasks.

This is a boundary on `cwk` dispatch, not adversarial OS security, proof of
human identity, or a defense against the same OS user. That user can edit or
remove the file, automate a terminal, change the configuration base, install
another executable, or use a PAT outside `cwk`. The command scope therefore
must not be presented as a provider permission or as a general restriction on
the principal. PAT validation, provider permissions, canonical-reference
validation, effect/intent/impact, and access-change/destructive confirmation
remain independent controls after a command is enabled.

The absent profile is distinct from a saved empty profile: it exposes the
control plane and returns `command_selection_required` for known configurable
tasks until the user saves a selection. A saved empty profile returns
`command_disabled` for those same exact tasks. Deleting the preference returns
to first-run behavior; it does not silently activate Chatwork tasks.

`help`, `doctor`, `version`, and the single exact `config` write remain
catalog-declared always-on so the local boundary can be diagnosed and changed.
Namespace help may list disabled canonical names separately and without usage
details. Active usage, examples, workflows, and typo suggestions exclude
disabled commands. Re-enabling a path only admits dispatch through this `cwk`
process; it does not grant Chatwork authority. PAT authentication, provider
permissions, canonical-reference validation, typed intent and impact, and
access-change/destructive confirmation still run unchanged. An attached TTY,
raw mode, cursor movement, Space, and Enter prove no human identity or
authorization. Enter records the local dispatch scope through the
fixed-target mutation policy; it cannot approve a Chatwork action.

The interactive screen retains textual `[read]`, `[create]`, and `[write]`
badges. Cyan, yellow, and magenta are redundant visual cues, never the source
of effect truth; red is not used because effect is not a destructive-severity
classification. ANSI sequences are fixed CLI structure excluded from display
width, while command text is visibly escaped before it can enter a row. A
catalog string cannot introduce a terminal control sequence or replace an
effect badge.
If the terminal cannot display the complete current command path together with
its checkbox and effect, movement, Space, and Enter are ignored and the screen
requests a resize. Only q/Ctrl-C/closure remains effective, so clipped or absent
identity cannot authorize an unseen preference replacement.

The non-secret preference is stored at
`${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json` on macOS and
Linux and `%AppData%\\cwk\\command-selection.json` on Windows. It is separate
from PAT state and from ADR 0003's retired OAuth `cwk/config.json`, and its
schema accepts only bounded exact command paths. It must not contain a token,
provider response, personal data, or copied command metadata. On macOS and
Linux, infrastructure permits the configuration home itself to be an existing
symbolic-link alias: it resolves the complete alias once to an absolute
directory and uses only that result for the invocation. It still rejects
symbolic-link or special-file `cwk` directories and preference files and
replaces from a restricted same-directory temporary file. This distinction
supports user-controlled dotfiles namespaces without letting a mutable owned
leaf redirect replacement. Unix uses rename plus opened-directory sync;
Windows replace-existing has no portable atomicity or durability guarantee.
Malformed, unsafe, or unavailable state never silently activates the complete
Chatwork surface or makes a failed load executable. Only malformed serialized
content enters the explicit `config` repair flow: it starts from a clearly
marked replacement draft and changes nothing until Enter. Unsafe modes/objects
refuse the selector and require external filesystem repair; unavailable storage
must have access restored. The always-on read-only `doctor` reports invalid,
unsafe, or unavailable command-selection state without mutating it.
Complete-catalog static help is the sole read-only fallback: root and namespace
output identifies that state and points to `doctor`, exact help exposes only its
bounded invocation facts, and no fallback path dispatches a Chatwork task.

Terminal control is also phase-bounded. The infrastructure-only
`golang.org/x/term` adapter requires interactive stdin and stdout, owns raw
mode, alternate-screen entry, size lookup, and idempotent restoration, and
exposes no selection or authority policy. A non-TTY invocation fails before
persistence. Up/Down and Space modify only an in-memory draft. q, Escape, EOF,
terminal closure, or cancellation before Enter restores the terminal on the
bounded graceful path and leaves the prior preference unchanged. Enter
validates the complete active view and fixed-target request, restores the
terminal, and only then invokes the save action; restoration failure makes zero
save calls. After replacement is attempted, a raw failure is uncertain and
reconciles through `doctor`; confirmed success is not replaced by late
cancellation. Process termination that cannot run cleanup is not claimed to
restore terminal modes, but it still cannot persist a draft whose Enter save
boundary was never crossed.

The uncertain write fault exposes the candidate versioned SHA-256 fingerprint,
and `doctor` exposes the actual fingerprint of the ordered canonical enabled
paths together with its source. These bounded values and counts contain command
metadata only, never PAT or provider data, and let the read-only diagnostic
distinguish the state present after an uncertain write. Confirmed success does
not repeat this recovery identifier; it reports only the human-readable display
summary and any nonzero cleanup. Profiles from the preceding selector may
contain `doctor` or `version`; those two exact legacy entries are ignored for
active-view construction and removed on the next Enter save. No other always-on
or unknown path receives that migration exception.

### Processes

- Prefer in-process APIs.
- If a subprocess is required, allowlist the executable and fixed argument structure.
- Keep secrets out of argv and inherited environment unless explicitly justified.
- Bound output, time, and cancellation behavior.

### Network

- Declare allowed destinations and redirect behavior.
- Use timeouts, cancellation, and response-size limits.
- Validate protocol and host before sending credentials or user data.
- Treat remote names and content as unsafe terminal and machine context.

Every Chatwork HTTP request carries a bounded `User-Agent` value of the form
`cwk/<version>`. Infrastructure accepts exactly `dev` or a release-compatible
SemVer value without a leading `v` or build metadata, limited to 128 ASCII
bytes. Numeric version components and numeric prerelease identifiers reject
leading zeroes under SemVer rules. Invalid injected versions silently become
`cwk/dev` and are never echoed. The value contains no PAT, account, or other
credential data. The shared task execution path applies it to identity
preflight, reads, mutations, and uploads while preserving the fixed destination
and existing authentication and redirect policy.

For the first Chatwork implementation, metadata and ordinary provider requests have a 20-second timeout, file uploads have a 60-second timeout, and every logical operation has one transport attempt. Successful response bodies are capped at 8 MiB and provider error bodies at 64 KiB. The complete rendered result is capped at 16 MiB, aggregate lists at 10,000 items, and uploads at 5 MiB. The five provider-documented 100-item list operations retain their lower bound. Exceeding any limit fails closed without partial successful output; user configuration cannot raise a ceiling.

The shared HTTP transport also has a 20-second `ResponseHeaderTimeout` for every
request, including uploads. It starts after the request body has been sent and
bounds the wait for response headers. The per-task context is 20 seconds for
ordinary operations and 60 seconds for uploads; the HTTP client has a 60-second
outer timeout. Thus an upload has a 60-second overall ceiling, but it still
allows at most 20 seconds for response headers after its upload body is sent.
These are fixed implementation limits, not evidence that longer mutation
timeouts would improve delivery or that an uncertain write was not applied.

For a read, timeout wording is chosen only when `errors.Is(ctx.Err(),
context.DeadlineExceeded)` or `errors.Is(err, context.DeadlineExceeded)` is
true. The public `operation_canceled` kind/code/exit contract stays fixed, as
does cancellation wording for a caller cancellation. Ordinary I/O failure
remains `chatwork_unavailable`. A 403 may mention possible room non-membership
only for a selected room target or scope, using fixed text without provider
body content. Mutation outcomes retain the existing phase-aware taxonomy.

Message `--start-index` and `--count` are bounded local selection inputs, not
network controls. Values outside 1..100, duplicates, and malformed values fail
before authentication or I/O. Infrastructure rejects either field if it crosses
the application port, and a provider result above `source-limit=100` fails
before local selection can conceal it. A derived next start index carries no
credential, opaque provider state, or snapshot-stability claim.

The deprecated message `--limit` spelling is a catalog-declared alias of the
same typed `--count` input, not another authority or network control. It remains
for at least two minor releases and successful use writes only a migration
warning to stderr. Did-you-mean diagnostics never rewrite argv, choose a target,
or execute a suggestion; invalid input still fails before authentication/I/O.

Message `--contains` is a local read selector, never a network control or
authorization boundary. Its terms are exact case-sensitive substrings of typed
body text; they are not trimmed, normalized, parsed as Chatwork notation, or
forwarded to provider search. At most 100 nonempty valid UTF-8 terms of up to
1,024 bytes each are accepted. Invalid terms fail before authentication/I/O,
and any query text rendered in the result uses the terminal-safe quoted-text
projection. Infrastructure rejects a leaked body-term filter, while selection
retains the original provider bound so fewer displayed messages cannot claim
older-history or provider-filter completeness.

Message `--since`, `--until`, and `--on` are also local read selectors, never
network controls or authorization boundaries. Exact bounds require explicit
offsets and whole seconds; `--on` uses fixed Tokyo calendar rules and a single
injected clock observation rather than ambient host-local time. Invalid,
conflicting, or reversed periods fail before authentication/I/O. Infrastructure
rejects leaked period fields, and selection retains the original source bound
so a smaller output cannot claim older-history or provider-filter completeness.

The relation-fetch budget is application policy, not a provider list query or
pagination control. It is removed before infrastructure calls. One invocation
performs exactly one list request plus at most the declared number of exact
message reads, all through the same admitted authentication binding, fixed
Chatwork destination policy, caller context, timeout, response bounds, and
one-attempt operation contract. Source reuse consumes no fetch slot; the result
reports both the configured limit and actual attempts.

Rate-limit headers and error envelopes are untrusted external data. Chatwork
reset timing is accepted only as one bounded strict-decimal
`x-ratelimit-reset` Unix timestamp in the official five-minute window;
missing, duplicate, malformed, stale, or distant values produce unknown timing
instead of a guessed delay. `Retry-After` is not authoritative for this
provider. The adapter may compare the bounded message/task posting error body
with the one official room-limit sentence to select 10 seconds, but never
publishes that body. A delay carried by a non-retryable mutation fault is
display-only evidence: it cannot authorize automatic or agent-inferred replay,
and the one-attempt mutation contract still applies.

`messages reconcile-send` validates a body-file exactly once through the
infrastructure-owned bounded reader before any provider read. The submitted
file bytes stay in a client-, request-, and binding-bound opaque matcher that
is consumed once and zeroed whether matching completes or a later reply/list
read fails. As in the existing typed send/list task boundary, the application
receives the original request fields and typed provider messages; reconciliation
does not return file contents from the reader or return candidate bodies from
the matcher. The application receives only match booleans from the matcher and
drops temporary provider-body references after comparison. Results, faults,
and logs omit the submitted body and local path. Literal input uses the same
complete ordered recipient/reply prefixes from the send builder. The task
obtains the current account through one same-binding, same-context read and returns only
matches with that exact typed sender reference. An always-present `account_ref`
identifies the current account even with no candidates; every candidate's
`sender_ref` equals it. A failed or invalid account
lookup stops before the list read; it never assumes that the original send used
the same account. The result keeps `outcome=unknown` so content equality cannot
become proof that this particular invocation succeeded.

Machine-readable policy belongs in `.harness/project.json` or a project-specific typed manifest and is checked by `tools/repoguard` or a dedicated contract test.

## Output and terminal safety

Trusted CLI-authored human prose is Japanese. This does not relax structural
validation: Japanese catalog and TUI strings are validated and rendered as CLI
structure, while stable machine tokens remain ASCII. Localization must never
rewrite an opaque reference, credential, URL, provider response field, or
external Chatwork text. Translation is not sanitization and does not make
external text trustworthy.

Remote or file-derived text may contain terminal controls, format characters, Unicode line/paragraph separators, existing backslash escape-looking sequences, JSON-looking fragments, prompt-like prose, or excessive data. The template's visible projection escapes backslash first, then Unicode category C runes and U+2028/U+2029. Category C includes controls and format characters as well as private-use and unassigned code points; these are represented by visible escapes rather than removed. Therefore an actual newline remains distinguishable from the two input characters `\n`; JSON encoding applies its own structural escaping afterward. TSV delimiters and record newlines are added only by the renderer.

This projection protects terminal and TSV/JSON structure. It does **not** detect intent in printable text, remove prompt-like content, or prove that a language model will ignore it. Printable text such as `SYSTEM ...` or JSON-looking content remains semantically untrusted data. Agent consumers must keep data fields separate from instructions and apply their own trust policy; the CLI cannot claim semantic prompt-injection prevention.

The internal `AgentContract` validates this boundary as
`external_text_trust=untrusted_data`,
`external_text_projection=visible_escape`, and
`opaque_reference_policy=validated_exact_bytes`. Public help does not dump that
contract; successful task output retains its trust framing and runtime
structured faults retain their classification and recovery action.

Display-name discovery does not weaken this boundary. A `members find` query is
text, not an account identity; matching external member names remain untrusted
and visibly escaped, and all matches retain validated exact account references.
The discover command never authorizes or automatically invokes a reference-
consuming command, including when exactly one candidate is returned.

The same rule applies to `rooms list --query`: Unicode simple case folding is
application-only candidate matching over one complete typed provider result,
not identity normalization. No Unicode/whitespace normalization, trimming,
ranking, multi-rune expansion, hidden provider search, or unique-result
auto-selection is permitted. `members find` remains case-sensitive.

A derived project must decide:

- which output modes are stable;
- how control characters are escaped or rejected;
- maximum stdout and stderr budgets;
- whether partial output is ever allowed;
- how secrets and confidential fields are redacted;
- which stream carries errors versus data.

Do not let presentation sanitization change the identity used for authorization. Authorization uses validated domain values; display labels/content use a visible projection. Opaque references bypass that projection and retain the exact validated value.

For every presentation candidate, structure is CLI-authored and message text is external data. Its framing must prevent provider text from injecting candidate-specific records, fields, hierarchy, or completeness signals. Optimization must not remove the internally validated `external_text_trust: untrusted_data` meaning or its visible runtime framing. Hostile-text fixtures are shared across candidates so a format cannot improve token score by weakening structural safety.

Candidate C's historical context capsule used local aliases only inside one output document. They were never accepted by command parsers, mutation policy, audit identity, or infrastructure. The current `messages list` likewise uses actor aliases only to factor repeated display data inside one document. Every actor dictionary entry exposes its validated canonical account reference, every message record exposes its validated canonical message reference, and command inputs accept only those canonical values unchanged. Aliases never cross into mutation policy, audit identity, application, or infrastructure.

The current projection is also subtractive at the trust boundary: only catalog-declared task fields, exact canonical references, task-relevant bounds/completeness/uncertainty, and external-text trust framing may cross into the text contract. Provider/wire extras, display conveniences, and non-contract defaults are not promoted merely because they are available. Raw Chatwork notation may remain inside a declared untrusted message body, but presentation neither exposes it as separate semantic structure nor reparses it to create relationships. Removing fields must never remove the framing that distinguishes external data from CLI-authored structure.

## Supply-chain boundary

- Add a dependency only with a documented purpose and license review.
- Pin CI actions and security tools according to repository policy.
- Verify module integrity and known vulnerabilities in the security profile.
- Review generated changes and prove generation is reproducible.
- Build releases from reviewed source through the documented workflow.
- Decide signing and provenance explicitly; absence of signing is also a release-model decision.

### Go patch proposals

The scheduled Go patch updater treats release metadata as untrusted network
input. Production reads only the fixed HTTPS endpoint
`https://go.dev/dl/?mode=json`, with a 15-second request deadline and an 8 MiB
response limit. Redirects remain bound to that endpoint. Parsing rejects
duplicate JSON keys, unknown fields, trailing values, invalid versions, and
invalid release records. The selected candidate must be a numerically newer
stable patch in the exact major/minor line already named by `go.mod`; a
prerelease or major/minor change is ineligible. The updater has no source URL
or arbitrary file-path override and rewrites only the `go` directive.

The weekly and manual workflow runs only for the default branch. Preparation,
candidate validation, and proposal use the same immutable workflow SHA. The
`prepare` job has Contents read permission and no repository secrets. It uses
the Go version already declared by `go.mod` to inspect the official release
list and records the source SHA. If a candidate exists, it changes its local
`go.mod` directive and exports that candidate only after confirming the diff
is limited to that directive; with no candidate, it confirms the worktree is
unchanged. It does not run candidate gates. The separate `validate` job
also has Contents read permission and no repository secrets. It checks out
that same source SHA, installs the exact candidate Go toolchain, reapplies the
candidate directive, confirms the one-file diff, and runs the full, security,
and release profiles. Only after validation succeeds may the separate write
job proceed. That job checks the SHA and candidate again and requires the
sole diff to be the one-line `go.mod` directive change. It stages only
`go.mod` through the pinned pull-request action, with `GITHUB_TOKEN` limited
to Contents and Pull requests write. There is no PAT or expanded GitHub App
installation, no PR-head checkout or code execution, and no automatic merge.

GitHub may require approval before workflows for a `GITHUB_TOKEN`-created
pull request run. The candidate validation job proves the generated tree
passed its gates, but it does not create or impersonate head-attached status
checks. Branch protection that requires those checks still depends on the
normal approval and CI process. Dispatching the PR ref and using
`pull_request_target` are excluded because they would let PR-controlled
workflow code run in a privileged context.

### Shared Homebrew tap publication

Stable releases propose the already rendered, checksum-pinned, and strictly
audited `Formula/cwk.rb` to the public `tasuku43/homebrew-tap` repository. The
workflow never pushes directly to the tap's default branch and never stages a
Formula wildcard. Prereleases do not cross this write boundary.

The macOS Formula audit job has no App credential and uploads the strictly
audited Formula as a workflow artifact. A fresh Ubuntu publish job checks out
no source repository and executes no checked-out tagged source or Formula
content. It downloads that one-file artifact, validates its exact regular-file
identity as data, and only then exchanges GitHub-hosted repository secrets
`HOMEBREW_APP_ID` and `HOMEBREW_APP_KEY` for a short-lived installation token.
Token creation names owner `tasuku43` and only repository
`homebrew-tap`, and explicitly requests only Contents write and Pull requests
write as write-capable token permissions. Before a stable tag, the App
installation must be manually confirmed as restricted to that repository with
matching maximum permissions and absent from `cwk` and unrelated repositories.
The audit job's source-repository `GITHUB_TOKEN` stays Contents-read-only. The
token-bearing Formula publish job has no source-repository `GITHUB_TOKEN`
permissions; the separate GitHub Release publish job retains Contents write
only for release creation.

Post-publication Formula recovery is a separate, stable-tag-only entry to the
same workflow. Its preflight resolves the quoted input through the release-tag
validator, runs the canonical full gate at that immutable revision, and grants
the recovery job only Contents read. Recovery accepts only an existing
non-draft, non-prerelease Release whose six exact filenames and five checksums
verify. It uploads only the verified checksum file to the credential-free
Formula audit job; it has no Release write permission and does not execute
checked-out source. The later tap write still occurs only in the existing
fresh runner with the same App installation token restrictions.

Both source and tap checkouts disable persisted credentials and occur on
different runners. The installation token is supplied explicitly only to the
tap checkout and pinned pull-request
action, and it must not appear in Formula content, archives, logs, source, or
Git configuration left by checkout. The workflow and both Formula jobs admit
only their reviewed fields, so workflow- or job-level `env` and `defaults`
cannot inject runtime options into the token action. Before copying, the
publish job rejects a symbolic-link `Formula` directory, a symbolic-link
`Formula/cwk.rb`, and any existing non-regular target. The pull request targets
`main`, uses the tap automation's reviewed branch/title prefix, and changes
only `Formula/cwk.rb`. Tap merge remains a separate review/automation boundary;
a successful release job does not claim that the Formula has already merged.

Secret provisioning and App-installation review occur in GitHub settings
before the first stable tag. Secret values are never copied into a work packet
or release note. See [ADR 0004](decisions/0004-shared-homebrew-tap.md).

## Required negative tests

Every new side-effect class should include tests for:

- unknown or missing effect;
- incomplete or mismatched target;
- malformed input before adapter invocation;
- alternate, transformed, or ambiguous reference forms before adapter invocation;
- policy or authorization rejection with zero side-effect attempts;
- nil and typed-nil external ports with zero downstream calls;
- cancellation and timeout;
- adapter-entry and body-file-construction cancellation proving zero mutation `Do` calls, versus cancellation after `Do` remaining unknown;
- valid confirmed mutation success preserved when cancellation races after the adapter returns it, while read cancellation still suppresses its result;
- structured versus unclassified post-mutation outcomes, including a deadline cause that must not erase the adapter's stable classification;
- remote error without secret leakage;
- authentication or scope mismatch with zero downstream calls;
- incomplete pagination with no partial successful output;
- unsafe mutation retry or missing idempotency declaration;
- timeout or attempt configuration above the project's declared maximum;
- keyed retry that changes key within one logical operation or reuses a key for a different target or payload;
- oversized or hostile output;
- raw ESC/newline/bidi/zero-width/U+2028/U+2029 characters, existing backslashes, JSON-looking text, and prompt-like printable data across TSV, JSON, and stderr;
- actual control characters remaining distinguishable from pre-existing visible escape-looking text;
- repeated invocation when authorization must not be reused.
- exact disabled-command and catalog-alias invocation producing non-retryable
  `command_disabled` before handler, PAT resolution, or provider I/O, with exact
  `config` recovery; exact disabled scoped/trailing help has the same outcome,
  while unknown or incomplete non-namespace paths remain `unknown_command`;
- namespace help exposing disabled names only in a separate section, never
  usage/examples/workflows; disabled-only namespace root navigation; unsafe
  static-help fallback and unconfigured command-selection outcomes remaining
  distinct; re-enabling preserving the original authentication and mutation
  contracts;
- unknown-command suggestions using only the active catalog, and only the
  distance-nearest candidate when its effect is `read`; create/write/unknown
  nearest candidates suppress suggestions without promoting farther reads;
- malformed, oversized, duplicate-key, trailing-value, symbolic-link, or
  special-file command-selection state without a silent all-enabled execution
  fallback; complete-catalog static help remains read-only and cannot dispatch a
  configurable task;
- non-TTY rejection; fragmented CSI/SS3 keys; bounded terminal frames; and
  restoration across success, quit, EOF, cancellation, validation failure, and
  terminal failure;
- canceled, EOF, invalid, dependency-incomplete, or pre-Enter selection leaving
  the prior preference unchanged, and restoration failure making zero save
  calls;
- uncertain command-selection writes reconciling through the exact read-only
  `doctor` fingerprint rather than another mutation;
- legacy `doctor`/`version` selection entries normalizing without admitting
  another always-on path;
- textual effect badges remaining present without ANSI color, with hostile
  command text unable to create a control sequence;
- a re-enabled Chatwork mutation still requiring its original PAT, exact target,
  and typed confirmation policy.

## Derived-project threat-model pass

Before implementing a real integration, add a table covering:

| Task | Assets | Inputs | Effect | Target or impact | External boundary | Authorization | Stored data | Enforcement |
|---|---|---|---|---|---|---|---|---|

Then document credible abuse cases and limits. If a risk is accepted rather than eliminated, record the rationale in an ADR and state what evidence would trigger reconsideration.

## Limits

The template cannot protect a fully compromised developer machine, maintainer account, CI platform, or external service. It cannot infer whether copied source is legally publishable. It narrows accidental bypasses and makes review evidence repeatable; it does not replace independent review for high-impact systems.
