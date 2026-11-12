# Work Context: Optional JST message timestamps

This file records verified facts and unresolved questions.

## Current behavior

- Message rows currently render `send-time` as Unix seconds in the headerless task projection (`internal/cli/capsule/capsule.go`).
- `messages list` already routes through `capsule.RenderWithOptions` for body and output byte caps; list and show typed results include Unix-second `SendTime` values.
- `messages show` may render an anchor, direct reply children, around-context records, and relation-context records.
- Exact `messages list --help` is currently 2,048 bytes, leaving no room for an additive option until concise catalog descriptions reclaim bytes.
- `--max-body-bytes` is removed into CLI presentation state before the typed request and provider call.

## Relevant structure

- Entry point: `internal/cli/runChatwork`
- Domain rule: `chatwork.Message.SendTime` remains signed Unix seconds.
- Application use case: unchanged
- Infrastructure boundary: unchanged
- CLI catalog or presentation: `internal/cli/chatwork_catalog.go`, `internal/cli/chatwork.go`, `internal/cli/chatwork_output_budget.go`, `internal/cli/capsule`
- Existing tests and harness checks: capsule goldens, CLI request guards, help/catalog tests, README schema test, readiness scenarios

## Constraints

- Preserve existing defaults byte for byte.
- The user-facing CLI and documentation use Japanese.
- The formatter uses `time.FixedZone` with a constant +09:00 offset and does not consult `time.Local` or named-zone data.
- Check signed Unix-second bounds before calling `time.Unix` for JST local years 0000 through 9999.
- Keep the original Unix field unchanged and display a separate conditional string field.

## External facts

- RFC Editor, [RFC 3339](https://www.rfc-editor.org/rfc/rfc3339), checked 2026-11-08: section 1 bounds the current-era dates to years 0000..9999; section 5.6 defines a four-digit year and numeric `±HH:MM` offset grammar.

## Unknowns

- [x] Whether the Go formatter emits year zero in four digits: covered by a boundary test.
- [x] Whether ambient local time zone changes formatting: covered by a test that replaces `time.Local`.

## Thesis evidence

- Repeated design decision or point of agent confusion: None yet.
- User outcome or friction observed in the minimal slice: Optional readable time without changing sortable/reusable Unix seconds.
- Code workaround or exception being considered: None.
- Current thesis that resolves it, or proposed thesis revision: Existing semantics-before-presentation and controlled-boundary theses suffice; no thesis change proposed.
- Downstream product, architecture, security, Skill, catalog, and harness impact: Conditional CLI output field and optional flag; no domain/provider changes.

## Reproduction or observation

```sh
go run ./cmd/cwk messages list --help | wc -c
```

Observed before this change: `2048` bytes. The flag description and usage line must fit the exact-help cap.

## Security and public-boundary notes

- Assets and side effects involved: None; formatting only.
- Credentials or confidential data involved: None.
- New dependencies, destinations, files, processes, or generated content: No dependencies or destinations; temporary work packet only.
- External schema provenance, publication rights, and drift evidence: No provider schema change.
- Pagination, timeout, retry, idempotency, and cancellation facts: Unchanged.
- Publication and licensing concerns: None.

## Glossary

- JST display: A fixed numeric UTC+09:00 offset for presentation, not a host or political time-zone database lookup.
