# Work Goal: One-command single-account OAuth login

- Status: In progress
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: ADR 0002 (to be amended)

## Outcome

A user configures and authorizes the one supported Chatwork account with one
`cwk auth login --client-id <public-client-id>` invocation, a browser consent,
and one complete callback-URL paste. Successful login persists only non-secret
public-client selection in the platform user configuration and tokens in the
operating-system credential store. Later `auth login`, `auth status`, `auth
logout`, and Chatwork API tasks require neither a profile reference nor OAuth
configuration exports.

## Why now

The completed OAuth implementation proved the protocol and storage boundary,
but its fixed `--profile cwk_chatwork_oauth_public_v1` argument and three
repeated exports do not resolve any ambiguity in a product that deliberately
supports one account. The user selected a manual full-callback paste over OS
URI-handler registration, while asking to remove every other avoidable step.
This is direct thesis evidence that a mandatory opaque reference is ceremony
when an exact command already binds a tool-owned singleton.

## Non-goals

- Multiple accounts, profiles, tenants, or account switching.
- OS custom-URI handler registration, loopback HTTP, hosted callback relay, or
  automatic callback capture.
- Persisting PATs, OAuth tokens, authorization codes, state, or PKCE verifiers
  in the user configuration.
- Silent credential probing, OAuth-to-PAT fallback, or changing Chatwork API
  task/output coverage.
- Remote token revocation, release signing, or packaging work.

## Acceptance criteria

- [ ] First login is `cwk auth login --client-id <public-client-id>`; the exact
      redirect is fixed to `cwk://oauth/callback`.
- [ ] The CLI opens the Chatwork authorization URL in the default browser when
      the platform opener is available, otherwise prints the transient URL as
      a fallback, then reads exactly one complete callback URL from stdin.
- [ ] Later login/status/logout commands take no profile argument, and
      `auth profiles` is no longer public.
- [ ] A successful OAuth login makes later API tasks use the stored explicit
      OAuth selection without `CWK_AUTH_METHOD`, `CWK_OAUTH_CLIENT_ID`, or
      `CWK_OAUTH_REDIRECT_URI` exports.
- [ ] An explicit `CWK_AUTH_METHOD=pat` remains possible for PAT automation;
      invalid or failed selected methods never probe or fall back.
- [ ] Public configuration contains only schema version, `oauth2`, client ID,
      and the fixed redirect; token material remains only in the OS credential
      store and callback/code/verifier material remains transient.
- [ ] The fixed command-bound singleton target is mechanically declared and
      validated rather than implemented as an undocumented default.
- [ ] Agent help reaches login in one scoped request and names the first-run
      client ID, browser, callback, storage, and recovery behavior without
      requiring discovery of a synthetic profile.
- [ ] `task check`, `task security`, and `task public:check` pass.

## Governing documents

- Thesis: Axioms 1, 2, 6, and 8; single-account authentication decision
- Product contract section: Authentication and external-call decisions
- Architecture or security invariant: Command roles/reference flow,
  controlled process/filesystem boundaries, and credential separation
- Existing ADR: ADR 0002, Chatwork public OAuth client

## Completion definition

The work is complete when the executable CLI, catalog, persistent configuration
boundary, OAuth adapter, tests, README, agent-readiness evidence, governing
documents, and ADR agree on the no-profile one-command flow; all required gates
pass; and no live credential, callback, browser history, or temporary artifact
is retained.
