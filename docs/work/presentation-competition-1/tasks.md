# Work Tasks: Select the next agent presentation by evidence

## Understand

- [x] Read governing theses, product, architecture, security, harness,
  authentication, external-API, and agent-readiness documents.
- [x] Observe current rooms, messages, task, and file behavior in isolated live
  test scope without retaining credentials or private transcripts.
- [x] Record verified presentation and contract facts in `context.md`.
- [x] Confirm that the current theses already require an evidence-based
  presentation decision.

## Decide

- [x] Define materially different candidate concepts and a control.
- [x] Review concept mocks with the project owner before renderer
  implementation.
- [x] Freeze fixture corpus and presentation-independent answer keys.
- [x] Freeze prompts, agent/model/tool versions, repetitions, token accounting,
  score calculation, and promotion thresholds.
- [x] Decide whether the competition may select a task-family hybrid or must
  select one universal grammar.
- [x] Record the owner-selected compatibility decision after the inconclusive
  evaluation. Evidence: [decision.md](decision.md),
  [evaluation-audit.md](evaluation-audit.md).

## Repair shared correctness

- [x] Add all-result catalog-field completeness tests.
- [x] Preserve parent room references for message send, task create, and file
  upload results.
- [x] Preserve explicit zero unread/mention counts for read-state results.
- [x] Preserve declared acknowledgement for destructive/acceptance results.
- [x] Render declared contact-request names.
- [x] Align quote fixtures with the real adapter's account-and-time relation.
- [x] Align the `members replace` administrator field name with the catalog.
- [x] Pass focused contract, capsule, and relationship tests.

## Compete

- [x] Create one worktree per candidate from the same reviewed base commit.
- [x] Freeze a base commit containing shared correctness repairs, fixtures,
  answer keys, evaluator, frozen protocol, and passing gates.
- [x] Implement C0 measurement without behavior changes.
- [x] Implement P, L, R, and J behind the presentation boundary only.
- [x] Evaluate semantic, identity, trust, bounds, determinism, and
  output-boundary eligibility without promoting an ineligible candidate.
  Evidence: the strict scorer selected no eligible challenger in
  [evaluation-audit.md](evaluation-audit.md).
- [x] Run pinned agent tasks and retain raw per-run results, losing candidates,
  and invalidated attempts. Evidence:
  [evidence/manifest.json](evidence/manifest.json).
- [x] Calculate and audit quality, token, byte, tool-step, latency, and
  maintenance results without discarding failed runs. Evidence:
  [evaluation-audit.md](evaluation-audit.md).
- [x] Record the frozen outcome as no eligible challenger, then keep the
  owner's separate product decision outside the benchmark gate. Evidence:
  [decision.md](decision.md).

## Integrate and verify

- [x] Implement the reviewed subtractive task projection from the P seed on
  the integration branch, with shared semantic kind hardening and redundant
  coverage prose removed. Evidence: `258087d`, `3751fec`, `a832f69`, and
  `4669d41`.
- [x] Update the schema version, compatibility decision, goldens, and
  subtractive projection contract tests. Evidence: `07e6961` and
  [decision.md](decision.md).
- [x] Run focused domain and presentation tests during integration. Evidence:
  focused suites accompanied the integration and contract-test commits above.
- [ ] Run `task check`. Evidence:
- [ ] Run `task security`. Evidence:
- [ ] Run `task public:check` for publishable changes. Evidence:
- [ ] Remove temporary credentials and diagnostics. Evidence:
- [ ] Confirm repository status and candidate-worktree cleanup. Evidence:

## Hand off

- [ ] Acceptance criteria have evidence.
- [x] Durable decisions are promoted to governing documentation. Evidence:
  `5be3106` plus the final consistency review in this work packet.
- [x] Raw competition evidence identifies exact commits and tool versions.
  Evidence: [evidence/manifest.json](evidence/manifest.json).
- [x] Follow-up work is finite and does not reopen unrelated API coverage or
  authentication goals. Evidence: [goal.md](goal.md) limits remaining work to
  required gates and cleanup.
