# Work Context: Message index selection

## Verified current behavior

- The preceding contract offered `messages list --limit N`, which could select
  only newest ranks 1 through N and could not name a later slice.
- Sender OR precedes local selection; direct one-hop reply context follows it.
- Chatwork's message-list endpoint exposes `force`, not count, offset, or cursor.
  The adapter fetches one maximum-100 response; local selection changes output
  size, not response bytes.
- Presentation preserves provider order even though typed `send_time` and later
  provider position establish newest-first rank.

## External facts reviewed 2026-07-20

- RFC 7644 section 3.4.2.4 defines SCIM index pagination: `startIndex` is the
  one-based index of the first result and `count` is the requested maximum
  number of results. Responses expose `totalResults`, `startIndex`, and
  `itemsPerPage`. Its example continues with `startIndex=11&count=10`.
  Source: https://datatracker.ietf.org/doc/html/rfc7644#section-3.4.2.4
- RFC 7644 also warns that index pagination is stateless and may be inconsistent
  if the result set changes between requests.
- RFC 9865 defines a later SCIM cursor-pagination alternative. Chatwork exposes
  no corresponding cursor on this endpoint, so this work does not emulate one.
  Source: https://www.rfc-editor.org/rfc/rfc9865.html

## Constraints and decision

- Use SCIM-derived CLI vocabulary and arithmetic, not a claim of SCIM protocol
  conformance: `--start-index 11 --count 20` means ranks 11 through 30.
- Rank is newest first by typed send time, with later provider position breaking
  ties; physical output stays in provider order.
- Keep candidate count, start index, requested count, actual items per page, and
  optional next start index as application-owned provenance.
- Each command re-fetches its `recent` or `changes` source; ranks may move after
  source mutation. No snapshot-stability claim is allowed.
- No new dependency, credential, destination, storage, mutation, or schema
  fixture is introduced.

## Relevant structure

- CLI assembly: `internal/cli/chatwork.go`
- Domain: `internal/domain/chatwork/chatwork.go`
- Application: `internal/app/chatworkcmd/message_selection.go`
- Infrastructure guard: `internal/infra/chatworkapi/request.go`
- Catalog/presentation: `internal/cli/chatwork_catalog.go` and
  `internal/cli/capsule/capsule.go`
- Readiness: `tools/presentationeval/active_message_index*`

## Security and public boundary

- Effect remains one exact-room bounded read.
- Credentials remain the process-local `CWK_API_TOKEN` binding.
- Invalid input fails before authentication/I/O; oversized provider results fail
  before local selection can hide them.
- No private data, live credential, external fixture, or license change.
