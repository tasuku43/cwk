# Work Tasks: Feedback #17 error guidance

## Understand

- [x] Read the governing docs, add-capability skill, and feedback attachment.
- [x] Inspect current transport, catalog, policy, and runtime error paths.
- [x] Coordinate shared-file scope with the root agent and #12 implementer.

## Implement

- [x] Add deadline-specific read wording only for typed deadline errors while preserving `operation_canceled`; retain caller-cancellation wording, ordinary I/O unavailability, and mutation taxonomy.
- [x] Add room-target-only fixed permission guidance.
- [x] Surface only the exact known confirmation value after actual policy refusal.
- [x] Add focused unit, CLI, and transport tests, including hostile value/body cases.
- [x] Update durable contract, harness, and agent-readiness wording as needed.

## Verify and hand off

- [x] Focused tests pass and evidence is recorded. Evidence: `go test ./internal/app/chatworkcmd ./internal/app/execution ./internal/infra/chatworkapi ./internal/cli -run 'Test(ReadDeadline|ReadPortDeadline|ReadTransportFault|ProviderForbidden|RunChatworkPreservesTimeout|RunChatworkConfirmation|ContactRequestAcceptance|ConfirmationPolicy|InvokerPreservesSafeStructured|InvokerKeepsCancellation)' -count=1` passed on 2026-11-08.
- [ ] Same reviewer receives concrete findings and test commands.
- [ ] Temporary packet is removed before final tree.
