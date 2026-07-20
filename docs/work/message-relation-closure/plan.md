# Plan: Self-contained bounded message relations

## Chosen approach

1. Add pure domain values for a relation-fetch budget, per-target resolution
   state, context provenance, and period reachability.
2. Keep the list request plus exact-message reads behind the existing
   application port and one admitted authentication binding.
3. Assemble the selected source result first, then resolve its explicit missing
   reply targets breadth-first. Reuse original-source messages before spending
   the exact-fetch budget and enqueue parents found in supplemental context.
4. Preserve source messages/sequences unchanged and publish supplemental
   relation context separately.
5. Derive oldest-boundary reachability from the unfiltered typed source only
   under the trustworthy recent-window conditions.
6. Extend the catalog, current text projection, durable contracts, and active
   synthetic readiness probes together.

## Alternatives rejected

- An undeclared implicit extra call: the selected default-five budget is instead
  part of scoped help and every result's fetch-limit/attempt evidence.
- A boolean resolution flag with an unstated limit: bounded in code but not in
  user intent or agent planning.
- Merging fetched records into source sequences: fabricates membership in the
  provider list response and corrupts count/index provenance.
- Treating an old-period zero as ordinary empty: repeats the T2 exploration
  failure.
- Treating an unavailable exact target as complete absence: hides deletion or
  access evidence and overstates relation closure.

## Risks and mitigations

- Provider-call amplification: explicit 1..100 budget, unique-target
  deduplication, one attempt per target, exact attempt reporting.
- Partial success after transient failure: only reviewed not-found/restricted
  target outcomes are retained; every other fault aborts the whole task.
- Unbounded recursive expansion: a 0..100 exact-fetch budget, breadth-first
  queue, and visited target set bound every chain, branch, duplicate, and cycle.
- False reachability: classify only recent, non-limited, nonempty sources;
  otherwise emit unknown.
- Presentation confusion: keep source records and supplemental context in
  separate declared sections with exact canonical references.

## Verification

- Domain invariants and reachability truth table
- Application orchestration, deduplication, breadth-first order, call-budget,
  recursion, and cycle tests
- Exact show result-binding and adapter request-boundary tests
- CLI parsing, pre-auth rejection, help, output, and fault behavior tests
- T1/T2 no-post-processing readiness fixtures
- `task check:fast`, then `task check`
