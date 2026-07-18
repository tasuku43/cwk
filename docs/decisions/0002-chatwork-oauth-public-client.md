# ADR 0002: Use a fixed public-client OAuth flow and operating-system credential store

- Status: Superseded
- Date: 2026-07-18
- Amended: 2026-07-18, single-account command-bound login and public configuration
- Deciders: Chatwork CLI maintainers
- Scope: Chatwork authentication, infrastructure dependencies, credential storage, and harness
- Supersedes: None
- Superseded by: [ADR 0003](0003-chatwork-pat-only.md)

## Context

The first complete Chatwork CLI must support both a process-scoped PAT and
OAuth 2.0 without leaking credentials across the infrastructure boundary or
making an agent guess which credential will be used. Chatwork supports the
Authorization Code Grant for public clients, requires PKCE for that client
class, and rejects an HTTP redirect URI for it. A conventional loopback HTTP
listener is therefore not the selected callback model.

OAuth adds two distinct implementation concerns. Protocol machinery needs a
maintained implementation of authorization URLs, PKCE, code exchange, token
validity, and refresh. Persistence needs a cross-platform abstraction over the
operating-system credential facilities. ADR 0001 requires both concerns to stay
optional and infrastructure-only until a derived product selects OAuth; this
project has now made that selection.

## Decision drivers

- Implement Chatwork's supported public-client flow without a client secret or
  a locally maintained OAuth protocol implementation.
- Keep codes, verifiers, tokens, credential-store keys, authenticated clients,
  and private causes inside infrastructure.
- Make authentication method and account continuity deterministic.
- Keep refresh races and rotated-token persistence from authorizing a task with
  an identity or credential state different from the admitted session.
- Use native credential protection on macOS, Linux/BSD, and Windows, while
  failing closed on systems where it is unavailable.
- Keep the dependency and license surface explicit and replaceable.

## Considered options

### Implement OAuth and platform stores locally

This avoids external modules but makes the project responsible for subtle
state, PKCE, refresh, Keychain, Secret Service, and Credential Manager behavior.
That security and maintenance surface is larger than the reviewed dependencies.

### Use a provider SDK and plaintext configuration file

A provider SDK would broaden the dependency surface beyond the needed protocol
primitives. A plaintext token file would work in more headless environments but
would weaken the credential boundary and create permissions, backup, and
cleanup risks. Neither is justified by the fixed task surface.

### Use a fixed public-client adapter and OS credential-store abstraction

This preserves the four-layer boundary, uses narrowly scoped libraries for the
two specialized concerns, and makes an unavailable store an explicit recovery
condition rather than a reason to downgrade storage.

## Decision

Choose the fixed public-client adapter and OS credential-store abstraction.

The public flow is Authorization Code Grant with a fresh unpredictable state
and PKCE S256 verifier for every login. The registered redirect is the exact
non-HTTP, non-HTTPS `cwk://oauth/callback` URI. The CLI starts no listener: it
opens the transient consent URL through a bounded shell-free platform opener,
prints the URL only when opening fails, and reads the complete redirected URI
from stdin. It validates the exact redirect components and state before
exchanging one code. First login provides a non-secret client ID in argv and
has no client secret. The fixed scopes are `users.all:read`,
`rooms.all:read_write`, and `contacts.all:read_write`; `offline_access` is not
requested. Authorization, token, identity, and API destinations are fixed in
production, and credential-bearing redirects are disabled.

`golang.org/x/oauth2` is pinned at `v0.36.0` and imported only by
`internal/infra`. It is maintained by the Go project, has a BSD-3-Clause
license, and supplies the reviewed authorization URL, PKCE, exchange, token
validity, and refresh primitives. Provider-specific callback validation,
scope policy, identity verification, bounded transports, stable faults, and
redaction remain in the Chatwork infrastructure adapter rather than being
delegated to the library.

`github.com/zalando/go-keyring` is pinned at `v0.2.8` and imported only by
`internal/infra`. It has an MIT license and presents the small get/set/delete
surface needed here. Its platform backends are macOS Keychain through the
system `security` program, Secret Service over D-Bus on Linux/BSD, and Windows
Credential Manager. The reviewed transitive modules are
`github.com/danieljoos/wincred v1.2.3` (MIT),
`github.com/godbus/dbus/v5 v5.2.2` (BSD-2-Clause), and
`golang.org/x/sys v0.27.0` (BSD-3-Clause). The selected `x/oauth2` module also
declares `cloud.google.com/go/compute/metadata v0.3.0` in its module graph; the
Chatwork adapter imports only the root OAuth package and does not use cloud
metadata discovery.

The keyring service/account identifiers are fixed private infrastructure
constants and are not public target identity. The store contains
the access token, optional refresh token, token type, provider-advertised
expiry, exact granted scope set, and verified Chatwork account ID as one bounded
credential record. No plaintext store or alternate secret source is attempted
when the selected OS backend is missing, locked, denied, or returns an error.

An explicitly present `CWK_AUTH_METHOD=pat|oauth2` is authoritative. Otherwise
the exact `oauth2` selection persisted by the first login attempt is used. Missing,
unknown, corrupt, or unavailable selection fails before credential access.
Selecting one method never probes, prefers, or falls back to the other,
including after OAuth login, store, expiry, refresh, identity, or permission
failure.

