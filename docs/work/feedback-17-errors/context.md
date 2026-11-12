# Work Context: Feedback #17 error guidance

## Verified current behavior

- `internal/infra/chatworkapi/transport.go:transportFault` currently groups `ctx.Err()`, `context.Canceled`, and `context.DeadlineExceeded` as `operation_canceled` for reads; non-context errors become `chatwork_unavailable`.
- The same helper returns the existing mutation unknown-outcome fault before inspecting cancellation. This behavior must remain unchanged after dispatch.
- Provider 403 mapping currently uses one fixed generic permission message. The bounded provider error body is not included in the public fault.
- `internal/app/execution.Invoker` wraps policy refusal as `mutation_rejected` with a generic public message, while `chatworkcmd.ConfirmationPolicy.Check` has the expected value and does not currently reflect `Provided`.
- `internal/cli/chatwork_catalog.go` declares `access-change` and `destructive` on the exact mutation commands that require them; catalog error normalization supplies recovery metadata.
- The shared worktree has concurrent edits in transport, catalog, runtime tests, docs, and the add-capability skill. Avoid replacing or reverting unrelated changes.

## Constraints

- Read deadline classification tests must cover `ctx.Err()` and returned `err` sentinels, plus caller cancellation and non-context I/O failure.
- Mutations retain their post-dispatch unknown outcome and pre-dispatch proof contract; no timeout can establish non-dispatch.
- The 403 text must be a fixed safe Japanese phrase and selected only for an explicit room-target task set.
- Confirmation text may include only the fixed catalog values `access-change` or `destructive`; never interpolate argv, causes, provider bodies, or arbitrary `Required` values.
- All failure recovery remains catalog-derived and exact.

## Decisions

- Preserve the public `operation_canceled` code, kind, retryability, and exit mapping for read deadlines; use a fixed Japanese timeout message only when the context or transport error matches `context.DeadlineExceeded`.
- Treat a context explicitly canceled by its caller as cancellation even if a separate wrapped cause is also present.
- Keep 403 room-membership wording scoped to tasks whose selected target or room scope is explicit in the fixed task request.

## Thesis evidence

- Repeated confusion: timeouts and caller cancellation currently share `operation_canceled`, despite different recovery meaning.
- User outcome: a deadline is described as a timeout while retaining the existing cancellation machine contract; ordinary I/O failure remains unavailable.
- Implementation exception: none; classify only typed context sentinels at infrastructure.
- Thesis consequence: Axiom 7 already requires errors to preserve stable recovery semantics; no thesis revision is needed.

## Security and boundary notes

- No new external destination, dependency, credential, or fixture.
- Provider error bodies and transport causes remain private.
- Exact confirmation wording is produced only after policy rejection, from a strict two-value allowlist.
