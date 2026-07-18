# Work Goal: Complete the first Chatwork API-backed CLI

- Status: Accepted
- Owner: Project maintainers
- Target: First complete API-backed implementation
- Related ADRs: [ADR 0001](../../decisions/0001-oauth-library-boundary.md)

## Outcome

An agent can use `cwk` to complete every user task represented by the 32 REST
operations in the Chatwork API documentation snapshot checked on 2026-07-18.
The public CLI remains task-oriented: it does not expose arbitrary HTTP routes,
methods, headers, or bodies. The first stable presentation is the selected
context-capsule design (candidate C), including canonical references,
relationship-aware hierarchy where applicable, explicit bounds, and safe
external-text framing.

The completed product includes the concrete example “get the messages in this
room”: discover a room, pass its exact reference to the bounded message task,
and understand the result without `jq`, `grep`, provider-notation parsing, a
custom join, source inspection, or an exploratory Chatwork request.

The same finite completion includes both process-local PAT authentication and
one public-client OAuth 2.0 profile. An agent can discover the profile, complete
login with state and PKCE S256 through a non-HTTP custom redirect and stdin
callback, inspect its secret-free status, remove its local stored credential,
and select PAT or OAuth2 for an API task without an implicit fallback.

## Fixed scope

The coverage universe is the 32 API reference operations linked by the official
`https://developer.chatwork.com/llms.txt` index as retrieved on 2026-07-18.
Future additions to that index do not extend this work automatically. A checked
coverage manifest maps every one of the 32 operations to at least one public
capability and maps every Chatwork-backed public capability back to a reviewed
operation.

## Non-goals and prohibitions

- Do not chase Chatwork operations published after the fixed snapshot.
- Do not add webhook receivers, organization/admin APIs, private endpoints, or
  undocumented provider behavior.
- Do not expose a raw route, HTTP method, header, form, query, or response-body
  passthrough.
- Do not generate a one-command-per-endpoint mirror without a user-task design.
- Do not perform hidden fuzzy selection or accept presentation aliases as
  authorization identity.
- Do not accept an API token in argv, persist it in plaintext configuration,
  render it, log it, or include real credentials/data in tests and fixtures.
- Do not retry an unsafe mutation or classify an uncertain mutation as success.
- Do not silently truncate, emit an undeclared partial success, or infer a
  reply/quote/identity relation from prose, names, layout, or proximity.
- Do not implement candidate A/B variants, a presentation competition, or
  further token optimization in this work. Candidate C is the sole first
  presentation contract.
- Do not add confidential-client, device, or loopback-HTTP OAuth grants;
  multiple accounts/profiles; a GUI; release publication; or unrelated
  refactoring.
- Do not require live Chatwork access for the test suite or collect real account
  data to build fixtures.

## Acceptance criteria

- [ ] The governing theses, product contract, architecture, security model,
  harness, capability ledger, and add-capability workflow agree with exhaustive
  coverage of the fixed 32-operation snapshot.
- [ ] A machine-checked manifest proves 32/32 operation coverage and rejects an
  unowned operation, unknown operation, duplicate operation key, non-public
  capability, or Chatwork capability with no upstream operation.
- [ ] Root/scoped agent help lets an agent select and invoke every task within
  the existing two-invocation discovery budget.
- [ ] Discover commands produce exact room, account, message, task, file, and
  incoming-request references required by action commands; round trips preserve
  every byte.
- [ ] A single-account authentication boundary supports exact
  `CWK_AUTH_METHOD=pat|oauth2` selection without fallback. PAT remains
  process-local; OAuth uses the reviewed public-client Authorization Code +
  state + PKCE S256 flow, a registered non-HTTP custom redirect, full callback
  through stdin, and operating-system credential storage. Both keep secrets in
  infrastructure, bind the validated session to each task call, and allow only
  the documented Chatwork HTTPS destinations in production.
- [ ] `auth profiles` produces one opaque OAuth profile reference consumed
  unchanged by login/status/logout. Login refuses overwrite, status is
  secret-free and read-only, logout removes only local credential material, and
  unknown local mutation outcomes reconcile through exact `auth status`.
- [ ] Metadata/read and non-upload calls time out at 20 seconds, upload at 60
  seconds; every operation has one attempt; success/error bodies are capped at
  8 MiB/64 KiB, output at 16 MiB, aggregate lists at 10,000 items, documented
  endpoint lists at 100 items, and upload at 5 MiB. Declared partial provider
  windows remain explicit in successful results.
- [ ] Every mutation declares effect, exact target roles, impact, idempotency,
  policy, and read-only reconciliation for an uncertain outcome. Ordinary exact
  creates/updates require no extra flag; the reviewed access-changing and
  destructive sets require exact `--confirm=access-change` and
  `--confirm=destructive`. Rejection, malformed input, authentication failure,
  and permission failure make zero mutation attempts.
- [ ] Candidate C renders deterministic context capsules from typed task
  results. Its compact aliases are display-only; the same capsule exposes exact
  canonical references for subsequent commands.
- [ ] Synthetic fixtures and a local HTTP test server verify the request,
  successful response, empty response, provider fault, cancellation, bounds,
  hostile text, and secret-redaction behavior for all 32 operations.
- [ ] Agent-readiness transcripts complete representative reads and mutations
  without external post-processing, provider exploration, or guessed identity.
- [ ] `task check`, `task security`, and `task public:check` pass.

## Governing documents

- Thesis: [Project theses](../../00_theses.md)
- Product contract: [Product contract](../../01_product_contract.md)
- Architecture: [Architecture](../../02_architecture.md)
- Security: [Security model](../../03_security_model.md)
- Harness: [Harness](../../04_harness.md)

## Completion definition

This work is complete only when every acceptance criterion above has linked
evidence in `tasks.md`, the fixed coverage check reports exactly 32/32 with
`coverage_status: complete`, all
three required profiles pass, and no temporary diagnostic, live credential,
real Chatwork data, or unreviewed generated artifact remains. Passing a useful
subset, including the message example, is progress but is not completion.
