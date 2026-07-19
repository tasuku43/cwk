# External API Contracts

This document defines the small set of cross-project contracts supplied for API-backed CLIs. It does not turn an upstream API into the public CLI and does not provide a universal HTTP client. The product remains a set of user outcomes; transport is an implementation detail behind those outcomes.

## Boundary: template decisions versus derived decisions

| Area | Fixed by this template | Decided by the derived project |
|---|---|---|
| Public surface | Commands describe user outcomes; catalog metadata is complete and machine-readable | Which outcomes, commands, aliases, and compatibility promises exist |
| Discovery and action | Discovery emits opaque references; action consumes an exact reference without rediscovery or transformation | Reference kinds, filters, ambiguity rules, and task-specific workflows |
| Authentication | PAT-only secret-free requirements and session metadata, a fail-closed application gate, an ephemeral binding issued only by infrastructure and passed unchanged through task ports, exact record revalidation before I/O, typed failures, and zero downstream calls on rejection | Any future method, persistence, expiry, refresh/cache, tenant/account selection, login UX, or revocation requires a new product and security decision |
| Future OAuth implementation | OAuth is not supplied by the current core; a future proposal must keep reviewed protocol machinery behind infrastructure | Whether OAuth is justified and which reviewed adapter/library is acceptable; see [ADR 0001](decisions/0001-oauth-library-boundary.md) |
| Effects | `read`, `create`, or `write`; a create binds one opaque parent/scope input, a write binds one matching opaque existing-target input plus optional parent, generic impact is explicit, and policy is injected at one application boundary | Confirmation, approval, dry-run, OS authentication, authorization reuse, and domain-specific impact |
| Pagination | Opaque cursor envelope, explicit budgets, loop detection, cancellation, complete-or-no-result traversal, and a JSON-only public-page contract with a top-level completion cursor | Exhaustive versus public paged behavior, page size, ordering/snapshot semantics, limits, and user overrides |
| Calls | Finite timeout, attempt count, and upstream idempotency are explicit; unsafe mutation retry is rejected | Vendor error classification, retry/backoff budget, idempotency-key support, and endpoint-specific timeouts |
| Failures | Stable kind/code/retryability/next-action model | Product-specific codes and commands that resolve a failure |
| Schemas | Wire DTOs stay in infrastructure; drift is tested with publishable fixtures | Schema source, update cadence, unknown-field policy, compatibility window, and fixture license |
| Capabilities | Public catalog entries are finite and validated; unsupported work is recorded rather than exposed accidentally | Upstream coverage, deferred/internal capabilities, owners, rollout order, and explicit non-goals |
| Output | Declared format, fields, types, completeness, terminal escaping, and contract tests | Which human and machine formats are stable and the bounded size/streaming policy |
| Release | One gate, public-boundary checks, byte-for-byte reproducible archives for identical pinned inputs, checksums, and immutable release intent | Supported platforms, signing/provenance, package managers, cadence, and long-term support |

The template side of this table fixes vocabulary, validation, and enforcement points; it does not silently choose the derived-side settings. The derived side is not a gap: it is where the product thesis and security model must become concrete before the corresponding live capability is enabled.

## Authentication and credential flow

Read [Authentication](07_authentication.md) before adding a network adapter. Application code receives only a validated, secret-free session description and passes its non-serialized ephemeral binding unchanged into the task port. Infrastructure resolves that binding and revalidates the exact private authentication record at I/O time. The PAT and authorization header remain inside infrastructure. Other credential methods are future work rather than dormant current-core behavior.

Authentication is a precondition, not a transport error to discover after a write. A failed or mismatched requirement must produce zero downstream API calls. Authentication and permission are different failure kinds: reauthentication is not presented as a remedy for a valid identity that lacks authorization.

A non-nil catalog authentication requirement means the command uses the template application gate. The catalog must declare the gate's complete standard fault set with exact code, kind, retryability, and command-valid recovery actions; validation rejects omissions before dispatch. Provider-specific authentication, rate-limit, unavailable, or unsupported faults are additional derived-project declarations rather than replacements for that base set.

