# Goal: Message period selection

## Outcome

An agent can ask `messages list` for only the messages inside one explicit
time interval or one Japanese calendar day, including `today` and `yesterday`,
without loading the provider's whole returned window into model context or
performing external date filtering.

## Non-goals

- Provider-side date filtering or fewer provider response bytes.
- Reading beyond Chatwork's single maximum-100 message window.
- Provider pagination, local message history, archive import, or backfill.
- Thread traversal, missing-relation fetches, cross-room following, full-text
  search, system-message classification, or body compression.
- A runtime locale or arbitrary time-zone selector.

## Acceptance criteria

- `messages list` accepts inclusive `--since <RFC3339>`, exclusive
  `--until <RFC3339>`, and `--on <YYYY-MM-DD|today|yesterday>`.
- `--on` is mutually exclusive with `--since` and `--until`; two explicit
  bounds must form a non-empty interval.
- RFC3339 bounds require an explicit offset and whole-second precision.
- `--on` uses the fixed `Asia/Tokyo` calendar contract. Relative days resolve
  once from an injected clock before authentication or provider I/O, and the
  result exposes the effective date and Unix interval.
- Exact sender OR and the period predicate form the primary candidate set;
  start-index/count apply next; direct reply context applies last and may add
  records outside the period while anchors remain explicit.
- The application clears every local period field before the one existing
  provider call, preserves source order and canonical references, and rejects
  invalid input before authentication/I/O.
- Unfiltered output remains byte-identical. Filtered output declares source
  count, effective bounds, candidate count, anchors, and context provenance.
- A publishable synthetic readiness fixture shows that a one-day request keeps
  the required answer facts with zero external post-processing and materially
  fewer input tokens than the same 100-message source window.
- `task check` passes.
