# Context: Message period selection

## Verified repository facts

- `.harness/project.json` has profile `ready`.
- `messages list` currently makes one provider request using only Chatwork's
  documented `force` query and receives at most 100 messages.
- Existing `--start-index` and `--count` are local ranks over that one result;
  they do not page or fetch older history.
- Application-owned selection already clears `MessageFilter` before the port,
  resolves typed reply edges, retains provider sequence/order, and returns
  selection provenance.
- Existing composition is sender OR -> typed-time rank -> index/count -> direct
  one-hop reply context.
- Message `send_time` is a provider-independent Unix-second semantic fact.
- The current CLI has no injected general-purpose command clock. Authentication
  owns a separate clock that must not become message-selection policy.
- The product thesis selects Japanese user-facing prose and names developers
  and operators in Japan as the primary user. It does not currently define a
  general runtime time-zone switch.

## User evidence

- The motivating RQ1 and RQ2 were answerable from the afternoon of 2026-07-17,
  while each evaluated method loaded a source window spanning roughly
  2026-07-09 through 2026-07-18.
- Deterministic local period selection can remove unrelated records without
  spending model tokens on date classification, like the already accepted
  sender and index selectors.
- The owner additionally identified `today` and `yesterday` as useful task
  vocabulary so an agent need not calculate a calendar date before invocation.

## Constraints

- The feature can reduce rendered bytes and model input, but cannot reduce the
  one provider response or discover a date outside the returned 100 messages.
- Yearless dates, offset-free timestamps, inclusive `until`, host-local time,
  and a relative day resolved more than once would be ambiguous.
- `today` and `yesterday` need a clock. Production must inject `time.Now`; tests
  use fixed clocks. A missing clock must fail before authentication/I/O.
- `Asia/Tokyo` is fixed for the day shorthand in this slice. Explicit interval
  bounds carry their own RFC3339 offsets. An arbitrary time-zone selector is
  deferred until user evidence justifies its contract and tzdata dependency.
- Context records can fall outside the primary period only when the caller
  explicitly requests `--context replies`; output must keep anchors distinct.
- Filtering raw body text, display names, provider order, or presentation text
  is forbidden.

## Baseline observation

- `cwk help messages list --format agent` declares room/window, sender,
  start-index/count, and direct reply context but no period input.
- The initial sandbox command used the host Go cache and failed before running.
  Verification succeeds with the existing Go 1.26.5 toolchain and a writable
  `/private/tmp` build cache; this is an execution-environment fact, not a
  repository failure.

## Decisions

- Public syntax is `--since <RFC3339>`, `--until <RFC3339>`, and
  `--on <YYYY-MM-DD|today|yesterday>`.
- `since` is inclusive and `until` is exclusive. RFC3339 values must contain
  an explicit `Z` or numeric offset and no fractional seconds.
- `--on` resolves to the half-open Tokyo interval `[00:00, next 00:00)`.
- Relative input is normalized before the typed application request; semantic
  result metadata records the effective date, zone, and Unix bounds rather
  than the unstable word `today` or `yesterday`.
- Primary predicates are `(sender in requested senders OR no sender filter)`
  AND `(send_time in requested interval OR no period filter)`.
- Candidate count is after all primary predicates and before index/count.
- No new command, capability ID, authentication requirement, provider call,
  destination, dependency, persistence, or side effect is introduced.

## Implementation and measurement evidence

- Domain invariants own positive half-open bounds and exact fixed-Tokyo day
  correspondence. CLI fixed-clock tests resolve `today` and `yesterday` once.
- Application tests prove sender AND period membership before rank/index,
  exact lower/upper boundaries, explicit out-of-period reply context, source
  order, anchor provenance, one provider call, and a cleared provider filter.
- CLI/catalog tests reject ambiguous, offset-free, fractional, conflicting,
  empty, and reversed periods before authentication/I/O and expose effective
  bounds/day/zone in the selection prelude.
- The active 100-message fixture selects 40 in-day anchors, retains the
  decision/owner/deadline answer key, and uses one provider call with zero
  external post-processing.
- Pinned `tiktoken==0.13.0` `o200k_base` measurement reduces the frozen output
  from 2,940 to 1,381 tokens (53.0%) and 12,876 to 5,718 bytes (55.6%). Exact
  hashes and reproduction are in [token-measurement.md](token-measurement.md).
- The final full `task check` passed on 2026-07-20. Review confirmed that the
  effective period remains local typed selection, the adapter still emits only
  the documented `force` query, unfiltered output goldens remain unchanged,
  and the public documentation does not imply provider filtering or historical
  paging.
