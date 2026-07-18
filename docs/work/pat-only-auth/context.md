# Context: PAT-only authentication

## Verified facts

- Chatwork documents API-token authentication as the ordinary API starting
  path.
- The current CLI supports PAT and public-client OAuth through an explicit
  method-selection contract.
- The current OAuth path adds `auth` lifecycle commands, XDG/AppData public
  configuration, an OS credential store, browser opening, PKCE callback input,
  refresh and identity continuity, and two third-party modules.
- An unregistered `cwk://` callback reaches the browser but cannot return to an
  unpackaged CLI; retrying the consumed authorization request produces provider
  error 16000.
- The user selected the API token as the standard path after observing this
  live interaction.

## Constraints

- API tokens remain secrets and cannot enter argv, plaintext configuration,
  output, logs, fixtures, or public history.
- The existing application authentication gate and infrastructure-issued
  binding remain useful for proving authentication before provider I/O.
- External API coverage and output contracts are already complete and must not
  be reopened by the authentication simplification.
- Automated tests use synthetic tokens and local servers only.

## Decision

The supported Chatwork product uses PAT only. `CWK_API_TOKEN` is the sole
credential input and no separate method selector is necessary. Generic domain
vocabulary may continue to model OAuth for reusable boundary tests, but no
Chatwork production adapter, catalog entry, dependency, or documentation may
claim OAuth support.

## Unknowns

- A future release may evaluate OS-protected PAT persistence, but it is outside
  this bounded change and requires a separate thesis/security decision.

## Completion evidence

- `cli.Catalog` contains no authentication lifecycle command and every
  Chatwork task declares only `pat`.
- Production resolves `CWK_API_TOKEN` lazily on the first API task, so root and
  scoped help succeed without reading a credential.
- Missing and malformed PAT tests fail closed before constructing a usable
  authenticated task service; adapter binding tests reject cross-client use.
- OAuth/config/browser/keyring packages and the OAuth/keyring dependency graph
  are absent. `go list -m all` contains only the main module.
- Manual agent-help replay showed one PAT prerequisite, no method selector, and
  exact scoped-help recovery for `chatwork_token_missing`.
- `task check:fast`, `task check`, `task security`, and `task public:check`
  passed on 2026-07-18 with Go 1.26.5.
