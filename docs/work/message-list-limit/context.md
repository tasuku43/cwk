# Work Context: Bounded message selection

## Verified current behavior

- `messages list` declares `--room`, `--window`, repeatable `--sender`, and
  `--context`; it has no public count limit.
- Its text header's `limit=100` is `Coverage.Limit`, the documented provider
  response ceiling, not a user input.
- The existing public `--limit` name belongs to `room-tasks create` and means a
  Unix deadline. Its `Request.Limit` field must not be reused for message
  cardinality.
- Chatwork's official message-list endpoint documents only `force`: zero returns
  the differential result and one returns up to the latest 100 messages. It
  documents no limit, cursor, or offset parameter:
  `https://developer.chatwork.com/reference/get-rooms-room_id-messages`.
- The official contract calls the forced result the latest messages but does
  not guarantee response-array direction. Typed `send_time`, with provider
  position as a deterministic tie-break, must therefore select newest records;
  presentation continues to preserve provider order.
- The adapter makes one request, preserves response-array order, and declares
  message coverage as incomplete with a 100-message source limit.
- Application currently removes sender/context policy before the port, resolves
  typed replies over the provider window, applies sender OR selection and
  optional direct reply context, then validates the selected result.
- Selection provenance already records provider source count, original
  one-based sequences, and direct-match anchor sequences.
- Current validation bounds filtered `SourceCount` by coverage but does not
  reject an unfiltered 101-message result before local selection.

## Constraints

- `cli.Catalog` remains the only public input/help declaration.
- Limit is application-owned typed selection, never a provider query.
- Default invocation bytes and behavior remain unchanged except that the
  message header renames the provider bound to `source-limit` to distinguish it
  from the new selection input.
- Sender matching precedes limit. Repeated senders remain one OR candidate set,
  not N results per sender.
- Reply context follows limit. Context may increase displayed count beyond N,
  but cannot leave the provider source window or traverse transitively.
- Provider sequence and canonical references are never reconstructed from
  timestamps. Timestamps select candidates only; they do not reorder output.
- `--limit` does not imply `--window recent`; current differential default is
  explicit in help.
- No live account, token, or real Chatwork data is required for tests.

## External-call and security facts

- Effect remains `read` against the exact room reference.
- Authentication remains process-local PAT through the existing secret-free
  binding.
- Destination, timeout, attempts, response limits, and output ceiling do not
  change.
- Invalid limit input must fail in CLI/domain validation before adapter
  construction or authentication.
- A local limit can reduce output/token cost when it removes primary records;
  it never reduces provider response bytes.

## Resolved questions

- Public syntax: `--limit <count>`.
- Range: 1 through 100 inclusive; zero means absent only inside typed state.
- Default: absent, preserving the existing maximum-100 source selection.
- Composition: sender predicate, latest-N primary anchors by `send_time`, then
  optional direct reply context.
- Tie-break: later provider position wins when `send_time` is equal.
- Output order: original provider order and original source sequences.
- Limit-only context: allowed; it adds direct typed neighbors to the selected
  newest anchors. Context without either sender or limit remains invalid.
- Provider bound: remains 100 and is rendered as `source-limit`; requested
  limit belongs to the selection record.
- Pagination: none; one complete bounded task result is returned.

## Thesis evidence

The repeated desire to avoid receiving all 100 messages is evidence for Axiom
1's operational closure and Axiom 3's finite typed selection: asking every
agent to pipe or manually trim message output would reintroduce routine external
processing and unnecessary tokens.

## Runtime observation commands

```sh
go run ./cmd/cwk messages list --help
go run ./cmd/cwk help messages list --format agent
go test ./internal/domain/chatwork ./internal/app/chatworkcmd ./internal/cli ./internal/infra/chatworkapi
go test ./tools/presentationeval -run 'TestActiveMessage'
```

## Completion evidence

- Human and agent scoped help expose the optional non-repeatable 1..100 limit,
  its sender-before-limit-before-context composition, and the need for
  `--window recent` when the outcome is the room's current latest window.
- Synthetic domain/application fixtures cover non-chronological provider
  order, equal timestamps, sender OR, no-sender selection, direct context beyond
  N, omitted unresolved parents, empty/1/100 bounds, and oversized sources.
- CLI and adapter tests prove invalid and duplicate input fails before
  authentication, the room-task Unix deadline remains independent, local
  selection never crosses the provider port, and HTTP emits only documented
  `force` state.
- The active readiness scenario completes newest-N selection, relation
  traversal, and canonical-reference reuse in one provider task call with no
  external processing.
- Independent domain/application, CLI/presentation, and documentation reviews
  reported no blocking issue.
- The full `task check` gate passed on 2026-07-19 with Go 1.26.5, including
  standard and race tests, security and vulnerability checks, release lint, and
  public-boundary checks.
