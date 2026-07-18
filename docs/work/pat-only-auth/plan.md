# Plan: PAT-only authentication

## Chosen approach

1. Change the durable product and security decisions from dual-method
   selection to one PAT source.
2. Remove OAuth lifecycle commands and their catalog capability.
3. Simplify production composition to construct `chatworkapi` directly from
   `CWK_API_TOKEN`.
4. Remove Chatwork-specific OAuth seams, configuration, browser opener,
   credential-store code, and unused dependencies.
5. Replace dual-method/fallback tests with PAT-only, missing-token,
   invalid-token, no-argv, no-persistence, and zero-provider-call tests.
6. Replay root/scoped help and run full, security, and public gates.

## Alternatives rejected

- Keep OAuth as an optional fallback: preserves the same discovery and support
  burden and leaves two authentication paths for agents to reason about.
- Register a custom URI handler: adds installation and platform-specific
  behavior to a CLI that does not otherwise need it.
- Host an HTTPS callback relay: creates a new remote service, privacy boundary,
  and availability dependency.
- Persist PAT in plaintext XDG configuration: violates the credential boundary.

## Risks and controls

- Removing catalog entries can leave stale recovery actions: whole-catalog
  validation and scoped-help tests must reject them.
- OAuth-only faults may remain declared on API tasks: contract tests must assert
  the exact PAT-specific recovery surface.
- Dependency cleanup may expose platform-specific build assumptions: full and
  security profiles plus release-target architecture lint cover them.
- Historic work packets describe completed OAuth work: retain them as dated
  evidence and clearly mark the durable ADR superseded rather than rewriting
  history.

## Verification

- `go test ./internal/domain/authn ./internal/app/authn ./internal/infra/chatworkapi ./internal/cli`
- `task check:fast`
- `task check`
- `task security`
- `task public:check`

All listed verification completed successfully on 2026-07-18. The public
agent-help replay additionally confirmed that the root index has no `auth`
namespace and scoped Chatwork help declares `methods: ["pat"]` plus the sole
`CWK_API_TOKEN` prerequisite.
