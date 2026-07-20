# Plan: Message period selection

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

1. Extend the durable filtering contract with exact and relative period
   vocabulary, fixed Tokyo day semantics, composition order, and the explicit
   maximum-100 source limitation.
2. Add concrete half-open Unix bounds plus optional effective day/zone facts to
   the domain message filter and selection result. Keep relative words out of
   the application/domain boundary.
3. Inject one CLI clock, normalize `today`/`yesterday` once, validate exact
   dates and RFC3339 bounds, and reject ambiguous or conflicting inputs before
   authentication.
4. Apply the period predicate with sender matching in application-owned source
   selection. Clear every period field before the existing port call.
5. Extend catalog help and the selection prelude with effective period facts;
   leave unfiltered bytes unchanged and keep message records/provider order
   unchanged.
6. Add domain, application, adapter-leak, CLI, presentation, hostile/negative,
   and readiness tests plus a same-source token measurement.
7. Update active Japanese documentation and repository contracts, then run the
   full completion gate.

## Alternatives considered

### Only `--since` and `--until`

Rejected as the sole interface because an agent must calculate Tokyo calendar
boundaries for common “today”, “yesterday”, and single-day questions.

### Only `--on`

Rejected as the sole interface because partial-day and multi-day questions
would still require external filtering.

### Yearless dates such as `7/17`

Rejected because the year and grammar are ambiguous.

### Host-local `today`

Rejected because identical invocations could silently select different data
on agents running in different environments.

### Arbitrary `--time-zone`

Deferred. It broadens validation, tzdata, daylight-transition, documentation,
and reproducibility policy without evidence from the motivating Japanese
workflow. Explicit RFC3339 bounds already support other offsets.

### Provider query or historical paging

Rejected because the pinned endpoint exposes neither date bounds nor a cursor,
offset, or count. Local selection must not masquerade as provider filtering or
history access.

## Risks and controls

- **Boundary ambiguity:** half-open interval, explicit offset, whole seconds,
  and exact date grammar are validated before auth/I/O.
- **Clock race:** resolve the injected clock exactly once and render effective
  concrete bounds.
- **Silent context escape:** allow out-of-period records only through explicit
  direct reply context and keep anchor sequences visible.
- **Provider-policy leak:** adapter contract rejects any nonzero message filter;
  runtime tests prove one unchanged provider call.
- **False completeness:** retain `source-limit=100`, `complete=false`, source
  count, and the effective local selection.
- **Output regression:** keep unfiltered golden bytes unchanged and add exact
  filtered goldens.
- **Quality loss:** readiness answer keys must remain correct before token
  reduction is accepted.

## Verification

- Focused Go tests for domain, application, CLI, catalog, capsule, and adapter.
- Fixed-clock tests immediately before/after Tokyo midnight.
- Exact lower/upper boundary, one-sided bounds, empty result, sender AND period,
  index/count, reply context outside the period, equal timestamps, and invalid
  combination tests.
- One provider call and zero leaked local filter fields.
- Same-source publishable token measurement with pinned tokenizer metadata.
- `task check`.

## Completion evidence

- Focused domain/application/CLI/capsule/infrastructure/presentationeval tests
  passed, including fixed-clock and one-provider-call assertions.
- `task check:fast` passed after implementation.
- The full `task check` passed on 2026-07-20, including race tests, repository
  hygiene, architecture/contract/localization lint, security/public guards,
  vulnerability scan, release workflow lint, and all Go packages.
- Final review found no credential, live Chatwork data, private identifier,
  new dependency, provider-query leak, hidden additional call, or unrelated
  working-tree edit.
