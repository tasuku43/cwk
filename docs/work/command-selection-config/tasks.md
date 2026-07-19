# Work Tasks: Persistent command selection

## Understand and decide

- [x] Read the governing documents and `$add-capability` skill.
- [x] Inspect catalog routing, help projections, mutation contracts, PAT
  laziness, and the retired configuration store.
- [x] Record outcome, non-goals, completion, path, schema, effect split, and
  non-security semantics before implementation.
- [x] Audit storage, catalog/view, and selector UX in parallel.

## Implement persistence

- [x] Add and test the bounded command-selection domain profile.
- [x] Add and test the application load/save use case and explicit-save policy.
- [x] Add and test XDG/AppData strict JSON storage, Unix durable replacement,
  and the explicit Windows replace-existing limitation.

## Implement catalog and CLI

- [x] Declare always-on versus configurable leaves in the catalog.
- [x] Derive and validate the active catalog without weakening full validation.
- [x] Load the active view before help normalization/routing and preserve lazy
  authentication and test isolation.
- [x] Add catalog-complete `config show` and `config edit` contracts.
- [x] Implement deterministic show/selector output, atomic line parsing,
  dependency diagnostics, repair, save, cancel, EOF, and blocked-read handling.

## Verify contracts

- [x] Cover missing, saved, empty, stale, malformed, and newly-added profiles.
- [x] Cover root/namespace/exact/trailing human help and every agent-help view.
- [x] Cover recovery/workflow closure and namespace disappearance.
- [x] Prove disabled execution is `unknown_command` with zero PAT/provider I/O.
- [x] Prove re-enabling retains authentication and mutation confirmation.
- [x] Add hostile file/input/output and secret-canary coverage.
- [x] Update capability ledger, thesis, product, architecture, security, harness,
  readiness, README, and `$add-capability` guidance.
- [x] Run focused tests and `task check`.

## Hand off

- [x] Record verification evidence and mark acceptance criteria complete.
- [x] Obtain independent architecture, security, and UX review.
- [x] Commit the reviewed slices at intentional boundaries.

## Verification evidence

- Focused domain, application, infrastructure, and CLI tests pass, including
  race-enabled CLI/storage tests.
- Windows amd64 command-selection storage cross-compiles successfully.
- Synthetic CLI scenarios prove help/routing removal, zero PAT resolution,
  exact producer diagnostics, malformed-content repair, unsafe-storage refusal,
  pre-save cancellation, and confirmed-save behavior under late cancellation.
- The accepted incoming-contact-request confirmation policy is enforced after
  re-enabling; this corrected implementation drift without changing policy.
- Independent architecture, security/storage, and UX reviewers reported no
  remaining P1/P2 findings after fixes.
- `task check` passed the full lint, test, race, security, vulnerability,
  release, public-boundary, and contract gates.
