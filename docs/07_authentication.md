# Authentication Foundation

This document defines the PAT-only authentication boundary used by Chatwork
CLI. `CWK_API_TOKEN` is the sole credential input for the current product. The
repository supplies no OAuth method, protocol adapter, callback workflow,
credential store, or public authentication-lifecycle command.

The governing product decision is [ADR 0003](decisions/0003-chatwork-pat-only.md).
[ADR 0001](decisions/0001-oauth-library-boundary.md) remains historical design
guidance for a future proposal: if OAuth is reconsidered, do not implement its
protocol machinery locally. It does not describe a capability supplied by the
current core.

## What the repository guarantees

The repository supplies:

- `authn.MethodPAT`, with an invalid zero value;
- `authn.Requirement`, which declares the exact PAT method, authority,
  audience, optional account binding, and required capabilities;
- `authn.Session`, which contains only non-secret identity and authorization
  metadata;
- an opaque, non-serialized `authn.BindingID` that correlates a validated
  session with one infrastructure-owned token record without exposing the
  token;
- exact requirement-to-session matching without normalization or URL parsing;
- capability checks and an application `authn.Gate` that authenticates once,
  revalidates the session, rechecks cancellation, and calls the downstream
  action once;
- stable faults for missing authentication, context mismatch, insufficient
  capability, cancellation, and unclassified adapter failure;
- removal of private causes before structured failures cross the application
  boundary;
- tests proving that malformed declarations, token failure, cancellation,
  session mismatch, and capability failure cause zero downstream action calls.

The zero values of `Method`, `Requirement`, `BindingID`, `Session`, and `Gate`
are not usable defaults. Missing declarations fail closed.

## Runtime boundary

```text
CLI composition root
        |
        | PAT-only secret-free Requirement
        v
application authn.Gate
        |
        | Authenticate(Requirement)
        v
infrastructure Chatwork adapter ----> command-process CWK_API_TOKEN
        |
        | secret-free Session metadata + ephemeral BindingID
        v
application action -----------------> task port
                                         |
                                         | unchanged BindingID
                                         v
                                infrastructure binding lookup
                                         |
                                         | x-chatworktoken
                                         v
                                fixed Chatwork API destination
```

The API token and authorization header remain inside `internal/infra`. They are
not fields of a domain or application type, arguments to a public command,
return values from a use case, or values rendered by the CLI.

`Session` is metadata, not a bearer capability. The action receives a copy and
passes its `BindingID` unchanged to the task-specific application port. The
infrastructure adapter uses that opaque value as process-local correlation to
resolve the exact private token record immediately before credential use. The
binding is omitted from JSON, redacted by generic `fmt` diagnostics, and is not
a provider identifier, persistent key, or public command value. Architecture
lint rejects production calls to `NewBindingID` outside infrastructure.

This prevents application wiring in which authentication validates one record
but a task port silently uses another. Adapter tests prove that a binding is
usable only by the client and process-local record that issued it.

## Secret-free contracts

### Method

The current domain recognizes one credential method:

| Method | Meaning | Not implied |
|---|---|---|
| `pat` | Infrastructure established a Chatwork API-token session | Persistence, provider-side expiry, multiple accounts, or user selection |

Adding OAuth or another method requires a domain change, thesis and product
revision, security review, superseding ADR, catalog changes, migration plan,
dependency review, and tests. Unknown methods never become PAT implicitly.

### Requirement

`Requirement` is created from the reviewed command/use-case contract, not
inferred from a received token. Every current Chatwork task declares exactly:

- method `pat`;
- authority `chatwork`;
- audience `chatwork-api-v2`;
- the reviewed Chatwork API capability.

Authority, audience, and capability are public internal identifiers, not
credential endpoints or secret values.

### Session

`Session` reports only the admitted method, authority, audience, stable
subject/account metadata when available, granted capabilities, optional
advertised expiry, and the ephemeral infrastructure binding. The binding is
excluded from JSON and ordinary output.

The current PAT adapter advertises no credential expiry. A zero expiry means
only that the adapter has no expiry metadata; it does not claim that the token
cannot be revoked or replaced. `cwk` exposes no authentication status command.

## Gate behavior

Before calling an authenticated action, `internal/app/authn.Gate` performs this
order:

1. Reject a nil context, nil action, invalid requirement, missing
   authenticator, or missing clock.
2. Reject cancellation before credential resolution.
3. Ask infrastructure for secret-free session metadata exactly once.
4. Sanitize unstructured authenticator errors and strip causes from valid
   structured faults.
5. Reject cancellation after credential resolution.
6. Validate the session metadata and ephemeral binding.
7. Match method, authority, audience, optional account, expiry metadata, and
   every required capability exactly.
8. Reject cancellation immediately before the action.
9. Call the action exactly once with a session metadata copy.

Steps 1–8 produce zero downstream action calls on failure. The action passes
`Session.BindingID` unchanged to its task port. Exact matching is intentional:
the gate does not trim, case-fold, decode, or parse URLs from authentication
metadata.

## Failure and recovery contract

The gate emits the following stable recovery classes:

