# Work Tasks: Remove non-actionable success-output metadata

## Understand

- [x] Read governing theses, product, architecture, security, harness,
  authentication, external-API, and agent-readiness documents.
- [x] Confirm repository profile is `ready` and main starts clean.
- [x] Record the owner's selected removals, finite scope, and non-goals.
- [x] Audit representative live read-only outputs without retaining private data.
- [x] Audit every success result variant and current active contract reference.

## Decide

- [x] Remove the schema/task preamble.
- [x] Remove the standalone coverage record and provider-oriented coverage kind.
- [x] Retain actionable limit/completeness/uncertainty on the closest task record.
- [x] Classify every remaining rendered field as keep/drop/conditional.
- [x] Record compatibility, safety, and historical-evidence boundaries.

## Implement

- [x] Add all-route and negative-canary tests.
- [x] Update the success renderer and current catalog declarations.
- [x] Update active synthetic fixtures/evaluation inputs.
- [x] Update governing documentation, Skill guidance, README, and migration note.
- [x] Commit in reviewable contract/implementation/documentation slices.

## Verify

- [x] Focused tests pass. Evidence: `go test ./internal/cli/capsule ./internal/cli ./tools/presentationeval`.
- [x] Representative live read-only commands match the reviewed output. Evidence: eleven account/room/message/task/file/invite read routes passed shape checks without retaining live output.
- [x] Agent discovery and interpretation stay within budget. Evidence: root and exact `messages list` schema-v3 help succeeded and declared the revised fields.
- [x] `task check` passes. Evidence: Go 1.26.5 full profile.
- [x] `task security` passes. Evidence: modules verified, security boundary clean, no vulnerabilities found.
- [x] `task public:check` passes. Evidence: public repoguard and contractlint passed.
- [x] Repository status and generated diff are understood. Evidence: implementation `4e6c5c2`, contract/docs `a201200`, and this closeout commit.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions were promoted out of the work packet.
- [x] Temporary diagnostics and sensitive artifacts were removed.
- [x] Follow-up work is explicit and does not reopen another optimization round.
