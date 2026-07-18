# Authentication Foundation

This document defines the reusable OAuth 2.0 and personal access token (PAT) boundary supplied by Chatwork CLI. The template fixes secret-free contracts and fail-closed application behavior. A derived project chooses the actual provider flow, credential source, storage, scopes, and account policy in its thesis and security model.

The governing dependency decision is [ADR 0001](decisions/0001-oauth-library-boundary.md): do not implement OAuth protocol machinery from scratch, and do not add an unused OAuth dependency to the template core.

## What the template guarantees

The base repository supplies:

- `authn.Method` for `oauth2` and `pat`, with an invalid zero value;
- `authn.Requirement`, which declares allowed methods, authority, audience, optional account binding, and project-defined required capabilities;
- `authn.Session`, which contains only non-secret identity and authorization metadata;
- an opaque, non-serialized `authn.BindingID` that correlates a validated session with one infrastructure-owned authentication record without exposing a credential or storage handle;
- exact requirement-to-session matching without normalization or URL parsing;
- expiry and capability checks;
- an application `authn.Gate` that authenticates once, revalidates the session, rechecks cancellation, and calls the downstream action once;
- stable `fault` classifications for missing authentication, expired authentication, context mismatch, insufficient capability, cancellation, and unclassified adapter failure;
- removal of upstream causes before structured failures cross the application boundary;
- tests proving that malformed declarations, authentication failure, cancellation, session mismatch, and capability failure cause zero downstream action calls.

The zero values of `Method`, `Requirement`, `BindingID`, `Session`, and `Gate` are not usable defaults. Missing declarations fail closed.

## Runtime boundary

```text
CLI composition root
        |
        | secret-free Requirement
        v
application authn.Gate
        |
        | Authenticate(Requirement)
        v
infrastructure Authenticator ----> credential source / OAuth library
        |
        | secret-free Session metadata
        v
application revalidation
        |
        | Session.BindingID, unchanged
        v
authenticated action ------------> task port ------------> infrastructure credential manager
                                                               |
                                                               | exact binding + freshness/refresh
                                                               v
                                                        provider request
```

The raw PAT, OAuth access token, refresh token, authorization header, authenticated HTTP client, callback verifier, and credential-store handle remain inside `internal/infra`. They are not fields of a domain or application type, arguments to a public command, return values from a use case, or values rendered by the CLI.

`Session` is metadata, not a bearer capability. The action receives a copy and passes its `BindingID` unchanged to the task-specific application port. The infrastructure implementation uses that opaque value as a process-local map correlation to resolve its private authentication record immediately before credential use. The binding representation is private, omitted from JSON, redacted by generic `fmt` diagnostics, and unsuitable as a provider identifier, token, persistent credential-store key, or public command value. Architecture lint rejects production calls to `NewBindingID` outside infrastructure, including dot-import bypasses, so application or CLI input cannot be promoted directly into a binding.

This closes accidental application wiring in which authentication validates account A but a task port silently uses a default credential for account B. It does not let the base gate inspect a provider token. The infrastructure authenticator and credential manager must prove in adapter tests that the issued binding record contains the same method, authority, audience, subject, account, capabilities, and expiry represented by the returned session.

## Secret-free contracts

### Method

The core recognizes two credential families:

| Method | Meaning | Not implied |
|---|---|---|
| `oauth2` | The infrastructure adapter established an OAuth 2.0 session | Grant type, browser/device flow, refresh, storage, or provider endpoints |
| `pat` | The infrastructure adapter established a PAT-backed session | Token source, lifetime, permission model, or account selection |

Adding another method requires a domain change, tests, security-model update, and compatibility review. Treating an unknown method as PAT or OAuth is forbidden.

### Requirement

`Requirement` is created from a reviewed command/use-case contract, not inferred from a received token. It declares:

- one or more allowed methods;
- a stable authority identifier;
- a stable audience identifier;
- an optional exact account binding;
- zero or more exact project-defined capabilities.

Authority and audience are public internal identifiers, not credential endpoints. Capabilities may represent OAuth scopes, PAT permissions, or higher-level project permissions, but the template does not define their names.

### Session

`Session` reports:

- the method actually used;
- the exact authority and audience;
- a stable subject identifier;
- an optional account identifier;
- granted project-defined capabilities;
- advertised expiry, when the credential has one;
- an ephemeral opaque binding to the infrastructure-owned authentication record, excluded from JSON and ordinary output.

