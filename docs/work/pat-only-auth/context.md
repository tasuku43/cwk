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
