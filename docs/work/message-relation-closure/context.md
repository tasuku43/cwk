# Context: Self-contained bounded message relations

## Verified facts

- Chatwork message list exposes only the provider's differential or forced
  latest window, with a documented maximum of 100 records and no cursor/date
  pagination.
- Chatwork exact-message read accepts a known room and message ID, so an
  explicit reply target can be retrieved without search or identity guessing.
- Infrastructure already parses reply notation into typed room/message
  references and distinguishes exact-message not-found from access restriction.
- Application already resolves same-window replies and adds direct in-window
  reply context after sender/period/index selection.
- Therefore `--count` alone does not lose a parent that remains in the original
  source when `--context replies` is active; an extra request is needed only
  when the target is outside the original source window.

## Constraints

- Additional calls must not be hidden or unbounded. The shortest command owns
  five declared default slots; each slot permits at most one exact-message
  request for one unique target, and explicit zero disables these calls.
- The list provider request remains filter-free except for documented `force`.
- Exact-message fetches reuse the admitted PAT binding, fixed destination,
  caller context, one-attempt policy, timeout, response bounds, and fault map.
- Only explicit reply IDs are fetchable. To, quote metadata, prose, names, and
  timestamps cannot create a fetch target.
- Fetched messages are context, not members of the source window, candidates,
  indexes, source sequences, or evidence of arbitrary history completeness.
- A source or fetched context message may enqueue its own explicit same-room
  reply parent. Breadth-first first-reference order and one visited set make
  branching, duplicates, and cycles deterministic; the fetch budget remains
  the hard recursion bound.
- Access limitation makes an oldest-visible timestamp insufficient to prove a
  reachable lower boundary.

## Selected contract

- Public input: `--resolve-relations <count>`, integer 0..100, optional, with a
  default of five and zero as the explicit opt-out.
- Deterministic target order: first appearance of unique unresolved reply
  targets in displayed provider order.
- Source-window target: include as relation context with provenance `source`;
  consume zero fetch slots.
- Out-of-window target: issue one exact-message read; success includes context
  with provenance `fetched`; `chatwork_not_found` and
  `chatwork_message_restricted` become typed per-target outcomes and consume a
  slot; all other faults fail the command without partial success.
- Recursive closure: relations found in included context enqueue one explicit
  same-room parent unless it is already available or visited. No other relation
  kind expands the queue.
- Reachability enum for a selected period:
  `within-reachable-window`, `partially-out-of-reachable-window`,
  `out-of-reachable-window`, or `unknown`.
- A trustworthy oldest boundary requires `recent`, no access limitation, and
  at least one source message. The oldest item is the minimum typed send time,
  with provider order breaking ties.

## Unknowns resolved by implementation evidence

- Existing result validation did not bind `messages show` to both the
  requested room and requested message. This packet adds that invariant before
  accepting an exact-fetch result.
- Supplemental context can remain outside source sequences by using dedicated
  `relation-context provenance=source|fetched` records and explicit
  `relation-gap` records. A displayed reply points to that canonical context
  with `reply=message-ref:<id>` rather than fabricating a local source sequence.

## Deferred quote-dedup boundary

Quote-body deduplication is a separate capability. A safe future rule may
replace a quote body only when its exact bytes match exactly one already
displayed message from the same room. Zero matches, multiple matches,
normalized-text matches, author/time resemblance, and cross-room material must
remain unchanged. The replacement must describe duplicate body provenance
(for example `duplicate-body=#N`), not `reply` or `re`, because a repeated body
does not establish a semantic relation.

## Verification evidence

- Focused domain, application, CLI, capsule, infrastructure-request, and
  presentation-evaluation tests pass with Go 1.26.5.
- The active T1 fixture follows `9001 -> 9002` recursively through the public
  default-five budget using one task invocation, three provider requests, and
  zero external processing calls.
- The active T2 fixture classifies 2026-07-08 as outside a source whose oldest
  reachable message is on 2026-07-17, using one provider request and no date
  probing.
- `task check:fast` and the full `task check` pass, including architecture,
  contract, security, vulnerability, public-boundary, release-lint, and
  cross-platform checks.
