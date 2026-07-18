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
- [x] Propagate the selected OAuth public-client/profile contract through the
  architecture, harness enforcement, add-capability workflow, capability
  ledger, provider-specific dependency/store ADR, and public catalog. Evidence:
  `b8bf3b4`, `ed199d8`, ADR 0002, and `3f5e894`.
- [x] Bind every mutation's uncertain-outcome fault to its exact implemented
  read-only reconciliation command. Evidence: catalog contract tests and
  `8689dd3` reject missing, mutating, or divergent recovery.

## Implement shared foundations

- [x] Add synthetic schema fixtures and schema-manifest digests. Evidence:
  `0513d3b` pins the publishable corpus and digest.
- [x] Implement environment PAT authentication and infrastructure binding.
  Evidence: `e425249`; `cafbff9` isolates the non-secret selector projection.
- [x] Review and pin `golang.org/x/oauth2` plus the selected OS credential-store
  dependency, including license, maintenance, transitive graph, supported
  platforms, vulnerability, and failure behavior. Evidence: ADR 0002,
  `43265a9`, `go mod verify`, `task security`, and zero reported vulnerabilities.
- [x] Implement `auth profiles` and exact opaque OAuth profile-reference flow.
- [x] Implement public-client OAuth login with state, PKCE S256, fixed endpoints,
  non-HTTP custom redirect, full callback stdin, and zero secret output.
- [x] Implement secret-free OAuth status and local-only logout with read-only
  unknown-outcome reconciliation.
- [x] Persist OAuth token material only in the OS credential store and implement
  bounded refresh/revalidation within the exact ephemeral binding.
- [x] Require exact `CWK_AUTH_METHOD=pat|oauth2` on API tasks and prove missing,
  invalid, unavailable, and failed selections never fall back or call the API.
  Evidence for the five OAuth/selection items: `2d716dc`, `43265a9`, `3f5e894`,
  CLI selection tests, OAuth race tests, and [agent-readiness.md](agent-readiness.md).
- [x] Implement fixed-origin bounded HTTP transport and provider fault mapping.
  Evidence: `e425249`, `8689dd3`, and the all-operation local-server matrix.
- [x] Implement typed Chatwork semantic values and exact reference validation.
  Evidence: `baa1ed5` and `fb033fc`.
- [x] Implement candidate-C context-capsule renderer and contract tests.
  Evidence: `baa1ed5`, golden/determinism/hostile-output tests, and runtime
  relationship projection tests.

## Implement public task slices

- [x] Account/status and personal-task reads.
- [x] Contact and incoming-request workflows.
- [x] Room discovery, create/show/update/leave/delete workflows.
- [x] Room-member list/change workflows.
- [x] Message list/send/read-state/show/update/delete workflows.
- [x] Room-task list/create/show/status workflows.
- [x] File list/upload/show workflows.
- [x] Invite-link show/create/update/delete workflows.
  Evidence for all public slices: `1e426c7` covers every typed task and fixed
  operation through local transport; `c6434b0` and `3f5e894` bind all 33 task
  outcomes into the validated public catalog.
- [x] Remove or internalize scaffold sample capabilities after replacement.
  Evidence: `sample list`, `sample read`, and `sample.inspect` are absent from
  `DefaultCatalog` and root help; the capability ledger marks `sample.inspect`
  internal, while explicit test-only catalogs retain the generic fixture.

## Verify and close

- [x] Coverage checker reports exactly 32/32 with all negative fixtures passing.
- [x] Set `.harness/chatwork_api_v2.json` `coverage_status` to `complete` and
  prove that every operation has a public capability owner. Evidence: `3f5e894`;
  `contractlint: OK` in every final gate.
- [x] Local-server E2E covers every operation and stable provider faults.
  Evidence: `1e426c7` and `8689dd3`.
- [x] Reference graph and exact round trips pass for all resource kinds.
  Evidence: catalog validation, adapter/runtime contracts, and hostile reference
  tests pass in `task check`.
- [x] Mutation preflight zero-call and unknown-outcome reconciliation tests pass.
  Evidence: `21254db`, `8689dd3`, and runtime zero-call tests.
- [x] Hostile-output, secret-canary, cancellation, bounds, and writer tests pass.
  Evidence: capsule, CLI, API, and OAuth suites pass in `task check`.
- [x] Agent-readiness transcripts meet discovery and no-processing budgets.
  Evidence: [agent-readiness.md](agent-readiness.md) links the executable help,
  runtime, and local-adapter checks.
- [x] OAuth synthetic transcript covers profile discovery, callback/state/PKCE,
  store denial/unavailability, expiry/refresh identity, method selection,
  redaction, and zero-task-call rejection without live credentials. Evidence:
  [agent-readiness.md](agent-readiness.md), `43265a9`, `3f5e894`, and OAuth race
  tests.
- [x] `task check` passes. Evidence: full gate passed on 2026-07-18 with Go
  1.26.5, including race, module verification, security, vulnerability,
  release-lint, public-boundary, and contract checks.
- [x] `task security` passes. Evidence: `repoguard (security): OK`, dependency
  verification, and `No vulnerabilities found.` on 2026-07-18.
- [x] `task public:check` passes. Evidence: `repoguard (public): OK` and
  `contractlint: OK` on 2026-07-18.
- [x] Acceptance criteria and durable documentation are complete. Evidence:
  governing docs, ADR 0002, this work packet, README, and executable contracts
  agree on the finished public surface.
- [x] Temporary diagnostics, test tokens, and local artifacts are absent.
  Evidence: only deterministic synthetic fixtures remain tracked; repository
  status contains the intentional documentation closure only.