A zero expiry means only that the adapter did not advertise expiry. It does not mean that the credential is permanent or safe to cache. A derived security model may require expiry for its selected method.

Do not render `Session` as normal command output. If a diagnostic command needs authentication information, expose a deliberately reduced projection such as method, status, and expiry class; never include raw provider responses or secret-bearing causes.

## Gate behavior

Before calling an authenticated action, `internal/app/authn.Gate` performs this order:

1. Reject a nil context, nil action, invalid requirement, missing authenticator, or missing clock.
2. Reject cancellation before credential resolution.
3. Ask the infrastructure authenticator for secret-free session metadata exactly once.
4. Sanitize unstructured authenticator errors and strip causes from valid structured faults.
5. Reject cancellation after credential resolution.
6. Validate the session metadata, including its ephemeral binding.
7. Match method, authority, audience, optional account, expiry, and every required capability exactly.
8. Reject cancellation immediately before the action.
9. Call the action exactly once with a session metadata copy.

Steps 1–8 produce zero downstream action calls on failure. The action is responsible for passing `Session.BindingID` unchanged to its task port. The port and credential manager honor cancellation after the action begins and classify provider failures as safe structured faults.

Exact matching is intentional. The gate does not trim, case-fold, decode, parse URLs from, or otherwise transform authority, audience, account, subject, or capability values.

## Failure and recovery contract

The base gate emits stable recovery classes:

| Condition | Fault kind | Stable code | Retry assumption |
|---|---|---|---|
| Gate context missing | `contract` | `missing_authentication_context` | No; repair the application wiring |
| Authenticated action missing | `contract` | `missing_authenticated_action` | No; configure the use case action |
| Authentication requirement invalid | `contract` | `invalid_authentication_requirement` | No; repair the catalog/use-case contract |
| Authenticator not configured | `authentication` | `missing_authenticator` | No; configure the project-selected method |
| Gate clock missing | `contract` | `missing_authentication_clock` | No; repair the application wiring |
| Credential resolution failed without a safe classification | `authentication` | `authentication_failed` | No automatic retry |
| Session metadata or ephemeral binding invalid | `authentication` | `invalid_authentication_session` | No; fix the adapter contract |
| Valid metadata could not be evaluated | `contract` | `authentication_evaluation_failed` | No; repair the gate/domain contract |
| Method, authority, audience, or account mismatch | `authentication` | `authentication_context_mismatch` | No; select the correct credential/account |
| Session expired | `authentication` | `authentication_expired` | No; reacquire according to project policy |
| Capability missing | `permission` | `insufficient_authentication_capability` | No; obtain the documented permission |
| Context canceled before the action | `canceled` | `authentication_canceled` | Only when the caller chooses a new attempt |
| Action returned an unstructured error | `internal` | `unclassified_authenticated_action_error` | No; the adapter must classify it |

A non-nil catalog `AgentContract.Authentication` binds the command to this template gate. Catalog validation therefore requires every stable code above with its exact kind and `retryable: false`; an omission or mismatch fails before dispatch. Provider-specific structured faults that the gate may pass through, such as refresh rejection, rate limiting, or temporary unavailability, remain additional derived-project declarations with command-valid recovery actions.

A derived CLI should attach concrete, command-valid next actions through its command/error catalog, for example a login command or a permission-inspection command. The template cannot name such a command before the product contract chooses whether one exists.

Rate limits, temporary identity-provider failures, and unsupported flows may pass through when an infrastructure adapter returns a valid, explicitly public structured fault. The gate removes its underlying cause because provider errors may contain authorization headers, request URLs, or credential material.

After the action begins, a valid structured adapter fault remains authoritative even if the caller context is also canceled. This preserves a known authentication, permission, unavailable, or unknown-outcome classification instead of replacing it with a less precise cancellation. An unstructured action error is mapped to cancellation when the caller context is canceled; otherwise it is collapsed to the internal fallback.

## Binding, expiry, and refresh at the I/O boundary

The gate's expiry check is an admission snapshot, not a lease over the subsequent provider request. The template does not choose an expiry headroom, refresh threshold, cache lifetime, or reuse policy because those values depend on the provider and operation budget. Every derived authenticated task follows this minimum shape:

