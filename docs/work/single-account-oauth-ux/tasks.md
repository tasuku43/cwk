# Tasks: One-command single-account OAuth login

## Decisions and contracts

- [ ] Amend the fixed-single-account thesis and reference axiom.
- [ ] Amend product, architecture, security, authentication, external API, ADR,
      harness, and agent-readiness contracts.
- [ ] Add and test the command-bound local singleton catalog contract.

## Infrastructure

- [ ] Implement strict non-secret platform-user configuration storage.
- [ ] Implement bounded shell-free default-browser opening with manual fallback.
- [ ] Compose public-config persistence with OAuth credential persistence and
      deterministic reconciliation.
- [ ] Resolve explicit environment PAT selection or stored OAuth selection
      without probing/fallback.

## Public authentication workflow

- [ ] Remove `auth profiles` and all public profile references.
- [ ] Make first login accept only optional first-run `--client-id` plus stdin
      callback; later login accepts no client ID.
- [ ] Make status/logout target the declared singleton with no arguments.
- [ ] Remove required OAuth registration/method exports from OAuth task help.
- [ ] Keep tokens and callback/code/verifier material out of config, argv,
      output, errors, fixtures, and logs.

## Verification and closure

- [ ] Update README and runnable agent-help snapshots/transcripts.
- [ ] Run focused tests and cross-platform checks.
- [ ] Run `task check:fast`.
- [ ] Run `task check`.
- [ ] Run `task security`.
- [ ] Run `task public:check`.
- [ ] Record evidence, mark the work packet complete, and commit in reviewed
      product/contract, implementation, and closure increments.
