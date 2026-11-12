# Work Plan: Feedback #17 error guidance

- Status: Proposed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Classify read transport failures at the infrastructure boundary with explicit precedence: caller cancellation, deadline, then ordinary unavailability. Preserve `operation_canceled` kind/code/exit for deadlines but use a fixed timeout-specific Japanese message only when the context or transport error matches `context.DeadlineExceeded`. Keep the existing mutation branch first so mutation uncertainty never becomes a retryable timeout. Add a task-based room-target predicate for the fixed 403 membership possibility phrase. Let the Chatwork confirmation policy produce a safe structured refusal only for the two known values, and let the generic invoker preserve that valid policy fault while retaining its existing fallback for ordinary policy errors.

## Alternatives considered

### Add a separate timeout fault code

Rejected because the requested text correction can distinguish a deadline from cancellation while preserving the existing public `operation_canceled` kind/code/exit contract.

### Include the confirmation cause in the generic invoker message

Rejected because a generic policy error may contain arbitrary user or private data. Use only a fixed allowlist value from the command's catalog contract.

### Echo provider 403 text

Rejected because provider content is untrusted and could disclose private details or hostile terminal text. Use a fixed Japanese explanation.

## Data and control flow

Provider or transport error -> infrastructure fixed fault (`chatwork_unavailable`, `operation_canceled`, or existing mutation unknown) -> application/public-copy boundary -> catalog recovery -> CLI output. Read deadlines use the existing `operation_canceled` machine contract with timeout-specific Japanese message text. Mutation policy mismatch -> fixed allowlisted `mutation_rejected` -> invoker preserves valid fault -> catalog recovery; authentication and provider actions remain uncalled.

## Verification

- Infrastructure unit tests distinguish context deadline, wrapped deadline error, caller cancel, and ordinary I/O failure.
- Provider mapping tests distinguish room-target and non-room-target 403 messages and reject private provider-body leakage.
- Application/invoker and CLI tests prove exact confirmation value, no user-value/cause leakage, stable exit and recovery, and zero auth/provider calls.
- Run focused tests for the edited packages; no repository-wide gate is in this packet's scope unless selected by the root agent.
