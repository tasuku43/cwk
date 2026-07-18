# Work Plan: Complete the first Chatwork API-backed CLI

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Implement a reviewed, task-oriented catalog over a fixed machine-readable
32-operation snapshot. Build shared PAT/OAuth authentication and bounded HTTP
transport once, then add resource-centered vertical slices for identity,
contacts, rooms/members, messages, tasks, files, invite links, and incoming
requests. Every slice owns typed semantic results and maps its required upstream
operations in the coverage manifest. Candidate C is the only public data
presentation added by this work.

## Alternatives considered

### Generate a public command for every endpoint

Rejected because it makes provider transport the product vocabulary, cannot
encode composed user outcomes, and creates a raw API explorer with a friendly
name.

### Implement reads first and declare writes deferred

Rejected for this goal because it would not satisfy the explicitly fixed
32-operation coverage universe. Slices may land incrementally, but completion
requires the write and delete operations with stronger policy and tests.

### Run a presentation competition first

Deferred. The user selected candidate C as the first complete implementation
and explicitly placed further token optimization afterward. This work fixes a
versioned C contract and does not claim it is globally optimal.

## Design

### Public contract

Resource namespaces express tasks: account/status, contacts, rooms/members,
messages, tasks, files, invite links, and incoming contact requests. Discovery
commands return canonical references; exact reads and mutations consume them.
Commands expose effect, authentication, inputs, result meaning, completeness,
faults, recovery, and workflows through `cli.Catalog`.

The context-capsule presentation contains:

1. a version, capability, result status, and coverage header;
2. a deterministic reference dictionary from compact local aliases to exact
   canonical references and safe labels;
3. task facts, with relationship hierarchy only where the typed result declares
   it;
4. explicit unresolved relations, bounds, and continuation/completion facts;
5. visibly escaped, structurally framed untrusted external text.

Aliases reduce repetition but are not accepted by commands. Machine consumers
reuse the canonical value from the dictionary unchanged.

### Layer changes

- Domain: Chatwork-neutral identity/reference, room, participant, message,
  relation, task, file, invite, request, coverage, mutation-impact values.
- Application: resource/task use cases and minimal ports; deterministic joins,
  filtering, bounded composition, and secret-free authentication binding.
- Infrastructure: process-environment PAT source, public-client OAuth library,
  OS credential store, binding manager, bounded Chatwork HTTP transport, wire
  DTOs, notation parser, multipart/form adapters, and stable fault mapping.
- CLI/catalog: authentication-profile discovery/lifecycle, deterministic method
  selection, task arguments, exact reference parsing, candidate-C rendering,
  composition root, complete agent contracts, and coverage registration.

### Data and control flow

```text
argv -> catalog contract -> exact references / typed intent
     -> exact PAT/OAuth2 selection -> authentication gate -> ephemeral binding
     -> application task -> bounded Chatwork port
     -> infrastructure resolves binding -> HTTPS request
     -> wire validation / notation parsing -> typed result
     -> application task semantics -> context capsule -> stdout
```

Mutations insert the shared execution invoker and resource-specific policy
between typed intent and the infrastructure request.

### Error and cancellation behavior

- Invalid input, missing auth, binding mismatch, policy rejection, and canceled
  preflight cause zero provider task calls.
- 401, 403, 404, 429, provider unavailability, malformed responses, and bounds
  map once to stable public faults without bodies or token-bearing causes.
- Every provider operation has one transport attempt; callers use typed
  recovery metadata rather than an automatic transport retry.
- Metadata/read and non-upload operations time out after 20 seconds; upload
  after 60 seconds. Success/error bodies are limited to 8 MiB/64 KiB, output to
  16 MiB, aggregate lists to 10,000, the five documented lists to 100, and
  upload input to 5 MiB.
- A structured provider mutation result is
  authoritative; any unclassified post-send result is non-retryable and points
  to an exact read-only reconciliation task.
- No partial stdout is a successful result.

### Mutation confirmation

- Exact typed invocation suffices for ordinary creates and updates.
- Room creation, member replacement, invite-link creation/update, and incoming
  request acceptance require exact `--confirm=access-change`.
- Room leave/delete, message deletion, invite-link deletion, and incoming
  request rejection require exact `--confirm=destructive`.
- Confirmation is scoped to one invocation and never reused or inferred.

### Security and public boundary

- PAT enters only through the selected environment variable. OAuth callback,
  code, state, verifier, tokens, refresh source, and store handle remain inside
  infrastructure; persisted token material uses only the OS credential store.
- API tasks require exact `CWK_AUTH_METHOD=pat|oauth2`; no method probing or
  fallback is allowed.
- OAuth is one public client with state, PKCE S256, a registered non-HTTP custom
  redirect, manual full-callback stdin, fixed Chatwork endpoints, and no client
  secret or `offline_access`.
- Production requests cannot redirect credentials away from the fixed Chatwork
  HTTPS origin.
- Local-server injection exists only in internal constructors used by tests.
- File input and provider responses have explicit byte ceilings.
- All fixtures are synthetic, publishable, and digest-pinned.

## Implementation slices

1. Governing contract, fixed operation manifest, coverage checker, and failing
   tests.
2. Shared PAT authentication, transport, fault, bounds, fixture, and
   context-capsule foundations.
3. Public-client OAuth profile discovery/login/status/logout, OS credential
   storage, refresh/revalidation, deterministic method selection, and secret
   canary tests.
4. Identity/contact/room discovery and exact read tasks, including recent room
   messages.
5. Remaining reads and all create/write/delete tasks with intent policy and
   reconciliation.
6. Complete catalog/help/reference graph, 32-operation local E2E matrix,
   hostile-output and agent-readiness validation.
7. Full, security, and public gates; evidence closure.

The final closure step changes the manifest from `coverage_status: planned` to
`complete`; contractlint then rejects any one of the 32 operations without a
public capability owner.

## Verification

- Unit/contract tests at each of the four layers.
- Local HTTP server fixtures for exact method, path, form/multipart fields,
  token header presence without disclosure, and stable fault mapping.
- Coverage check reporting exactly 32/32.
- Opaque-reference producer/consumer and byte-preserving round trips.
- Mutation zero-attempt and uncertain-outcome tests.
- Context-capsule semantic answer, determinism, hostile-text, writer-failure,
  and no-alias-as-identity tests.
- Agent transcript for room discovery/message retrieval and representative
  create/write/delete recovery.
- `task check`, `task security`, and `task public:check`.

## Rollout and rollback

Before `1.0`, the new public surface may replace the sample commands in one
reviewed change. Rollback is source rollback only; `cwk` creates no local state.
Remote mutations are never rolled back automatically. Their read-only
reconciliation commands remain available whenever the mutation is public.

## Documentation promotion

Before implementation exposure, promote fixed exhaustive snapshot coverage,
candidate-C selection, PAT/OAuth selection and storage policy, production
destinations, numeric call bounds, mutation confirmation policy, and
fixture/coverage enforcement to
the governing theses, product, architecture, security, harness, authentication,
external-contract, and agent-readiness documents.