### Chatwork PAT-only contract

Every fixed Chatwork API-task requirement admits exactly `pat` and identifies
`CWK_API_TOKEN` as its sole credential input. No selector is required because
there is no second method. Missing or malformed token input fails during
process-local adapter construction and makes zero provider task requests.

Infrastructure retains the token behind the exact ephemeral binding admitted
by the application gate, resolves that binding immediately before I/O, and
attaches the token only as `x-chatworktoken` to the fixed Chatwork API origin.
The CLI accepts no token argv, persistent configuration, credential-store,
login, status, logout, callback, browser, profile, or client-registration
surface. `CWK_AUTH_METHOD` is not a public input, and its ambient value cannot
change adapter selection or recovery.

The scoped catalog contract declares the required PAT method and token
environment prerequisite. `chatwork_token_missing` and
`chatwork_token_invalid` recover through exact scoped help; no error points to
an authentication lifecycle command. Synthetic adapter tests pin the exact
header/destination, token redaction, binding isolation, and zero-call failure
behavior.

## Pagination and completeness

`domain/page` defines a one-page envelope with an opaque cursor. `app/pagination.Drain` owns exhaustive traversal and requires explicit page, item, and page-size budgets.

An exhaustive command follows this contract:

1. Forward each cursor byte-for-byte; never decode, trim, reconstruct, or expose a resource URL as a replacement.
2. Stop only when the adapter returns an empty next cursor.
3. Detect repeated cursors and finite-budget exhaustion.
4. Honor the same cancellation context on every page.
5. Reject a page containing more items than the requested page size; the page size is a per-response memory bound, not only an upstream hint.
6. Validate every page and domain item before presentation.
7. Return the complete result or an error with no partial result.

`complete` and `paged` are different public contracts:

- `complete` means the command owns exhaustive traversal. It uses the bounded drain behavior above, exposes no pagination binding, and returns the whole declared result or no result.
- `paged` means one successful invocation returns one complete public page, not an exhaustive collection. Its `AgentContract.Pagination` binds exactly one optional cursor argument or flag to exactly one top-level string cursor field. That field is always emitted beside `schema_version` and the collection envelope; `CommandOutput.Fields` continue to describe only items inside the envelope. Both cursor endpoints carry the same dedicated opaque reference kind, and no other command, input, or output may use that cursor kind. The typed `completion: "empty_cursor"` rule makes the empty string the only completion marker; omission, JSON `null`, and a non-string value are contract failures, not completion.

Paged commands support only JSON and use JSON as their default. This keeps every successful presentation self-describing and prevents a text or TSV page without a completion marker from looking exhaustive. Catalog validation rejects a missing paged binding, a binding on complete output, any other output format, a required or non-CLI cursor input, an invalid or colliding top-level cursor field, a missing or unknown completion rule, non-opaque cursors, kind mismatch, and extra cursor candidates. Renderer fixture checks require the top-level cursor to be present and string-typed. Agent help projects the binding with the input/output contracts and derives a same-command continuation workflow, so an agent passes the emitted cursor bytes back without trimming, decoding, or guessing. A declared page is not an incomplete successful output; silently truncating that page, omitting its continuation cursor, or reaching a local limit without a cursor is a contract failure.

A finite application-owned projection over one already complete bounded task
result is not provider pagination. It may expose an explicit selection limit
only when output also retains the upstream source bound and selection provenance;
it must not imply that another page exists or manufacture a cursor.

## Timeout, retry, and idempotency

`domain/apicall.Policy` is declared per adapter operation:

- `Timeout` is finite.
- `MaxAttempts` includes the initial call and is at least one.
- `Idempotency` is `safe`, `keyed`, or `unsafe`; the zero value is invalid.
- A keyed operation has one opaque key per logical operation and reuses it across transport attempts.
- A mutation with more than one attempt is valid only when the upstream operation is safe or keyed.

