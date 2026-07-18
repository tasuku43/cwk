# Work Goal: Select bounded messages by exact sender

- Status: Completed
- Owner: Project owner and Codex
- Target: Current implementation cycle
- Related ADRs: None; this extends the accepted typed message-selection rule

## Outcome

An agent can ask `messages list` for messages authored by one or more exact
accounts, optionally retaining direct in-window typed reply neighbors, without
`jq`, text search, name matching, raw-notation parsing, or another provider
call. The result distinguishes sender matches from added context and preserves
the original provider sequence and canonical references.

## Why now

The complete message presentation removes repeated output work, but an agent
still has to consume every message in the provider window when the task concerns
one sender or a small speaker set. The owner explicitly requested sender and
person-pair filtering and optional related-edge context.

## Non-goals

- Filtering by display name, body text, time range, regex, or arbitrary query
  expressions
- Inferring a relationship from raw `[To]`/`[rp]` text, names, proximity, or
  message meaning
- Claiming that two selected senders prove an exclusive pairwise conversation
- Fetching reply parents outside the provider-returned window
- Adding transitive thread closure, To/quote target expansion, pagination, or
  another provider request
- Changing message bodies, actor aliases, canonical-reference syntax, mutation
  behavior, authentication, or unrelated command output

## Acceptance criteria

- [x] `--sender <account-ref>` is repeatable and matches any listed exact sender.
- [x] Two repeated sender flags provide a truthful two-sender-focused slice without
  claiming that untyped posts are directed between them.
- [x] `--context none|replies` defaults to `none`, requires a sender filter, and
  `replies` adds only direct in-window endpoints of typed reply edges touching a
  sender match.
- [x] Filtering occurs after the one bounded provider response and before
  presentation; it performs no additional I/O.
- [x] Provider source sequence remains stable and may contain gaps after
  selection; displayed canonical references remain directly reusable.
- [x] Output records source count, exact sender filter, context mode, and the
  source sequences that matched the sender condition once per document.
- [x] A reply whose parent is omitted becomes explicitly unresolved with its
  available canonical target; no false resolved edge remains.
- [x] Empty matches, duplicate/malformed references, invalid context use,
  interleaved branches, hostile text, and raw-notation canaries are covered by
  synthetic tests.
- [x] Existing unfiltered `messages list`, `messages show`, machine-format,
  authentication, and provider-request contracts remain unchanged.
- [x] Scoped agent help makes sender OR semantics, direct reply context, bounds,
  and canonical inputs discoverable in one invocation.
- [x] `task check` passes on the final committed state.

## Governing documents

- Thesis: Axiom 4, eligible agent-native presentation
- Product contract: Agent-output axioms and filtering/task composition
- Architecture: Semantic and presentation boundary
- Security: External text and bounded Chatwork read boundary
- Existing ADR: None required; no dependency, destination, or trust change

## Completion definition

This work ends when the two public flags, typed selection metadata, pure
application selection, presentation, synthetic semantic/readiness fixtures,
catalog/help contracts, durable documentation, and final `task check` agree.
Further filter kinds and deeper context traversal remain follow-up work.
