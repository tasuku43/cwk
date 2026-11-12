# Context

## Verified facts

- Infrastructure parses `[qt][qtmeta aid=<account> time=<timestamp>]...[/qt]`
  into quote metadata, currently represented by an account target and optional
  timestamp string. The provider notation contains no message ID.
- Message relations are resolved in `internal/app/chatworkcmd/relationships.go`
  only against a bounded message set. Quote relations are deliberately never
  marked resolved there.
- `messages list` receives one bounded provider source before local filters.
- `messages show` reads a bounded recent source only for explicit reply context
  or `--around`; exact-only show currently uses only its exact read.
- The text projection currently combines unresolved replies and quotes into
  `unresolved-relations`; unknown relation sets have a separate count.
- Existing coverage and message-access values already identify the original
  bounded source and any provider restriction.
- Supplemental reply-context messages can carry quote candidates computed from
  the original list source even when those messages are outside the selected
  output. Their candidates need the same source-sequence pruning as primary
  records; relation-context records themselves do not expose a source position.

## Constraints

- Canonical references are exact positive decimal values. Display aliases are
  not action identity.
- Quote author/time may produce candidate evidence only; they cannot prove an
  identity, fetch target, reply edge, or authorization target.
- Candidate source order must follow the provider window. A source sequence is
  emitted only when that source message is itself present in the delivered
  result; canonical candidate references remain available otherwise.
- Unknown relation sets carry no partial relation facts.

## Resolved design choice

- Final states and text spellings are frozen by the domain types, active
  semantic answer key, catalog field, and capsule goldens.

## Implementation evidence

- Quote candidate states and the `quote-candidate` projection use
  `unique-window-candidate`, `ambiguous-window-candidates`,
  `no-window-candidate`, `missing-timestamp`, and `unknown`. Every candidate
  retains `scope=source-window` and `identity=unproven`.
- `go test ./internal/domain/chatwork ./internal/app/chatworkcmd ./internal/cli/capsule -count=1` passed after the supplemental reply-context sequence-pruning case was added.
- `go test ./internal/cli -run TestChatworkCorrectnessCriticalCatalogFields -count=1` passed with `quote_candidates`, `unresolved_replies`, and `unresolved_quotes` declared.
- `go test ./tools/presentationeval -count=1` passed with the active quote answer key/readiness path and compatibility goldens. The historical Competition 1 fixture remains unchanged.
- Terra review caught that exact-message show anchors were not always traversed as quote-candidate owners. The result pruning pass now separates owners from source-sequenced visible records: replies-only anchors and outside/unknown around anchors always lose hidden candidate sequences, while an in-window child target keeps its source sequence. Dedicated service regressions cover hidden-source, visible-child, and outside-anchor cases.
- After that correction and parallel resources integration, `go test ./internal/domain/chatwork ./internal/app/chatworkcmd ./internal/cli/capsule ./internal/cli -count=1` and `go test ./tools/presentationeval -count=1` both passed.
- The full `task check` is pending parent integration with concurrent `messages resources` work.