1. Infrastructure creates an ephemeral `BindingID` independently of token, secret, provider identifier, and credential-store path bytes.
2. The authenticator stores the actual credential and its exact session metadata in an infrastructure-owned record, then returns only the secret-free session.
3. The use case passes `Session.BindingID` unchanged into its task port; it never selects an account by a global default after the gate.
4. The port resolves that binding immediately before I/O, verifies that the record still represents the validated authority, audience, subject/account, and capabilities, and checks expiry using the same caller context.
5. When the derived policy permits refresh, infrastructure refreshes inside that bound record. A refreshed identity or account mismatch is an authentication failure and makes zero provider task requests.
6. Refresh rejection is classified as authentication; insufficient provider permission is permission; a retryable identity-service outage may be unavailable. Causes and provider bodies remain private.
7. A missing, unknown, stale, or mismatched binding fails before the provider task request. A typed-nil task port fails at the use-case boundary through `portcheck.IsNil`.

The binding is process-local correlation metadata. Do not persist it, cache it across sessions, render it, log it, accept it from a user, or use it as proof of possession. Provider flow, storage, refresh locking, cache policy, and approval remain derived-project decisions.

## OAuth decision

OAuth protocol behavior is security-sensitive and provider-dependent. The template must not implement authorization URL construction, state or nonce validation, PKCE, callback handling, code exchange, token parsing, refresh, or authenticated transport itself.

When a derived project selects OAuth 2.0:

1. Record the grant and callback/device model in an accepted ADR.
2. Prefer `golang.org/x/oauth2` as the first candidate unless a provider requirement justifies another maintained implementation.
3. Add the dependency only to the derived project and import it only from `internal/infra`.
4. Pin the selected version and review its license, maintainers, transitive graph, release history, and vulnerability status.
5. Use the library for protocol machinery while retaining provider-specific validation and policy in the adapter.
6. Run `go mod verify`, `govulncheck`, dependency review, adapter tests, and the full one gate.

This accepts a bounded supply-chain dependency in preference to maintaining a private OAuth implementation. The dependency is optional so PAT-only projects do not inherit that risk or update burden.

A provider SDK is not the default. Use one only when an ADR demonstrates that provider-specific behavior materially reduces total security or maintenance risk and its dependency surface is acceptable.

## PAT decision

PAT support uses the same requirement, session, gate, fault, and adapter boundaries. A derived project still decides:

- how the PAT is supplied or acquired;
- whether an operating-system credential store, environment input, standard input, or another mechanism is acceptable;
- how authority, audience, subject, account, permissions, expiry, and revocation are verified;
- whether the PAT may be cached or reused;
- how the user replaces or revokes it.

Do not accept a PAT as a normal command-line flag: argv commonly reaches shell history, process inspection, logs, and agent transcripts. Never persist it in plaintext project configuration.

### Chatwork first implementation

Chatwork CLI supports one account through PAT and OAuth 2.0. Every provider API
task requires `CWK_AUTH_METHOD` to be exactly `pat` or `oauth2`. The selector is
secret-free but mandatory: missing, unknown, or unavailable selection fails
before a task request, and infrastructure never tries the other method as a
fallback.

For `pat`, infrastructure reads `CWK_API_TOKEN` from the command-process
environment, creates one ephemeral binding record, and sends the token only as
the `x-chatworktoken` header to the fixed production Chatwork API origin.
`cwk` never writes the token to disk. Environment delivery remains a deliberate
automation trade-off with environment-inspection and parent-process risks;
documentation recommends setting it only for the command process and clearing
inherited values.

For `oauth2`, the fixed profile is one public OAuth client using Authorization
Code Grant, exact state validation, and PKCE S256. The authorization endpoint is
`https://www.chatwork.com/packages/oauth2/login.php`; the token endpoint is
`https://oauth.chatwork.com/token`. The exact registered redirect URI must not
use `http` or `https`; the CLI does not start a loopback listener. The login task
shows one transient consent URL on stderr, then reads one complete redirected
URL from stdin. It rejects a changed redirect, missing/duplicate callback
field, state mismatch, authorization denial, or invalid code before persistence.

The OAuth public registration uses non-secret `CWK_OAUTH_CLIENT_ID` and
`CWK_OAUTH_REDIRECT_URI`; there is no client-secret input. The reviewed scope
set is `users.all:read rooms.all:read_write contacts.all:read_write`, the finite
set needed by the fixed API coverage goal. `offline_access` is not requested
because Chatwork reserves it for confidential clients. Infrastructure stores
access and refresh tokens only in the operating-system credential store and
uses `golang.org/x/oauth2` for authorization URL, PKCE, exchange, and refresh
machinery. Provider-advertised expiry remains binding metadata; refresh occurs
only inside the exact bound private record immediately before API I/O. A
refresh that changes identity/account, fails, or is canceled causes zero task
requests and never falls back to PAT.

