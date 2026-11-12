# Work Tasks: Message send reconciliation

## Understand

- [x] Read repository policy, add-capability skill, feedback #4, and selected contracts.
- [x] Inspect send body expansion, semantic message parsing, authentication binding, bounded list, output projection, and recovery grammar.
- [x] Confirm design with the owner and coordinate shared worktree files.

## Implement

- [x] Add the provider-independent reconciliation task and body-free result vocabulary.
- [x] Add the application use case with optional exact reply read, one latest list read, same binding/context, and no mutation path.
- [x] Add an infrastructure-owned one-use expanded-body matcher using the existing send builder and bounded body-file reader.
- [x] Add CLI catalog, request parsing, stdin/body-file behavior, exact error recovery, and active-view closure.
- [x] Add body-free candidate output with hostile-name escaping and complete bounded-source metadata.
- [x] Add focused domain, app, infra, CLI, catalog, output, and agent-readiness tests.
- [x] Promote the contract to durable docs and the add-capability skill after shared-file handoffs.

## Verify and hand off

- [x] Run focused tests for affected packages and report results.
- [x] Review all diffs for preserved concurrent changes and public-boundary leaks.
- [ ] Remove this temporary packet after Terra review verifies the documented application boundary.

## Focused verification evidence

- `go test ./internal/domain/chatwork ./internal/app/chatworkcmd ./internal/infra/chatworkapi ./internal/cli/capsule ./internal/cli ./tools/presentationeval` passed after the quote-candidate changes were integrated.
- The set includes a local HTTP server scenario proving one body-file read before one exact reply GET and one `force=1` list GET, zero POSTs, and no body/path in the result.