The application mutation invoker calls its action once. Any proven-safe transport retry happens inside the adapter and does not repeat policy, confirmation, or logical intent construction. An adapter may retry only typed retryable failures, must respect `Retry-After` when applicable, and must not sleep past context cancellation or its overall budget.

Read-only application services recheck cancellation immediately after a port returns and suppress the result when a port ignored cancellation. Because exhaustive pagination returns no partial result, every `operation_canceled` path in its drain is retryable and matches the catalog's common read cancellation contract.

Mutation semantics are phase-sensitive. Before the action call, `execution.Invoker` guarantees zero mutation attempts, so its common `operation_canceled` fault is retryable. Once the action is called, cancellation does not prove that the provider rejected or rolled back the effect. A valid structured adapter fault is authoritative even when its private cause is `context.Canceled` or `context.DeadlineExceeded`; the invoker returns a detached `fault.PublicCopy` and preserves its kind, code, and retryability. Any other action error, including a raw cancellation, becomes non-retryable `contract/unclassified_mutation_outcome` because the invoker cannot infer whether the effect occurred. That common fault must point only to an exact read-only reconciliation command. A nil action error is a confirmed success and is not overwritten by cancellation observed after confirmation.

The adapter contract must distinguish a request that was not sent, a confirmed result, and an unknown outcome when the provider makes that distinction possible. An unknown mutation outcome is non-retryable by default and points to an exact read/discover command that reconciles the target before another write. Do not translate cancellation into permission to repeat an unsafe action.

The template does not select a backoff formula or universal numeric ceiling because vendor limits and latency budgets differ. A derived security/product contract records the maximum accepted timeout and attempt count, formula, jitter source, caps, and tests; user configuration above those bounds must fail rather than create an effectively unbounded call.

Chatwork's first implementation chooses no automatic transport retry: every read and mutation has `MaxAttempts: 1`. Metadata, reads, and non-upload mutations have a 20-second request timeout; upload has 60 seconds. A successful provider body is limited to 8 MiB and an error body to 64 KiB. The application/CLI boundary limits a complete output to 16 MiB and an aggregate list to 10,000 items; file input is limited to 5 MiB. `GET /my/tasks`, `GET /rooms/{room_id}/messages`, `GET /rooms/{room_id}/tasks`, `GET /rooms/{room_id}/files`, and `GET /incoming_requests` retain the provider's documented maximum of 100 items instead of using the larger aggregate ceiling. Crossing any bound yields no partial successful result; in particular, a message response above its declared coverage fails before local selection can hide it.

CLI task interpretation maps an omitted window or explicit
`messages list --window recent` to `ForceRecent=true`; explicit
`--window changes` maps to false.
Infrastructure therefore sends `force=1` for the normal latest bounded window
and omits `force` only for the explicitly requested provider differential
window. Neither mode claims complete room history.

`messages list --sender`, `--limit`, and `--context` are application-owned
selection inputs over that single bounded message response. The optional limit
accepts 1 through 100 primary messages. Exact-sender OR matching runs first;
typed `send_time` then chooses the newest N candidates, with later provider
position breaking equal-time ties; direct typed reply context runs last and may
increase displayed count beyond N. Membership selection does not reorder the
provider records or their original sequences. These inputs never become
Chatwork query parameters, trigger a second request, or fetch a referenced
message outside the returned window. Adapter request construction rejects them
if they cross the application port boundary, and the one provider request
continues to use only the documented `force` query. There is no cursor, offset,
or pagination. The provider's 100-message ceiling is exposed separately as
`source-limit`; the requested limit and pre-limit candidate count belong to
selection provenance. Invalid limit values fail before authentication or
provider I/O.

For keyed mutation retry, create one key only after the complete logical intent and payload have been validated. Reuse that key for transport attempts of the same logical operation, never reuse it for a different target or payload, and never regenerate it merely because the transport result is uncertain. Adapter tests must prove same-operation reuse and cross-operation separation; `apicall.Policy` validates the generic declaration but cannot infer provider-specific key binding.

## Side-effect and impact boundary

