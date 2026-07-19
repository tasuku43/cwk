# Work Goal: Bounded message selection

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: None

> Historical scope note: this completed packet intentionally preserved the
> then-current differential default. The later
> `message-window-default-recent` packet supersedes only that non-goal; the
> limit design and evidence below remain unchanged.

## Outcome

An agent or human can ask `messages list` for at most the newest N primary
messages in the selected provider window with `--limit <count>`, optionally
after exact sender selection and before explicit one-hop reply context is
added. The command remains one bounded provider call, preserves provider order
and canonical references, and requires no external filtering.

## Why now

The user needs only about the latest ten messages in common workflows. Today
`messages list` always projects the provider's result of up to 100 messages;
the displayed `limit=100` is a provider bound, not a user-selectable input.

## Non-goals

- Fetching messages older than Chatwork's one maximum-100-message response.
- Adding pagination, cursors, offsets, hidden additional calls, or an
  undocumented provider query parameter.
- Changing the default `changes` versus explicit `recent` window behavior.
- Limiting every list command or generalizing a query language.
- Sorting the rendered output or changing the flat adjacency schema.
- Treating reply context records as primary matches counted by `--limit`.
- Changing authentication, effects, mutation policy, or API coverage.

## Acceptance criteria

- [x] `messages list` declares optional, non-repeatable `--limit <count>` with
  the exact inclusive range 1 through 100 in catalog-derived human and agent
  help.
- [x] Omitting `--limit` preserves the existing selection behavior.
- [x] Without `--sender`, limit chooses the N greatest typed `send_time`
  messages; with one or more senders, it chooses the N greatest matching
  messages across their OR set. Equal times use provider position as a stable
  tie-break, while final output remains in original provider order.
- [x] `--context replies` runs after limit and may add direct typed parents or
  children beyond N; `--context none` emits at most N primary messages.
- [x] Selection metadata distinguishes the provider source bound, pre-limit
  candidate count, requested primary-message limit, anchors, and added context.
- [x] Original provider sequences and canonical message references survive;
  omitted parents are unresolved rather than inferred or fetched.
- [x] Invalid, duplicate, zero, negative, non-integer, and above-100 values
  fail before authentication or provider I/O.
- [x] The provider port receives no local limit/filter policy and exactly one
  Chatwork request; the request query remains limited to documented `force`.
- [x] A provider result exceeding its declared 100-message coverage fails
  before local selection can hide the violation.
- [x] Documentation, semantic tests, presentation tests, help, and the active
  agent-readiness scenario agree.
- [x] `task check` passes and the change is committed in coherent
  implementation, readiness, and documentation increments.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1, 3, and 4
- Product contract: `docs/01_product_contract.md`, Filtering and task
  composition
- Architecture: `docs/02_architecture.md`, Semantic and presentation boundary
- Security invariant: bounded read; invalid input and undeclared local policy
  fail before external I/O
- Existing ADR: None; this refines the existing typed message-selection use
  case without changing a trust or dependency boundary

## Completion definition

The work is complete when all acceptance criteria have executable evidence,
the no-post-processing latest-N scenario succeeds in one provider call, the
required documents agree, `task check` passes, and the reviewed change is
committed.
