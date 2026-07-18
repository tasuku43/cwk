# Work Tasks: Message sender selection

## Understand

- [x] Read governing documents and `$add-capability`.
- [x] Observe current scoped help and implementation flow.
- [x] Record provider, semantic, presentation, and security constraints.
- [x] Confirm public outcome and closed non-goals.

## Decide

- [x] Compare sender-only, boolean-related, and typed-context interfaces.
- [x] Select repeatable sender OR plus `context=none|replies`.
- [x] Keep read effect, exact room target, PAT boundary, and one provider call.
- [x] Classify this as an extension of public `chatwork.messages.manage`.
- [x] Fix source-sequence, anchor, omission, and compatibility semantics.

## Implement

- [x] Add domain filter and selection invariants.
- [x] Add application selection and reply-context tests.
- [x] Move reply resolution from CLI into application outcome assembly.
- [x] Bind catalog, parser, help, and runtime behavior.
- [x] Preserve provider request and unfiltered output contracts.
- [x] Render filtered source sequence and selection metadata.
- [x] Add semantic/readiness fixture and hostile/round-trip tests.
- [x] Update durable documentation and skill guidance.

## Verify

- [x] Focused tests pass. Evidence: domain, application, infrastructure, CLI,
  capsule, and `tools/presentationeval` tests passed on 2026-07-19.
- [x] `task check` passes. Evidence: `task check` with Go 1.26.5 passed the
  full repository, race, vet, module, security, vulnerability, and release
  reproducibility gates on 2026-07-19.
- [x] Agent-readiness scenario meets one-command/zero-processing budget.
  Evidence: `TestActiveMessageSenderSelection*` fixes one `messages list`
  invocation, one provider task call, zero external processing, the semantic
  answer key, and exact canonical next-command input.
- [x] Final diff and repository status are understood. Evidence: changes are
  limited to typed selection, application assembly, catalog/parser/presentation,
  adapter guard, synthetic evidence, and governing documentation.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions are promoted.
- [x] Temporary diagnostics and sensitive artifacts are absent.
- [x] Work packet is closed and committed.
