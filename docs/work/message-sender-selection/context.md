# Work Context: Message sender selection

## Current behavior

- `messages list` accepts only exact room and `recent|changes` window inputs.
- Chatwork returns at most 100 messages for the reviewed endpoint and exposes no
  sender filter in the pinned request contract; selection must operate on the
  complete bounded typed response.
- The renderer preserves provider order as contiguous `#sequence` values and
  rejects a resolved reply whose target is absent from the displayed result.
- Typed sender, To, reply, and quote facts are available before presentation.
  Raw body notation is retained but is not a semantic source in presentation.
- Reply resolution currently runs at the CLI boundary after application
  execution; selection belongs with application outcome assembly instead.

## Relevant structure

- Entry point: `internal/cli/chatwork.go`
- Domain rule: `internal/domain/chatwork/chatwork.go`
- Application use case: `internal/app/chatworkcmd/service.go`
- Typed reply resolution: `internal/app/chatworkcmd/relationships.go`
- Infrastructure boundary: `internal/infra/chatworkapi/request.go` and
  `response.go`
- Catalog/presentation: `internal/cli/chatwork_catalog.go` and
  `internal/cli/capsule/capsule.go`
- Active semantic fixture: `tools/presentationeval/active_message_adjacency.go`

## Constraints

- Filters use canonical `chatwork-account` references, never names.
- Repeated senders use OR semantics; mixing implicit AND/OR expression grammar
  is out of scope.
- `context=replies` is one hop over typed, resolved, in-window reply edges. It
  does not inspect To, quote, body, or external IDs to invent an endpoint.
- Added context may have a different sender, so output must identify the exact
  source sequences that satisfied the sender predicate.
- Selection must retain source sequence gaps and source-window count so
  `count=0` is not mistaken for an empty provider response.
- No new dependency, credential, network destination, provider call, or wire
  schema is needed.

## External facts

- The repository's pinned Chatwork API contract records a 100-item bound for
  `GET /rooms/{room_id}/messages`; this work does not update that snapshot.
- The existing adapter maps only `force=1` for recent-window retrieval. Sender
  and context selection remain provider-independent application behavior.

## Unknowns resolved before implementation

- [x] “Who and who” means a speaker slice produced by repeated sender inputs;
  it does not assert exclusive pairwise addressing without typed evidence.
- [x] Related context means direct typed reply neighbors, not transitive thread,
  To-recipient expansion, quote expansion, or raw-tag parsing.
- [x] Default context is `none` for token efficiency; `replies` is explicit.
- [x] Context records are distinguished by a document-level list of sender-match
  source sequences rather than repeated per-record labels.

## Measurement evidence

The frozen 14-message readiness fixture and pinned `tiktoken==0.13.0`
`o200k_base` measurement are recorded in [token-measurement.md](token-measurement.md).
Exact sender anchors reduced the fixture from 443 to 208 tokens; direct reply
context reduced it to 341 tokens while preserving the tested semantic answer.

## Implementation evidence

- `MessageFilter` and `MessageSelection` bind exact sender refs, context,
  source count, original sequences, and anchor sequences in the typed result.
- Domain validation proves anchors match requested senders and every added
  context node is a direct resolved reply parent or child of an anchor; known
  empty provenance uses explicit non-nil empty collections.
- Application assembly clears local filter fields before one port call,
  resolves the full source window, performs stable OR selection and optional
  one-hop expansion, then re-resolves the displayed subset so omitted parents
  remain canonically unresolved.
- The Chatwork adapter rejects leaked selection fields. `messages show` retains
  the strict relation-consistency check formerly applied at the CLI boundary.
- Catalog input repeatability is machine-readable and is the parser's source of
  truth; scoped help declares the sender OR rule, 100-reference bound, and
  direct reply context.
- Exact unfiltered and filtered goldens, hostile/raw-notation canaries,
  canonical-reference round trips, and active semantic/readiness fixtures pass.

## Thesis evidence

- Repeated external filtering would violate the no-post-processing outcome.
- Filtering after rendering would parse the presentation and weaken the shared
  semantic boundary.
- Moving reply resolution and selection into application outcome assembly
  aligns the existing implementation with the architecture statement that
  filtering/context selection are application concerns.

## Reproduction or observation

```sh
go run ./cmd/cwk help messages list --format agent
```

Observed before this change: only `--room` and `--window` are declared.

## Security and public-boundary notes

- Effect remains read-only; target remains the exact room.
- Account filters are public canonical references, not credentials.
- The existing PAT-only authentication and fixed Chatwork destination remain
  unchanged.
- All fixtures use synthetic numeric references and hostile synthetic text.

## Glossary

- **sender match**: a message whose canonical sender is any supplied sender.
- **reply context**: a direct in-window reply parent or child connected by an
  existing typed reply edge to a sender match.
- **source sequence**: the one-based index in the provider-returned window
  before local selection.
