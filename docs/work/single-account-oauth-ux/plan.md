# Plan: One-command single-account OAuth login

## Chosen approach

1. Amend the thesis and contracts so an exact command may bind one declared
   tool-owned local singleton without a discovery reference. Preserve the
   opaque-reference rule for every user-selected or remote target.
2. Add an executable catalog target contract for that singleton. Login uses it
   as the create scope; status reads it; logout writes it. Remove the synthetic
   public profile producer and profile values from domain/application output.
3. Add a bounded platform-user configuration record containing only schema,
   explicit method `oauth2`, public client ID, and fixed redirect URI.
4. Accept `--client-id` only when configuration is absent. Build the existing
   public-client OAuth manager from that input, open the authorization URL via
   an injected platform opener, and retain the complete callback stdin path.
5. Persist public configuration before consent and the OS credential only after
   validated exchange and identity verification. A partial local outcome leaves
   only reusable non-secret configuration, never triggers PAT fallback, and
   makes no provider task request.
6. Resolve API authentication from an exact environment selector when present;
   otherwise use the persisted explicit OAuth selection. Unknown/invalid
   sources fail closed.
7. Update catalog help, docs, ADR, readiness transcript, and tests, then run the
   full security/public gates.

## Alternatives rejected

- Keep the fixed `--profile`: preserves generic reference uniformity but adds
  a discovery and argument that distinguish no possible target.
- Require a separate `auth configure`: clearer mutation separation but adds a
  command to the first-use path the user explicitly wants shortened.
- Prompt interactively for client ID: shorter syntax but less deterministic
  for agents, automation, help, and recovery.
- OS custom-URI handler: removes the paste but adds persistent platform
  registration and a larger security/packaging boundary.
- Clipboard monitoring: removes a paste at the cost of ambient clipboard
  access and unclear failure behavior.
- Infer an available credential: violates explicit selection and makes
  OAuth/PAT behavior depend on hidden machine state.

## Risks and controls

- Generic singleton escape hatch: constrain it to one explicit target kind,
  stable ID, tool-local scope, and no simultaneous target references; keep all
  existing remote reference tests.
- Config/token partial persistence: order writes so no provider task can use a
  token without validated public configuration; expose recovery through
  read-only status and explicit logout/login.
- Browser command injection: no shell, exact HTTPS authorization origin/path,
  bounded URL, injected runner tests, and redacted errors.
- Authorization URL in opener argv: document the bounded residual exposure;
  it contains state/challenge but no code/verifier/token, and PKCE remains the
  interception control.
- Config tampering: bounded strict schema, exact method/redirect validation,
  restrictive permissions, atomic replacement, and no fallback on invalid
  content.

## Verification

- Focused domain/application/infra/CLI tests for every modified boundary.
- Cross-platform build/lint coverage for browser and configuration adapters.
- `task check:fast`
- `task check`
- `task security`
- `task public:check`
- Runnable scoped help and synthetic login/readiness transcript.