Every command has an `operation.Effect`. A mutation also declares:

- a canonical `TargetRef`;
- a catalog `MutationContract` that binds its structured opaque inputs to the target roles;
- impact cardinality: one, many, or unbounded;
- whether it sends notifications;
- whether it changes access;
- whether it is destructive.

Each impact dimension uses an explicit declaration; omitted values fail closed. Product-specific effects such as message recipients, visibility transitions, file sharing, or workflow triggers belong to a derived domain type and may make the policy stricter.

The binding rules distinguish an object that does not exist yet from an existing object being changed. A reference-bound `create` declares exactly one `parent_input`, consumes that input as an opaque parent or scope reference, and declares no `target_id_input`. A reference-bound `write` declares `target_id_input` as an opaque reference whose kind equals `TargetKind`; it may also declare a distinct opaque `parent_input`. `target_inputs` contains exactly those named roles. A command-bound local singleton instead declares a fixed target with matching kind, explicit empty `target_inputs`, and no input roles; the effect determines create scope versus write target. Missing roles, extra or duplicate inputs, non-reference inputs, invalid fixed scope, and target-kind mismatches fail catalog validation before any mutation policy or adapter call.

`app/execution.Invoker` snapshots and validates command, effect, target, and impact; applies an injected policy; checks cancellation; then calls one logical mutation action. It deliberately does not decide whether policy means human approval, dry-run, OS authentication, role authorization, or another mechanism.

For Chatwork, the injected policy has three finite decisions. Ordinary creates and updates need no extra flag after exact references, payload, effect, target, and impact validate. Room creation, room-member replacement, invite-link creation/update, and incoming-request acceptance require exact `--confirm=access-change`. Room leave/delete, message deletion, invite-link deletion, and incoming-request rejection require exact `--confirm=destructive`. The flag is invocation-local typed policy input, not reusable approval. Failure to supply the required exact value makes zero provider calls.

All Chatwork mutations are unsafe for automatic retry under this first contract because the provider snapshot supplies no CLI-owned idempotency guarantee. An unknown post-send outcome is non-retryable and its catalog fault names a read-only reconciliation command. That command may inspect the target or parent scope but cannot call a create/write task.

## Failure and recovery contract

`domain/fault.Error` provides stable recovery metadata:

- `kind`: broad recovery class;
- `code`: stable project-specific identifier;
- `retryable`: whether repeating the same logical command can be correct;
- optional `retry_after`;
- `next_actions`: exact commands that can resolve or investigate the failure;
- a human message that is useful but not required for machine classification.

The common kinds cover invalid input, authentication, permission, not found, ambiguity, policy rejection, rate limiting, temporary unavailability, cancellation, unsupported capability, contract failure, and internal failure. An upstream error is mapped once at the infrastructure/application boundary. `fault.PublicCopy` extracts and validates a structured fault while discarding outer wrappers and its private cause; public boundaries use that helper before testing generic cancellation, so a deadline cause cannot erase a more precise valid classification. A malformed typed fault remains a contract failure. Raw upstream bodies and credential-bearing errors are never public output.

## Wire schemas and drift

An API adapter owns wire DTOs and maps them into domain values. Do not reuse SDK or generated wire types as public output or application input.

For every remotely decoded shape, commit the smallest legally publishable fixtures that exercise:

- a minimal valid response;
- additional unknown fields;
- required-field absence or null;
- unknown enum values;
- malformed or oversized content;
- a representative error envelope.

Record fixture provenance, schema/version, checksum, license, and whether it was synthesized. A generator is pinned, deterministic, and unable to register a public command or relax an effect automatically. Schema drift fails a contract test and becomes a reviewed product/security decision when it changes capability, output, or impact.

## Chatwork semantic-output boundary

Chatwork message responses cross two distinct compatibility boundaries:

1. infrastructure validates the reviewed provider wire shape and parses supported Chatwork notation into typed semantic facts;
2. a presentation candidate projects those facts without changing the semantic answer, canonical identity, bounds, or trust classification.