The public authentication workflow is:

1. `auth profiles` discovers the one fixed `chatwork-oauth-profile` reference.
2. `auth login --profile <ref>` consumes it unchanged, refuses to overwrite an
   existing credential, and stores a validated OAuth credential.
3. `auth status --profile <ref>` performs no refresh or provider API call and
   returns only method, `unconfigured|ready|expired`, and advertised expiry.
4. `auth logout --profile <ref>` removes the exact local credential-store entry
   and reports `remote_revocation: false`; it does not claim remote revocation.

Login is a create within the discovered profile and logout is a destructive,
access-changing write to that exact profile. Their explicit task invocation is
sufficient; the Chatwork provider-mutation confirmation flags do not apply.
Both declare read-only `auth status` reconciliation for an unclassified local
mutation outcome. Authorization codes, state, PKCE verifiers, tokens, store
keys, and credential-bearing causes never enter argv, stdout, logs, snapshots,
fixtures, domain values, or application values.

## Decisions left to a derived project

The template intentionally does not fix these choices:

| Decision | Where to decide it |
|---|---|
| OAuth grant, browser/device behavior, redirect listener, callback URI, state, nonce, and PKCE policy | Selected above for the Chatwork first implementation; revise through a superseding product/security decision |
| Provider endpoints and client registration | Infrastructure configuration and security model |
| PAT input mechanism | Selected above: command-process environment only |
| Credential store and operating-system integration | Selected above for OAuth; exact dependency remains an infrastructure ADR and security review |
| Refresh, reuse, cache lifetime, logout, and revocation | Selected above for the fixed public profile; numeric refresh headroom remains in the infrastructure ADR |
| Concrete scopes, PAT permissions, and capability names | Selected above for the fixed API coverage goal |
| Account discovery and selection | One account and one OAuth profile in the first implementation |
| Whether a write requires human approval, reauthentication, or dry-run | Thesis, security model, and mutation policy |
| Authentication commands and recovery command names | Selected above and enforced by the command catalog |

These are not optional decisions. They are deliberately deferred until a real external API and user task make the tradeoffs concrete.

## Derived adapter requirements

An implementation of `app/authn.Authenticator` must:

- own every credential-bearing value and dependency in `internal/infra`;
- return only validated, secret-free `domain/authn.Session` metadata;
- issue a fresh non-secret binding independent of credential bytes and bind the returned metadata to that exact infrastructure record;
- respect context cancellation and project timeouts;
- map expected provider failures to stable, redaction-safe `fault.Error` values;
- avoid including provider response bodies, headers, URLs with query values, token claims, or secret values in public messages;
- avoid logging secrets at every log level;
- make account, audience, and capability mismatches observable without echoing their unsafe values.

Every authenticated task port must accept the opaque `authn.BindingID` supplied by the use case. Its infrastructure adapter resolves and revalidates that record immediately before credential use rather than consulting a process-wide default account. This is a structural application-port input, not a credential-bearing type crossing into application code.

The API adapter called after the gate must return structured faults for expected not-found, permission, rate-limit, unavailable, and contract failures. An unstructured error is intentionally collapsed to `unclassified_authenticated_action_error` so accidental provider prose is not exposed.

## Required verification in a derived project

Keep the template tests and add adapter-specific tests for:

- empty, malformed, expired, and revoked credentials;
- wrong authority, audience, subject/account, and permission set;
- two simultaneously available accounts/authorities/audiences, proving that a session for one cannot drive the other's task port record;
- missing, unknown, stale, mismatched, and cross-session binding IDs with zero provider task requests;
- callback state mismatch and PKCE failure for an applicable OAuth flow;
- cancellation during acquisition, exchange, refresh, and API execution;
- refresh failure and reuse behavior selected by the security model;
- expiry between gate admission and task-port I/O, including refresh success, refresh identity mismatch, refresh failure, and cancellation with zero unintended requests;
- credential-store unavailable and access-denied behavior;
- authentication failure with zero API calls;
- permission mismatch with zero API calls;
- typed-nil authenticator and task ports with zero acquisition or API calls;
- secret canaries absent from stdout, stderr, structured errors, logs, snapshots, fixtures, and test failure output;
- exact dependency and vulnerability checks after adding an OAuth library or credential-store module.

Use fake credentials and publishable provider fixtures. Never place a real token, client secret, private callback registration, internal endpoint, or account identifier in source, history, CI, release artifacts, or examples.
