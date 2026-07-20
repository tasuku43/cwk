# Work Plan: Safe exact-command agent-help drilldown

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Promote agent help to schema version 4. Reuse the compact catalog index entry
for both root and namespace selectors, add a namespace scope marker only to the
latter, and reserve the existing complete scope document for exact commands.
Point both indexes directly to `commands[].path` and
`cwk help <exact-command> --format agent`.

## Alternatives considered

- Keep the payload and add a size warning: rejected because the answer-loop
  tool has already received the large output by the time it can act on an
  in-band warning.
- Reject namespace agent help: rejected because a compact namespace index is a
  useful bounded navigation surface and already has a canonical catalog
  selector.
- Retain schema version 3: rejected because changing namespace `view=scope`
  from complete contracts to an index is an intentional machine-contract
  compatibility change.

## Implementation

1. Add optional scope metadata to the compact index projection and route every
   non-exact selector to it.
2. Make index request metadata exact-command-only.
3. Update human navigation copy and the `help` command's catalog contract.
4. Add root/namespace/exact shape and forbidden-detail tests plus a large
   namespace size-growth test.
5. Update README investigation recipes and durable product, architecture,
   harness, and readiness contracts.

## Verification

- Focused: `go test ./internal/cli`
- Fast repository gate: `task check:fast`
- Completion gate: `task check`
- Manual size observation from the built command for root, `messages`, and
  `messages list` agent help.

## Rollout and rollback

This is an intentional pre-1.0 help-schema break recorded as schema version 4.
Rollback must restore the schema number, selector metadata, namespace shape,
human guidance, durable contracts, and shape/size tests together.
