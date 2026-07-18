# Work Context: Complete the first Chatwork API-backed CLI

## Current behavior

- `.harness/project.json` reports `profile: ready`.
- The public sample scaffold has been removed from `DefaultCatalog`; its code is
  retained only as an explicitly constructed offline test fixture. The public
  Chatwork task catalog is being integrated alongside `help`, `doctor`, and
  `version`.
- No production Chatwork network adapter or credential source exists.
- The repository has reusable effect, intent, authentication-gate, binding,
  pagination, fault, catalog, and output-safety foundations.
- The previously accepted theses deliberately deferred full API coverage and
  presentation selection. The user has now selected exhaustive coverage of a
  fixed public snapshot and candidate C for the first complete implementation.

## Relevant structure

- Entry point: `cmd/cwk/main.go`
- Domain rules: `internal/domain/operation`, `authn`, `apicall`, `page`, `fault`
- Application boundaries: `internal/app/authn`, `execution`, `pagination`
- Infrastructure examples: `internal/infra/systemdoctor`; `sampledata` is an
  internal test fixture only.
- CLI catalog/presentation: `internal/cli/catalog.go`, `help.go`, `output.go`
- Coverage/schema ledgers: `.harness/capabilities.json`,
  `.harness/schemas.json`
- Canonical gate: `./scripts/check.sh`

## Constraints

- Public commands express user tasks and never expose an arbitrary transport
  escape hatch.
- Discovery owns ambiguity; action commands require exact opaque references.
- Domain and application layers remain provider- and presentation-independent.
- All mutations pass the shared intent/policy boundary and fail closed.
- API tokens never cross infrastructure or appear in public artifacts.
- Tests use synthetic fixtures and local servers only.

## External facts

- **Chatwork API documentation index**, `https://developer.chatwork.com/llms.txt`,
  checked 2026-07-18: the API Reference section links 32 REST operations.
- **Endpoints**, `https://developer.chatwork.com/ja/docs/endpoints`, checked
  2026-07-18: production base URI is `https://api.chatwork.com/v2`; requests use
  HTTPS; the token is carried in `x-chatworktoken`; POST/PUT forms use
  `application/x-www-form-urlencoded`; the general published limit is 300
  requests per five minutes; message/task creation also has a per-room limit.
- The same endpoints guide describes the API token as non-expiring and broadly
  capable. This makes argv, plaintext project configuration, logs, and fixtures
  unacceptable credential locations.
- The official reference pages checked 2026-07-18 document a maximum of 100
  returned items for `GET /my/tasks`, room messages, room tasks, room files,
  and incoming contact requests. Other list endpoints do not inherit that
  lower statement merely because the project-wide aggregate ceiling is larger.
- **OAuth**, `https://developer.chatwork.com/docs/oauth`, checked 2026-07-18:
  Chatwork supports Authorization Code Grant; public clients require PKCE and a
  redirect URI that does not begin with `http`, while `offline_access` is
  limited to confidential clients. The authorization endpoint is
  `https://www.chatwork.com/packages/oauth2/login.php`, the token endpoint is
  `https://oauth.chatwork.com/token`, and API requests carry OAuth access tokens
  as `Authorization: Bearer`.

## Resolved decisions

- Scope is frozen by operation key and documentation URL, not by whatever the
  provider publishes later.
- Authentication supports one account through PAT and OAuth2. API tasks require
  exact `CWK_AUTH_METHOD=pat|oauth2` and never fall back. PAT is supplied only
  through the command-process environment. OAuth uses one public client,
  Authorization Code Grant, state, PKCE S256, a non-HTTP custom redirect, a full
  callback URL supplied through stdin, and operating-system credential storage.
- OAuth requests `users.all:read rooms.all:read_write
  contacts.all:read_write`; `offline_access` and a client secret are excluded.
  `auth profiles` produces one exact local profile reference for login, status,
  and logout. Login refuses overwrite and logout does not claim remote
  revocation.
- Production transport has a compile-time Chatwork base URL. Tests inject a
  local server through internal construction, never a public base-URL flag.
