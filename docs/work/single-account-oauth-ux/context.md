# Context: One-command single-account OAuth login

## Verified facts

- The product supports exactly one Chatwork account and has no planned
  near-term multiple-profile requirement.
- The current public flow requires `auth profiles`, a fixed profile reference,
  `CWK_OAUTH_CLIENT_ID`, `CWK_OAUTH_REDIRECT_URI`, and
  `CWK_AUTH_METHOD=oauth2` despite there being no choice among OAuth profiles.
- Chatwork public clients use Authorization Code Grant. The existing adapter
  already enforces state, PKCE S256, exact redirect matching, fixed provider
  endpoints, scope/account continuity, and OS credential storage.
- The user deliberately chose one manual full-callback paste instead of OS URI
  registration. The callback must remain stdin-only so code and returned state
  do not enter argv.
- `CWK_OAUTH_CLIENT_ID`, the redirect URI, and the `oauth2` selector are
  non-secret public configuration. Access/refresh tokens are secrets.
- The current generic catalog requires every act command to consume an opaque
  reference and every mutation target to come from a CLI input. A narrow,
  executable command-bound singleton contract is therefore required before
  removing `--profile`.

## Constraints

- Exact user-selected Chatwork resources still require canonical opaque
  references. The new singleton rule must not weaken remote target binding.
- The browser opener crosses a process/OS boundary. It must be allowlisted,
  shell-free, bounded, context-aware, and must not expose its URL in errors.
- The authorization URL contains state and a PKCE challenge and may be briefly
  visible to the platform opener process. It contains no authorization code,
  PKCE verifier, access token, refresh token, or client secret. PKCE and exact
  callback/state validation remain mandatory.
- Stored public configuration must be bounded, schema-versioned, atomically
  replaced, and separated from the OS credential record.
- A persisted OAuth selection is explicit because it is created by the login
  task. Absence remains an error; selected-method failure never enables PAT.
- Automated tests use temporary directories, fake openers, fake credential
  stores, and synthetic provider endpoints only.

## Resolved implementation decisions

- Public configuration is committed before consent. A failed consent leaves a
  reusable non-secret record and no credential; a usable token can never exist
  without its exact configuration.
- macOS and Linux use XDG configuration semantics; Windows uses AppData. A
  relative `XDG_CONFIG_HOME` fails closed instead of being interpreted against
  process state.
- Headless or unavailable platform opening retains the manual authorization-URL
  fallback; no shell participates on any platform.
- Legacy OAuth client-ID/redirect environment reads were removed. Only an
  explicitly present `CWK_AUTH_METHOD` can override stored method selection.

## Evidence to capture

- Before/after authentication transcript and invocation count.
- Catalog negative tests proving remote mutations still require references.
- Configuration hostile-file, symlink, size, atomicity, and permission tests.
- Browser opener validation, zero-call, cancellation, and redaction tests.
- OAuth no-fallback and zero-provider-task-call tests using persisted selection.

## Completion evidence

- The former first-use contract required profile discovery, three OAuth
  configuration exports, a fixed profile argument, consent, and a callback
  paste. The completed contract is one
  `cwk auth login --client-id <public-client-id>` invocation, automatic browser
  handoff when available, and one complete callback paste. Later login is
  `cwk auth login`.
- `cwk help auth login --format agent` returns the complete first-run flag,
  stdin callback, browser fallback, fixed target, output, failure, and recovery
  contract in one schema-v3 scoped response; it exposes no OAuth profile or
  client-registration environment input.
- Catalog negative tests preserve opaque references for remote/user-selected
  targets while allowing only one reference-free `tool_local` singleton.
- Configuration tests cover strict schema and bounds, duplicate/unknown fields,
  permissions, symlink/special-file rejection, atomic replacement, cancellation,
  XDG resolution, AppData resolution, and platform cross-builds. Reads are
  confined through `os.Root` to the validated `cwk` application directory.
- Browser tests cover the exact Chatwork origin/path, exact redirect and scope,
  canonical single-value OAuth query, PKCE S256, hostile URLs, cancellation,
  redacted launcher faults, and compatibility with `oauth2.Config.AuthCodeURL`.
- Selection and CLI tests prove first-login ordering, later login without client
  ID, no PAT/OAuth probing or fallback, one callback line, automatic-open and
  fallback prompts, bounded output, and callback/code/state redaction.
- On 2026-07-18 with Go 1.26.5, `task check`, `task security`, and
  `task public:check` passed. The full gate included format, architecture,
  catalog, unit, race, vet, tidy, security, vulnerability, release, and public
  repository checks.
