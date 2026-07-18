# Work Tasks: Bounded message selection

## Understand and decide

- [x] Read governing repository documents and `$add-capability`.
- [x] Confirm the current command has no user count limit.
- [x] Verify the official provider query and 100-message boundary.
- [x] Audit CLI, application, domain, infrastructure, presentation, and tests.
- [x] Fix outcome, non-goals, ordering, composition, and completion criteria.

## Implement

- [x] Add and validate the typed message selection limit.
- [x] Add catalog/help and CLI parsing without breaking task deadlines.
- [x] Apply newest-N primary selection before optional reply context.
- [x] Preserve provider sequences/order, canonical refs, and unresolved edges.
- [x] Keep local selection out of the provider request.
- [x] Distinguish provider source bound from requested selection limit.
- [x] Reject oversized source results before selection.
- [x] Update durable documentation and the capability skill.

## Verify

- [x] Domain and application focused tests pass. Evidence: package tests and
  their race variants passed; newest-time, tie, context, empty/bounds, and
  oversized-source cases are executable.
- [x] CLI and infrastructure focused tests pass. Evidence: parsing/help,
  deadline regression, projection, filter non-leakage, and exact `force` query
  tests passed.
- [x] Active no-post-processing scenario passes. Evidence:
  `TestActiveMessageLimit*` passed with one provider call and zero external
  processing.
- [x] Runtime help exposes the exact interface. Evidence:
  `go run ./cmd/cwk messages list --help` shows optional `--limit <count>`, its
  1..100 bound, composition, and `--window recent` guidance.
- [x] Independent reviews have no blocking issue. Evidence: separate
  domain/application, CLI/presentation, and documentation reviews completed.
- [x] `task check` passes. Evidence: full gate passed with Go 1.26.5, including
  unit, race, security, vulnerability, release, and public checks.
- [x] Final diff and repository status are understood. Evidence: only this
  capability's implementation, tests, fixtures, governing docs, skill, README,
  and work packet are included.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Work packet status is complete.
- [x] No temporary, live-account, credential, or sensitive artifacts remain.
- [x] Commit records the bounded message-selection outcome and checks.