The platform user configuration is a separate bounded, schema-versioned,
atomically replaced record containing only the exact method, public client ID,
and fixed redirect. It never stores token, callback, code, state, verifier,
account identity, or credential-store key material. It is written before
consent, so a canceled or rejected attempt leaves reusable public configuration
but no credential, while a usable token can never exist without its exact
public configuration. macOS and Linux use
`${XDG_CONFIG_HOME:-$HOME/.config}/cwk/config.json`; Windows uses
`%AppData%\cwk\config.json`. Login, status, and logout
bind one catalog-declared fixed `tool_local` authentication target; the former
public fixed-profile reference and its discovery command are removed.

At login and after every token refresh, the adapter calls the fixed Chatwork
identity endpoint with the candidate access token, validates the required
scopes, and binds the resulting exact account to the private record. Refresh is
serialized for that record. A refreshed account must equal the stored and
admitted account, and the rotated credential must be persisted successfully
before its Bearer header can authorize the provider task request. A mismatch,
refresh failure, cancellation, or store failure makes zero provider task
requests and cannot trigger PAT fallback.

## Dependency review

Both direct dependencies are pre-v1 modules, so updates are reviewed changes
rather than compatibility-presumed maintenance. Their versions and checksums
are pinned in `go.mod` and `go.sum`; the security gate runs module verification
and vulnerability analysis. The project does not copy their source or license
text into runtime output.

`go-keyring` reduces per-platform code but retains platform operational
requirements: macOS must provide its standard `security` tool, Linux/BSD must
provide a usable Secret Service session, and Windows must provide Credential
Manager. Headless Linux and locked desktop sessions can legitimately produce a
store-unavailable fault. That is an explicit unsupported-environment outcome,
not permission to add a plaintext fallback. Platform release tests must build
all targets, while adapter contract tests use a fake store and never touch a
developer keychain.

The dependency review sources are the exact module `LICENSE`, `go.mod`, and
README files in the checksum-verified module cache, plus the upstream package
documentation linked from ADR 0001. A future dependency update repeats the
license, transitive graph, platform, vulnerability, and adapter-contract
review.

## Consequences

### Positive

- OAuth protocol and native credential storage stay replaceable behind one
  infrastructure boundary.
- Agents and users choose authentication deterministically.
- First login needs one command, browser consent, and one callback paste;
  subsequent OAuth commands and API tasks need no registration exports.
- Token refresh cannot silently change the account used by an admitted task.
- Native stores protect persisted credentials without adding a product-owned
  plaintext format.

### Negative

- OAuth is unavailable when the platform credential service is unavailable.
- The macOS backend invokes a fixed system program; Linux/BSD depends on a
  user-session D-Bus service; platform behavior needs dedicated test coverage.
- Browser launch has platform/headless failure modes and its subprocess may
  briefly expose state and the public PKCE challenge in an authorization-URL
  argument; it exposes no callback code, verifier, token, or client secret.
- Public-client refresh lifetime is provider-limited because `offline_access`
  is not available; re-login remains an expected recovery.
- Two pre-v1 dependencies and their transitive modules require ongoing review.

## Mechanical enforcement

- Architecture lint keeps both third-party imports out of domain, application,
  CLI, and command packages; dependency review rejects moving credential types
  across the infrastructure boundary.
- Catalog and CLI tests require an authoritative environment selection or the
  exact login-persisted OAuth selection and prove missing, unknown, unavailable,
  and failed selected methods make zero task calls with no fallback.
- Catalog tests constrain the reference-free auth lifecycle to one explicit
  command-bound `tool_local` target and leave every remote reference rule intact.
- Public-configuration and opener tests cover schema/bounds/symlinks/atomicity,
  exact authorization origin, shell-free invocation, fallback, and redaction.
- OAuth adapter tests prove state and exact redirect rejection before exchange,
  PKCE S256 verifier/challenge agreement, public-client token exchange without
  a client secret, fixed scopes without `offline_access`, and fixed production
  destinations with redirects disabled.
- Store tests prove missing, unavailable, denied, oversized, corrupt, and
  canceled operations yield typed secret-free faults and never use plaintext or
  another authentication source.
- Binding fixtures exercise two synthetic accounts, unknown/stale/cross-session
  bindings, and identity mismatch with zero unintended provider task calls even
  though the public product exposes only one account.
- Refresh-race tests run concurrent task authorization against one expired
  credential and prove one serialized rotation, exact account continuity,
  successful persistence before task authorization, and no stale-token reuse.
- Secret-canary tests cover consent/callback, exchange and refresh provider
  bodies, store errors, stdout, stderr, structured faults, logs, snapshots, and
  test diagnostics.
- `go mod verify`, vulnerability analysis, cross-platform builds, `task check`,
  and `task security` are required acceptance evidence.

## Compatibility and migration

This decision adds the single-account OAuth workflow but does not change
Chatwork task results. PAT and OAuth converge on the same secret-free session
and ephemeral binding. Adding another profile/account necessarily invalidates
the fixed target and requires an opaque-reference discovery flow. Replacing
either dependency, adding a
confidential or device client, changing callback delivery, or adding a
plaintext/non-native store requires a superseding ADR and product/security
migration plan.

## Reconsideration signals

Reconsider when Chatwork changes its public-client grant or redirect rules, a
selected module is deprecated or materially changes maintenance/security
posture, a supported release platform loses its backend, or measured user
outcomes show that failing closed on headless systems prevents required use.