| Condition | Fault kind | Stable code | Retry assumption |
|---|---|---|---|
| Gate context missing | `contract` | `missing_authentication_context` | Repair application wiring |
| Authenticated action missing | `contract` | `missing_authenticated_action` | Configure the use-case action |
| Requirement invalid | `contract` | `invalid_authentication_requirement` | Repair the catalog/use-case contract |
| Authenticator missing | `authentication` | `missing_authenticator` | Configure the PAT adapter |
| Gate clock missing | `contract` | `missing_authentication_clock` | Repair application wiring |
| Credential resolution failed without a safe classification | `authentication` | `authentication_failed` | No automatic retry |
| Session metadata or binding invalid | `authentication` | `invalid_authentication_session` | Repair the adapter contract |
| Metadata could not be evaluated | `contract` | `authentication_evaluation_failed` | Repair the gate/domain contract |
| Method, authority, audience, or account mismatch | `authentication` | `authentication_context_mismatch` | Supply the correct token/context |
| Session expired | `authentication` | `authentication_expired` | Replace the credential according to product policy |
| Capability missing | `permission` | `insufficient_authentication_capability` | Use a permitted Chatwork account |
| Context canceled before the action | `canceled` | `authentication_canceled` | Retry only when the caller chooses |
| Action returned an unstructured error | `internal` | `unclassified_authenticated_action_error` | Adapter classification must be repaired |

A non-nil catalog authentication requirement binds the command to this gate.
Catalog validation requires every stable gate code with the exact kind and
retryability. Chatwork-specific `chatwork_token_missing` and
`chatwork_token_invalid` faults additionally point to the exact scoped help for
the task; no recovery points to a login or authentication namespace.

After the action begins, a valid structured adapter fault remains authoritative
even when the context is canceled. An unstructured mutation error cannot prove
rollback and follows the mutation outcome contract instead of authorizing a
retry.

## Binding and credential use at the I/O boundary

Every authenticated Chatwork task follows this shape:

1. Infrastructure reads and validates `CWK_API_TOKEN` during process-local
   adapter construction.
2. Infrastructure creates an independent ephemeral `BindingID`, stores the
   token only in its private in-memory record, and returns a secret-free
   session.
3. The use case passes `Session.BindingID` unchanged into its task port.
4. The port resolves that binding immediately before I/O and verifies that the
   record still represents the admitted method, authority, audience, and
   capabilities.
5. A missing, unknown, stale, wrong-client, or mismatched binding fails before
   a provider task request.
6. The adapter attaches the token only as `x-chatworktoken` to the fixed
   production destination and follows no credential-bearing redirect.

`rooms create` additionally binds `Requirement.AccountID` to the exact
`--account` reference. After access-change confirmation and before the task
port runs, the authenticator uses the newly created private binding for one
`GET /me`, requires its canonical `account_id` to match, updates the
secret-free session's subject/account metadata, and lets the gate revalidate
that exact match. Failure removes the provisional record and sends no
`POST /rooms`. The transport repeats the exact stored-account/request-account
check before constructing the POST, preventing a generic binding from
bypassing the gate. This is credential verification, not account/profile selection,
owner assignment, or permission inference from the requested administrator
list. The extra read uses the same fixed destination, timeout, response bounds,
rate evidence, and cancellation context.

The binding is process-local correlation metadata. Do not persist it, cache it
across sessions, render it, log it, accept it from a user, or use it as proof of
possession.

## Chatwork PAT decision

`CWK_API_TOKEN` is the sole credential input. Infrastructure reads it from the
command-process environment and `cwk` does not write it to disk. Environment
delivery remains a deliberate automation trade-off: parent processes and
same-user inspection may expose environment values, so users should inject the
token only into commands that need it and clear inherited values afterward.

The CLI does not accept the token through a normal flag or positional
argument, stdin data payload, XDG/AppData file, project configuration, browser
flow, or operating-system credential store. It exposes no `auth login`, `auth
status`, `auth logout`, profile discovery, callback, client registration, or
method-selection command. `CWK_AUTH_METHOD` has no product meaning.

Production sends the token only to `https://api.chatwork.com/v2` in the
`x-chatworktoken` header. Public base-URL overrides are forbidden. Tests inject
synthetic tokens and local servers through internal construction and never read
a developer token or contact live Chatwork.

## Deferred OAuth decision

OAuth is not a current domain method or product capability. A future proposal
must first revise the thesis, product contract, architecture, security model,
catalog, and migration story. It must then supersede ADR 0003 and follow ADR
0001's rule to use a reviewed infrastructure library rather than implementing
authorization URLs, state, PKCE, callback handling, exchange, refresh, or token
storage locally. No dormant dependency or undocumented fallback is retained in
the current product.

## Decisions left for future work

| Decision | Current status |
|---|---|
| PAT persistence or OS-protected storage | Unsupported; requires a separate security and UX decision |
| Multiple accounts or account selection | Unsupported; one token determines one command-process account |
| OAuth or another method | Deferred; requires a full product/domain/ADR change |
| Provider-side token acquisition or revocation | Outside `cwk`; follow Chatwork account administration |
| Token expiry or replacement automation | Unsupported; replace the environment value explicitly |
| Human approval or dry-run for mutations | Governed separately by the mutation policy |

## Verification

The PAT boundary is complete only when tests prove:

- every public Chatwork requirement contains exactly `pat`;
- `CWK_API_TOKEN` is the sole required environment credential;
- missing and malformed tokens make zero provider task calls;
- obsolete method-selector values cannot select another adapter;
- the token is attached only to the fixed `x-chatworktoken` request header;
- the ephemeral binding is passed unchanged and rejected across clients or
  sessions;
- token canaries never reach argv, stdout, stderr, structured faults, logs,
  snapshots, fixtures, persistent configuration, or repository history;
- root/scoped help exposes no authentication lifecycle command and gives exact
  scoped recovery for token failure;
- `task check`, `task security`, and `task public:check` pass.