- Candidate C is a versioned context capsule. It uses deterministic compact
  aliases and hierarchy to reduce repeated labels, while a reference table
  retains exact canonical values. Aliases are never valid action inputs.
- Provider operation coverage and public command design are separate ledgers
  joined by a checked mapping; this permits one user task to compose operations
  and one operation to support more than one safe outcome.
- Metadata/read and non-upload provider calls time out after 20 seconds; upload
  times out after 60 seconds. Every logical operation has one attempt.
- Successful provider bodies are capped at 8 MiB, provider error bodies at
  64 KiB, complete output at 16 MiB, aggregate lists at 10,000 items, and file
  uploads at 5 MiB. The five documented endpoints retain their 100-item limit.
- Exact typed invocation suffices for ordinary creates/updates. Room creation,
  room-member replacement, invite-link create/update, and contact-request
  acceptance require `--confirm=access-change`. Room leave/delete, message
  deletion, invite-link deletion, and request rejection require
  `--confirm=destructive`.
- Confirmation is invocation-local. Mutations are not retried, and uncertain
  outcomes reconcile only through an exact read-only catalog task.
- `.harness/chatwork_api_v2.json` remains `coverage_status: planned` during
  implementation and must be changed to `complete` for goal closure.

## Remaining design questions

- [ ] Decide whether file upload reads only an explicit file path or supports
  stdin; either path must have a byte ceiling and secret-safe diagnostics.

## Conditional live-test authorization

The user may provide a dedicated test account after local authentication and
adapter validation are complete. The currently authorized live actions are
limited to creating an isolated test room and sending, updating, and reading
messages in that room. Contact-related changes are forbidden. Room deletion,
membership changes, invitation-link publication, file upload, and any action
against pre-existing rooms/data require a new explicit approval. Before asking
for credentials, present the exact bounded command sequence and side effects.

OAuth is part of the active goal by the user's later explicit instruction. This
does not broaden live-test authority: contacts remain forbidden, and OAuth
credentials must not be supplied until the local flow, redaction, and adapter
tests pass and the exact bounded live sequence is presented.
- [ ] Name each mutation's exact read-only reconciliation command as its public
  catalog contract is implemented; the policy and read-only restriction are
  fixed, but command paths do not exist yet.

## Thesis evidence

- The user explicitly values exhaustive public API task coverage for the first
  complete product, while retaining agent certainty and processed output.
- A perpetual “latest API” goal would not close. Freezing the 2026-07-18 set is
  the boundary that makes exhaustive coverage testable and finite.
- API coverage does not justify a raw endpoint mirror; the task-oriented thesis
  remains the design constraint for catalog grouping.
- The user selected one practical presentation before later token experiments,
  so empirical candidate competition is deferred rather than made a blocker.

## Security and public-boundary notes

- Assets: a full-access long-lived API token; OAuth authorization code, state,
  PKCE verifier, access/refresh tokens, and OS credential-store entry; rooms,
  memberships, messages, tasks, files, invite links, and contact requests.
- Effects include notifications, access changes, destructive room/message/link
  actions, file upload, and task state changes.
- Allowed production destinations: `https://api.chatwork.com/v2`, the exact
  Chatwork authorization endpoint, and `https://oauth.chatwork.com/token`.
- OAuth protocol machinery uses the reviewed `golang.org/x/oauth2` dependency;
  the operating-system credential-store dependency requires an explicit
  license, maintenance, transitive-graph, platform, and failure review.
- Fixtures will be synthetic derivatives of documented shapes and registered by
  digest; no provider example body is copied wholesale without a license review.

## Glossary

- **operation snapshot**: the fixed set of 32 official REST operation keys in
  scope, independent of future documentation changes.
- **coverage mapping**: checked operation-to-capability ownership, not a public
  command registry.
- **context capsule / candidate C**: the selected deterministic first
  presentation, with compact display aliases, canonical references, explicit
  relations/bounds, and structurally framed untrusted text.
- **canonical reference**: exact validated provider identity accepted unchanged
  by an action command.
