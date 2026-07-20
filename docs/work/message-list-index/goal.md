# Work Goal: Message index selection

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: None

## Outcome

An agent or human can request an unambiguous ordinal message slice. For an
unchanged bounded source and filter, `messages list --count 10` selects ranks 1
through 10 and `messages list --start-index 11 --count 20` selects ranks 11
through 30, while preserving provider-order output and exact canonical
references.

## Why now

The user wants progressive retrieval without the ambiguity of whether a generic
limit or second number means a count or an inclusive end rank. SCIM's one-based
`startIndex` plus maximum `count` vocabulary provides a familiar model.

## Non-goals

- SCIM protocol conformance, provider pagination, a cursor, or retrieval older
  than Chatwork's maximum-100-message source window.
- Persisting a snapshot or claiming stable ranks when messages change between
  invocations.
- Reducing provider response bytes or adding a second request per command.
- Changing room-task `--limit`, recent-window defaults, sender OR semantics,
  reply-context bounds, authentication, effects, or mutation policy.

## Acceptance criteria

- [x] `messages list` declares optional non-repeatable `--start-index <index>`
  and `--count <count>`, each in 1 through 100; count alone defaults start index
  to 1.
- [x] Sender OR runs first, typed newest ordering determines rank, start/count
  select primary anchors, and direct reply context runs last.
- [x] `--count 10` followed by `--start-index 11 --count 20` selects ranks 1--10
  and 11--30 respectively for an unchanged source and filter.
- [x] Output distinguishes candidate count, applied start index, requested
  count, actual items per page, optional next start index, and provider
  `source-limit`.
- [x] Provider order, source sequences, canonical references, anchors, and
  context distinction remain explicit.
- [x] Invalid or duplicate values fail before authentication/provider I/O, and
  application-only selection state cannot cross the provider port.
- [x] Scoped help and a synthetic readiness scenario complete a first and next
  slice with two bounded task calls and zero external parsing.
- [x] `task check` passes.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1, 3, 4, and 8
- Product contract: `docs/01_product_contract.md`, Filtering and task composition
- Architecture: `docs/02_architecture.md`, Semantic and presentation boundary
- Security invariant: bounded read; invalid local selection fails before I/O
- Existing ADR: None; this revises a pre-1.0 message-selection input

## Completion definition

Every criterion has executable evidence, durable contracts and the capability
skill agree, readiness requires no external processing, and `task check` passes.
