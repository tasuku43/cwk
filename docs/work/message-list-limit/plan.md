# Work Plan: Bounded message selection

- Status: Completed
- Owner decision: The explicit request to implement a message count limit
  approves this finite interface; adjacent presentation exploration remains out
  of scope.

## Chosen approach

1. Add `Limit int` to the existing typed `MessageFilter`; zero means absent.
2. Declare optional `--limit <count>` in the `messages list` catalog entry and
   parse it separately from the room-task Unix deadline.
3. Validate 1..100, normalize active selection context to `none`, and bind
   request/result equality through the typed filter.
4. Clear the complete local message filter before the provider port and make
   infrastructure reject any leaked sender/context/limit policy.
5. Resolve the full provider window, compute sender/all candidates, select the
   newest N by typed `send_time` with provider-position tie-break, add explicit
   direct reply context, and re-resolve only the displayed subset.
6. Extend selection provenance with the candidate count. Preserve source count,
   source sequences, anchor sequences, provider order, and canonical refs.
7. Keep provider coverage at 100, render it as `source-limit`, and render
   candidate/selection limit facts only when the new limit is active.
8. Add boundary tests, update durable contracts and the active readiness probe,
   then run the canonical gate and commit.

## Alternatives considered

### Send `limit` to Chatwork

Rejected. The reviewed endpoint documents only `force`; an undocumented query is
not a product contract and would move task policy into infrastructure.

### Slice the last N response-array records

Rejected. The official endpoint does not guarantee array direction, so provider
tail is not a sufficient proof of newest messages.

### Sort output by timestamp

Rejected. Existing output promises provider order and source sequences. Typed
timestamps select the anchor set without changing physical output order.

### Limit before sender selection

Rejected. `--sender A --limit 10` should mean A's newest ten messages in the
bounded provider window, not A's subset of the room's newest ten records.

### Cut reply context back to N

Rejected. It would remove the explicit context the caller requested and could
silently break relationship understanding. Context is labeled as additional to
the primary anchor limit.

## Risks and controls

- **Two meanings of limit:** rename message provider coverage to
  `source-limit`; keep requested limit in selection metadata.
- **Limit hides oversized provider response:** validate source cardinality
  against coverage before selection.
- **Equal timestamps:** choose later provider position deterministically and
  test it.
- **Limit-excluded matching message reappears as context:** retain it as a
  non-anchor and validate context from typed reply edges, not sender equality.
- **Local policy leaks into HTTP:** clear it in application and reject it in
  adapter request construction.
- **Unexpected output growth:** default context remains `none`; scoped help
  states that `replies` may exceed the primary-message limit but remains inside
  the 100-message source.

## Verification

- Domain truth tables for zero/1/100/101, filter equality, source/candidate/
  anchor invariants, and the unfiltered 101-message rejection.
- Application fixtures for unsorted timestamps, equal timestamps, sender OR,
  limit-only, sender+limit, direct reply expansion, context re-entry, empty and
  fewer-than-N sources, unresolved omitted parents, and one port call.
- CLI tests for usage/help, parsing, duplicate and malformed pre-auth failures,
  room-task deadline regression, projection, hostile text, and canonical refs.
- Infrastructure tests proving only documented `force` is emitted and leaked
  local selection fails.
- Agent-readiness scenario for latest two in one command with zero external
  processing.
- `task check`.

## Rollback

Remove the catalog input, typed filter limit/candidate provenance, application
selection branch, projection metadata, and their tests together. Preserve the
independent fix that rejects provider results exceeding declared coverage.
