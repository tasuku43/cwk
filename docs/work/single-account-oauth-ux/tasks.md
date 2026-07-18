# Tasks: One-command single-account OAuth login

## Decisions and contracts

- [x] Amend the fixed-single-account thesis and reference axiom.
- [x] Amend product, architecture, security, authentication, external API, ADR,
      harness, and agent-readiness contracts.
- [x] Add and test the command-bound local singleton catalog contract.

## Infrastructure

- [x] Implement strict non-secret platform-user configuration storage.
- [x] Implement bounded shell-free default-browser opening with manual fallback.
- [x] Compose public-config persistence with OAuth credential persistence and
      deterministic reconciliation.
- [x] Resolve explicit environment PAT selection or stored OAuth selection
      without probing/fallback.

## Public authentication workflow

- [x] Remove `auth profiles` and all public profile references.
- [x] Make first login accept only optional first-run `--client-id` plus stdin
      callback; later login accepts no client ID.
- [x] Make status/logout target the declared singleton with no arguments.
- [x] Remove required OAuth registration/method exports from OAuth task help.
- [x] Keep tokens and callback/code/verifier material out of config, argv,
      output, errors, fixtures, and logs.

## Verification and closure

- [x] Update README and runnable agent-help snapshots/transcripts.
- [x] Run focused tests and cross-platform checks.
- [x] Run `task check:fast`.
- [x] Run `task check`.
- [x] Run `task security`.
- [x] Run `task public:check`.
- [x] Record evidence, mark the work packet complete, and commit in reviewed
      product/contract, implementation, and closure increments.