Neither boundary exposes raw upstream JSON as the product model. A wire-field addition does not automatically become public output, and a notation parser update cannot silently create a new public relationship type. A declared message body may contain raw Chatwork notation as untrusted external text, but presentation does not publish that notation as a separate semantic record or reinterpret it.

The first message-context semantic fixture must declare:

- the exact room reference producer and consumer;
- whether the upstream request returns a latest window, a differential window, or another bounded snapshot;
- the maximum messages, bytes, and provider calls;
- stable ordering and duplicate handling;
- explicit To, reply, and quote mapping rules;
- behavior when a reply parent is outside the returned window;
- task-relevant completeness, coverage, uncertainty, and canonical references;
- rate-limit, authentication, permission, malformed-notation, response-bound, and incomplete-context failures.

An internally complete semantic result over a partial upstream window is still partial room context. Every eligible presentation must make that answer recoverable. A zero exit status means the declared bounded result was produced completely; it does not claim that all room history was retrieved.

A supported evaluation outcome must consume each candidate directly. If its acceptance transcript uses `jq`, a custom join, Chatwork-tag parsing, or an undocumented follow-up request, that candidate is ineligible or the capability's stated outcome is too broad. Candidate C (`cwk-context-capsule/1`) remains the tested first-stable baseline. A P-derived task projection (`cwk-task-projection/1`) was selected through an explicit owner compatibility decision after Competition 1 was inconclusive and hardened beyond the frozen candidate, not as its benchmark winner. The current default is its further reviewed headerless subtraction.

The current projection is subtractive: it publishes only catalog-declared task fields, exact canonical references, task-relevant bounds/completeness/uncertainty, and external-text trust framing. It defines no in-band schema/task preamble or display aliases and does not add a standalone provider coverage record, provider/wire fields, raw notation as derived semantics, empty optional shells, or helpful non-contract defaults. Moving from C to the P-derived projection and then to the headerless default are breaking pre-1.0 text migrations; semantic task contracts and canonical-reference identity remain governed independently. Competition raw runs, scoring defects, and audit findings remain evidence and cannot be rewritten to manufacture a winner claim.

## Capability and coverage discipline

The command catalog is the only registry of public commands. Do not create a second dispatcher from an OpenAPI document, SDK, or capability ledger.

The first complete Chatwork implementation maintains a fixed upstream operation snapshot for coverage evidence. It is not a dispatcher or public command list: entries contain reviewed method/path identity and capability ownership only. Each capability still has one ledger status:

- `public`: linked to exactly one catalog task or one documented composed workflow;
- `internal`: required by an implementation but not a user task;
- `deferred`: deliberately unsupported, with a reason or prerequisite;
- `excluded`: outside the thesis or security boundary.

Generation may update evidence about upstream operations, but it cannot promote a capability to `public`, select an effect, or invent an impact declaration. Those are reviewed product decisions.

For the 2026-07-18 Chatwork snapshot, completion means exactly 32 documented REST operations map to public task capabilities and every Chatwork-backed public capability maps back to at least one of those operations. A later provider operation is outside this work until a reviewed snapshot update changes the finite coverage universe.

## Adapter completion checklist

Before an external adapter is complete, prove:

1. Required authentication is declared and no secret crosses into application/domain/output.
2. The same context reaches every call; canceled reads emit no result, and mutation outcome uncertainty never enables an unsafe retry.
3. Timeout, response-size, pagination, and traversal budgets are finite.
4. Retryability and idempotency are explicit; unsafe mutation retry fails validation.
5. Wire fixtures cover drift and hostile data; terminal projection escapes control characters.
6. Every failure maps to a stable kind/code and useful next action.
7. Discovery returns canonical opaque references and action forwards them unchanged.
8. Missing or mismatched mutation bindings, malformed runtime targets, policy denial, auth failure, and cancellation each make zero mutation attempts.
9. Success output matches its declared schema and is emitted only when complete.
10. `task check`, `task security`, and any adapter-specific contract test pass.
