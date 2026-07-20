# Work Tasks: Message index selection

## Understand and decide

- [x] Read governing documents and `$add-capability`.
- [x] Observe newest-N behavior and provider boundary.
- [x] Review SCIM index and cursor specifications.
- [x] Select one-based start index plus maximum count.
- [x] Keep one bounded provider request per invocation and no public pagination.

## Implement

- [x] Add domain/application index-selection tests and invariants.
- [x] Add adapter non-leakage guard and tests.
- [x] Update catalog, parsing, declared output, and presentation.
- [x] Update readiness fixture and no-post-processing evidence.
- [x] Promote durable documentation, README, and capability skill.

## Verify

- [x] Focused tests pass. Evidence: domain, application, CLI, and
  `tools/presentationeval` packages pass on 2026-07-20.
- [x] `task check` passes. Evidence: full gate with Go 1.26.5 on 2026-07-20,
  including race, security, release, and public checks.
- [x] Runtime scoped help exposes the exact interface. Evidence:
  `go run ./cmd/cwk help messages list --format agent` declares both flags,
  1..100 bounds, and the ranks-11-through-30 example.
- [x] Continuation scenario meets its tool/provider-call budget. Evidence:
  `TestActiveMessageIndexScenarioUsesTwoCommandsWithoutPostProcessing` and
  `TestActiveMessageIndexContinuesWithoutRepeatingEarlierRanks`.
- [x] Final diff and repository status are understood. Evidence: only the
  message index capability, governing documents, and its work packet changed.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Temporary diagnostics and sensitive artifacts are absent.
- [x] Handoff explains provider `source-limit` and cross-call rank instability.
