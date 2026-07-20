# Work Plan: Message index selection

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Replace message `--limit` with optional one-based `--start-index` and maximum
`--count`. Rank sender-matched candidates newest first, take the requested
ordinal slice, add direct reply context, then restore provider order for output.

## Alternatives considered

- Generic offset/limit: rejected because `limit` is often confused with an end
  position and offset may be zero- or one-based.
- Provider cursor: rejected because Chatwork exposes none.
- Exact `--before <message-ref>` boundary: potentially more mutation-tolerant,
  but it complicates equal-time/sender semantics and does not solve the
  maximum-100 source boundary.

## Public contract

- `messages list`, `RoleAct`, `EffectRead` remains unchanged.
- `--start-index <index>` and `--count <count>` are optional integers 1..100;
  count alone defaults start index to 1.
- Composition: sender OR -> typed newest rank -> start/count -> direct one-hop
  reply context.
- Result facts: source/candidate count, start index, requested count, actual
  items per page, optional next start index, source/anchor sequences.
- The catalog result stays one complete bounded local task result over an
  incomplete maximum-100 provider window; it has no public pagination binding.
- Separate calls are stateless and do not promise rank stability.

## Layer changes

- Domain validates index/count and result provenance.
- Application selects ordinal membership and computes next start index.
- Infrastructure rejects leaked local policy.
- CLI/catalog parse, document, declare, and render the new interface.
- Readiness proves first/next slices without external processing.

## Verification

- Domain/application: first page, continuation, count-only default, start-only,
  short/empty sources, sender composition, ties, context beyond count.
- Negative: malformed/duplicate/out-of-range pre-auth failure and adapter leak.
- Presentation/catalog: usage, scoped help, output fields, canonical refs.
- Readiness: two commands, two provider calls, no external parser.
- Required gate: `task check`.

## Rollout and rollback

This is an intentional pre-1.0 breaking message-input change. Room-task
`--limit` is unrelated and remains. Rollback must restore parser, catalog,
domain, application, presentation, documentation, and tests together.
