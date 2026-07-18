# Work Tasks: Complete the first Chatwork API-backed CLI

## Understand and decide

- [x] Read governing theses, product, architecture, security, harness,
  authentication, external API, and agent-readiness documents. Evidence:
  reviewed before goal creation on 2026-07-18.
- [x] Confirm bootstrap readiness and current runnable surface. Evidence:
  `go run ./tools/projectmeta --field profile` returned `ready`; repository was
  clean and contains no Chatwork adapter.
- [x] Freeze the public operation universe. Evidence: official Chatwork
  `llms.txt` linked 32 API reference operations on 2026-07-18.
- [x] Confirm outcome, termination condition, and prohibitions in `goal.md`.
- [x] Select candidate C as the sole first presentation and defer comparison.
- [x] Promote the revised thesis/product/security decisions. Evidence: project,
  product, architecture, security, harness, authentication, external-API,
  agent-readiness, AGENTS, and add-capability contracts select candidate C as
  a replaceable first presentation and fix PAT/mutation boundaries. OAuth was
  subsequently added to this finite goal and requires its own dependency/store
  review before implementation is accepted.
- [x] Add the checked 32-operation manifest and bidirectional coverage rules.
  Evidence: `.harness/chatwork_api_v2.json` and `tools/contractlint` pin all 32
  exact ID/method/path tuples, reject substitutions, and require a public owner
  for each operation when `coverage_status` becomes `complete`;
  `go test ./tools/contractlint` and `go run ./tools/contractlint` pass.
- [x] Pin numeric timeout, byte, item, call, retry, and upload limits. Evidence:
  the checked manifest and governing contracts fix 20s/60s timeouts, one
  attempt, 8 MiB/64 KiB provider bodies, 16 MiB output, 10,000 aggregate items,
  five documented 100-item endpoints, and 5 MiB upload.
- [x] Decide mutation confirmation classes. Evidence: ordinary exact
  creates/updates need no extra flag; exact access-changing and destructive
  operation-ID sets are pinned to `--confirm=access-change` and
  `--confirm=destructive`; uncertain results are non-retryable and read-only.
- [ ] Propagate the selected OAuth public-client/profile contract through the
  architecture, harness enforcement, add-capability workflow, capability
  ledger, provider-specific dependency/store ADR, and public catalog.
- [ ] Bind every mutation's uncertain-outcome fault to its exact implemented
  read-only reconciliation command.

## Implement shared foundations

- [ ] Add synthetic schema fixtures and schema-manifest digests.
- [ ] Implement environment PAT authentication and infrastructure binding.
- [ ] Review and pin `golang.org/x/oauth2` plus the selected OS credential-store
  dependency, including license, maintenance, transitive graph, supported
  platforms, vulnerability, and failure behavior.
- [ ] Implement `auth profiles` and exact opaque OAuth profile-reference flow.
- [ ] Implement public-client OAuth login with state, PKCE S256, fixed endpoints,
  non-HTTP custom redirect, full callback stdin, and zero secret output.
- [ ] Implement secret-free OAuth status and local-only logout with read-only
  unknown-outcome reconciliation.
- [ ] Persist OAuth token material only in the OS credential store and implement
  bounded refresh/revalidation within the exact ephemeral binding.
- [ ] Require exact `CWK_AUTH_METHOD=pat|oauth2` on API tasks and prove missing,
  invalid, unavailable, and failed selections never fall back or call the API.
- [ ] Implement fixed-origin bounded HTTP transport and provider fault mapping.
- [ ] Implement typed Chatwork semantic values and exact reference validation.
- [ ] Implement candidate-C context-capsule renderer and contract tests.

## Implement public task slices

- [ ] Account/status and personal-task reads.
- [ ] Contact and incoming-request workflows.
- [ ] Room discovery, create/show/update/leave/delete workflows.
- [ ] Room-member list/change workflows.
- [ ] Message list/send/read-state/show/update/delete workflows.
- [ ] Room-task list/create/show/status workflows.
- [ ] File list/upload/show workflows.
- [ ] Invite-link show/create/update/delete workflows.
- [x] Remove or internalize scaffold sample capabilities after replacement.
  Evidence: `sample list`, `sample read`, and `sample.inspect` are absent from
  `DefaultCatalog` and root help; the capability ledger marks `sample.inspect`
  internal, while explicit test-only catalogs retain the generic fixture.

## Verify and close

- [ ] Coverage checker reports exactly 32/32 with all negative fixtures passing.
- [ ] Set `.harness/chatwork_api_v2.json` `coverage_status` to `complete` and
  prove that every operation has a public capability owner.
- [ ] Local-server E2E covers every operation and stable provider faults.
- [ ] Reference graph and exact round trips pass for all resource kinds.
- [ ] Mutation preflight zero-call and unknown-outcome reconciliation tests pass.
- [ ] Hostile-output, secret-canary, cancellation, bounds, and writer tests pass.
- [ ] Agent-readiness transcripts meet discovery and no-processing budgets.
- [ ] OAuth synthetic transcript covers profile discovery, callback/state/PKCE,
  store denial/unavailability, expiry/refresh identity, method selection,
  redaction, and zero-task-call rejection without live credentials.
- [ ] `task check` passes. Evidence:
- [ ] `task security` passes. Evidence:
- [ ] `task public:check` passes. Evidence:
- [ ] Acceptance criteria and durable documentation are complete.
- [ ] Temporary diagnostics, test tokens, and local artifacts are absent.
