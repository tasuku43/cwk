# Work Goal: Self-contained bounded message relations

- Status: Complete
- Owner: Product and implementation owner
- Target: Current work packet
- Related ADRs: None

## Outcome

`messages list` spends a declared default-five exact-message fetch budget to
recursively complete same-room reply context that is absent from its provider
window, while reporting every attempted, resolved, unavailable, and skipped
target. The same result reports whether a requested period is older than the
reachable start of a trustworthy recent source window, so an agent can stop
instead of probing more dates.

## Why now

The T1 evaluation lost the Aurora decision because replies were visible while
their two out-of-window parents remained unresolved. The T2 evaluation spent
about twenty tool calls trying dates that the latest-100 list operation could
not reach. Both failures are evidence that a supported bounded read must be
self-contained for known references and self-report its discovery boundary.

## Non-goals

- Discovering arbitrary non-referenced history older than the provider window
- Following cross-room references or continuing after the declared fetch budget
- Resolving quote records that do not carry a canonical message identity
- Display-name sender lookup
- Quote, system-message, URL, attachment, or body compression
- Provider pagination or undocumented date/search query parameters

## Acceptance criteria

- [x] `--resolve-relations <count>` accepts 0..100, defaults to five additional
  exact-message slot, and uses zero as the explicit opt-out.
- [x] Only unique explicit same-room unresolved reply targets attached to the
  displayed bounded result are eligible, in deterministic source order.
- [x] A target already present in the original source window is included as
  source context without consuming the fetch budget; an absent target consumes
  at most one exact-message request; newly attached context recursively queues
  its explicit same-room parent until the finite budget or a visited target.
- [x] The typed result distinguishes source-resolved, fetched, not-found,
  restricted, and budget-exhausted outcomes and reports fetch limit/attempts.
- [x] Cancellation, rate limiting, unavailable responses, malformed results,
  and unrelated permission failures return no partial successful result.
- [x] A recent, non-limited, nonempty source reports its oldest reachable
  message/time. Period selection reports within, partial-before, or wholly
  before that boundary; differential, limited, and empty windows report
  unknown rather than guessing.
- [x] T1 and T2 synthetic readiness probes use zero external post-processing;
  T1 needs one list invocation and bounded recursive internal exact fetches, while T2
  stops after one list invocation.
- [x] Catalog help states provider-call bounds and the latest-100 limitation.
- [x] `task check` passes. The commit is recorded immediately after closing
  this work packet.

## Governing documents

- Thesis: operational closure, typed semantics before presentation, executable claims
- Product contract section: filtering and task composition
- Architecture or security invariant: application-owned orchestration, bounded fixed-destination reads, no fabricated relations
- Existing ADR: 0003 Chatwork PAT only

## Completion definition

The work is complete when the public contract, four-layer implementation,
typed negative paths, current projection, T1/T2 readiness evidence, durable
documents, and repository gate agree, and the reviewed change is committed.
