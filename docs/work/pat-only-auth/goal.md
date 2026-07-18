# Work Goal: Make the Chatwork API token the sole authentication path

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: ADR 0001; ADR 0003 (supersedes ADR 0002)

## Outcome

An agent or operator can run every supported Chatwork task by supplying only
`CWK_API_TOKEN` to the command process. The CLI has no OAuth setup, callback,
profile, client registration, authentication-method selector, browser handoff,
or persisted credential state to discover or reconcile.

## Why now

Live OAuth testing showed that Chatwork's authorization-code flow leaves an
unpackaged cross-platform CLI with either an OS URL-handler installation, a
hosted callback service, or a manual callback-copy workflow. The user selected
the API token as the standard path to remove that interaction cost before the
first stable release.

## Non-goals

- Persisting the API token in a project file, XDG configuration, or another
  plaintext store.
- Accepting the token through argv.
- Retaining OAuth as an undocumented or fallback path.
- Supporting multiple accounts or implicit account selection.
- Changing the fixed 32-operation Chatwork coverage, command outcomes,
  candidate-C presentation, mutation policy, or provider-call bounds.
- Adding a hosted callback service, custom URI handler, device grant, or
  confidential OAuth client.

## Acceptance criteria

- [x] Every Chatwork API task declares PAT as its only accepted method and reads
  the token only from `CWK_API_TOKEN` in the command process.
- [x] `CWK_AUTH_METHOD`, `auth login`, `auth status`, `auth logout`, OAuth
  configuration, callback input, browser opening, OS credential storage, and
  OAuth dependencies are absent from the runnable product.
- [x] Missing or invalid token input fails before a provider task request with
  structured recovery pointing to exact scoped help.
- [x] Root and scoped agent help let an agent discover and invoke a known
  Chatwork task without an authentication-method choice or authentication
  lifecycle command.
- [x] No token is accepted in argv, persisted by `cwk`, emitted to stdout or
  stderr, or included in fixtures and diagnostics.
- [x] Existing task semantics, references, effects, mutation confirmation,
  API coverage, output bounds, and candidate-C output remain unchanged.
- [x] `task check`, `task security`, and `task public:check` pass.

## Governing documents

- Thesis: first complete implementation and authentication selection
- Product contract section: Authentication and external-call decisions
- Architecture or security invariant: Chatwork authentication topology and
  credentials and secrets
- Existing ADR: ADR 0003 supersedes ADR 0002 because the selected OAuth
  mechanism is removed

## Completion definition

The work is complete when the public and internal OAuth surface is removed,
PAT-only behavior and zero-call failures are mechanically tested, durable
contracts agree, all required verification profiles pass, and the changes are
committed in reviewable units.

Completed on 2026-07-18 with implementation commit `0f6d49b`, durable-contract
commit `6032b27`, and the verification evidence recorded in `context.md`.
