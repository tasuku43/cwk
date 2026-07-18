# ADR 0003: Use the Chatwork API token as the sole credential

- Status: Accepted
- Date: 2026-07-18
- Deciders: Chatwork CLI maintainers
- Scope: Chatwork authentication, public catalog, infrastructure dependencies,
  credential handling, and agent readiness
- Supersedes: [ADR 0002](0002-chatwork-oauth-public-client.md)
- Superseded by: None

## Context

The first implementation supported a process-scoped API token and a public
OAuth client. Live OAuth testing exposed a product-level mismatch for an
unpackaged cross-platform CLI: Chatwork's non-HTTP redirect requires either an
operating-system URI-handler installation, a hosted callback service, or a
manual callback-copy workflow. Each option adds interaction, platform, privacy,
or service boundaries that are not needed to complete the supported Chatwork
tasks.

The API-token path already covers the fixed 32-operation goal with one secret
input and no local credential lifecycle. Keeping OAuth as an optional path
would retain callback discovery, method selection, browser handoff, public
configuration, credential storage, refresh, dependencies, faults, and agent
recovery choices even when the standard workflow does not use them.

## Decision drivers

- Let an agent invoke a Chatwork task without choosing an authentication
  method or discovering a login lifecycle.
- Keep the token out of argv, plaintext configuration, output, fixtures, and
  repository history.
- Remove unused browser, platform credential-store, callback, refresh, and
  dependency boundaries.
- Preserve the fail-closed gate, ephemeral binding, exact production
  destination, and zero-provider-call authentication failures.
- Avoid retaining an undocumented OAuth fallback.

## Considered options

### Keep PAT and OAuth with explicit selection

This preserves already implemented behavior but makes every scoped contract,
failure surface, dependency review, and support matrix carry two paths. It does
not remove the callback interaction that triggered reconsideration.

### Make PAT the default and retain hidden or optional OAuth

This reduces common-path setup but violates the public catalog as source of
truth and leaves dormant credential and dependency behavior. A hidden fallback
also makes account choice harder to reason about.

### Use PAT only

This gives the current product one credential input, one adapter path, and one
recovery contract while retaining all API-task, effect, reference, output, and
transport guarantees.

## Decision

Choose PAT only. `CWK_API_TOKEN` is the sole Chatwork credential input and is
read only from the command-process environment. Every Chatwork API-task
requirement admits exactly `pat`; `CWK_AUTH_METHOD` is removed and has no
selection semantics.

The CLI exposes no `auth login`, `auth status`, `auth logout`, profile,
callback, browser, public-client, persisted-selection, or credential-store
surface. Production composition constructs the Chatwork API adapter directly
from `CWK_API_TOKEN`. Infrastructure validates the token, retains it in a
private process-local record behind an ephemeral binding, and attaches it only
as `x-chatworktoken` to the fixed `https://api.chatwork.com/v2` destination.
Missing or malformed token input fails before a provider task request.

The token is never accepted through argv or a normal command input, persisted
by `cwk`, or returned in output or faults. Environment delivery is a deliberate
trade-off rather than a secure-storage claim; users should inject the value
only into processes that need it and clear inherited values afterward.

Remove the Chatwork OAuth domain/application/infrastructure/CLI code, browser
opener, user-configuration store, OS credential-store adapter, OAuth-specific
fault and recovery declarations, and the `golang.org/x/oauth2` and
`github.com/zalando/go-keyring` dependency graph. OAuth is not retained as a
current-core method. A future OAuth proposal must revise the domain and product
contracts, supersede this ADR, and follow ADR 0001's reviewed-library rule.

## Consequences

### Positive

- Every Chatwork task has one authentication prerequisite and no selector.
- Root help has no authentication lifecycle namespace; scoped help can state
  the exact required environment input and PAT method.
- OAuth callback, refresh, browser, keyring, configuration, and platform
  failure modes disappear from the public contract.
- Two direct pre-v1 dependencies and their transitive platform modules are
  removed.
- Token handling still crosses one infrastructure boundary and remains bound
  to the admitted session immediately before I/O.

### Negative

- Users must obtain and manage a Chatwork API token outside `cwk`.
- Environment values may be observable to parent or same-user processes on
  some systems.
- `cwk` cannot inspect expiry, persist, rotate, revoke, or switch credentials.
- A future OAuth reintroduction is a deliberate compatibility and migration
  change rather than toggling dormant code.

## Migration and retired state

This decision is made before the first stable release. The PAT command surface
and Chatwork task results remain unchanged; the OAuth lifecycle and
`CWK_AUTH_METHOD` selector are removed.

The PAT-only binary does not read or silently delete state created by the
superseded implementation. Silent deletion would cross a platform credential
boundary during an unrelated task, while retaining the keyring dependency only
for cleanup would defeat the reduced dependency decision. A pre-release user
who completed OAuth should revoke that authorization through Chatwork and
remove the retired OS credential entry with service
`cwk.chatwork.oauth2`/account `default` using the operating system's credential
manager. The non-secret retired configuration may be removed from
`${XDG_CONFIG_HOME:-$HOME/.config}/cwk/config.json` on macOS/Linux or
`%AppData%\cwk\config.json` on Windows. The new binary ignores both records.

## Mechanical enforcement

- The public catalog contains no `auth` command and the capability ledger marks
  authentication management excluded.
- Every Chatwork requirement contains exactly `authn.MethodPAT`; scoped help
  names `CWK_API_TOKEN` and no method selector.
- Production composition has one path to `chatworkapi.NewFromEnvironment` and
  no OAuth/configuration/keyring/browser import.
- Adapter tests prove missing/invalid tokens make zero provider calls, exact
  bindings cannot cross clients, and the token reaches only
  `x-chatworktoken` at the fixed destination.
- Catalog tests reject OAuth-only faults and recovery commands referencing the
  removed namespace.
- Secret-canary tests cover stdout, stderr, structured faults, snapshots,
  fixtures, and diagnostics.
- Module tidy/verification and cross-platform architecture checks prove the
  retired dependencies and platform files are absent.
- `task check`, `task security`, and `task public:check` are required acceptance
  evidence.

## Reconsideration signals

Reconsider only when a required user outcome cannot reasonably use a
process-local API token, Chatwork materially changes token availability or
permissions, or measured workflows justify a new OAuth/service boundary. A
future decision must compare its complete agent interaction, platform,
credential, dependency, revocation, and migration costs rather than restoring
the previous code by default.
