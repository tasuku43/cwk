# Work Tasks: Typed send body-file preflight

## Implement

- [x] Add the narrow body preflight port and opaque prepared-body handle.
- [x] Read and validate body files before the reply GET without rereading or exposing filesystem access to application code.
- [x] Clear handle bytes, path-bearing source, binding, and adapter ownership after consume/discard.
- [x] Preserve exact form prefix/suffix, same context/binding, and single POST behavior.
- [x] Add invalid-source zero-call, cleanup/redaction, failure-recovery, and lost-response tests.
- [x] Update governing architecture, security, API, harness, readiness, and capability-skill contracts.

## Verify and hand off

- [x] Focused suites pass: `go test ./internal/app/chatworkcmd ./internal/infra/chatworkapi ./internal/cli ./internal/domain/chatwork`.
- [ ] Root-selected repository gate passes.
- [ ] Terra review accepts the P2 fix.
- [ ] Remove this temporary packet when the parent change completes.
