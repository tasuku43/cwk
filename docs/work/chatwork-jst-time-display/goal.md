# Work Goal: Optional JST message timestamps

- Status: In progress
- Retention: temporary
- Retention reason: None
- Governing contract: Japanese public CLI and catalog output contract
- Review/delete trigger: Delete after durable contracts and checks are updated
- Owner: Codex
- Target: Optional `--jst` display for messages list/show
- Related ADRs: None

## Outcome

Users can add a fixed UTC+09:00 RFC3339 timestamp beside the existing Unix send time in `messages list` and `messages show`, without changing the typed request, provider calls, or default output.

## Why now

The user requested an optional local-time display. The option is presentation-only and low priority; it should preserve canonical Unix seconds and remain independent of host time-zone data.

## Non-goals

- Change API requests, sorting, filtering, or domain timestamp semantics.
- Change output when `--jst` is omitted.
- Add local-time fields for quote candidates or reachability boundaries.

## Acceptance criteria

- [ ] `--jst` is available only on `messages list` and `messages show`, removed before typed request/provider execution, and performs no extra I/O.
- [ ] Each rendered message row adds `send-time-jst=YYYY-MM-DDTHH:MM:SS+09:00`; int64 values whose fixed-JST year is outside 0000..9999 emit `unavailable` after a pre-conversion bound check.
- [ ] List primary/context rows and show anchor/reply-child/around/relation-context rows share the same formatter; quote candidates and reachability metadata keep their existing Unix-only representation.
- [ ] Omitted-option bytes remain unchanged; `--max-bytes` redraws account for JST together with `--max-body-bytes`.
- [ ] Catalog schema, exact help within 2,048 bytes, README, product/architecture/harness/skill/readiness documentation and tests agree.
- [ ] Focused tests pass and the root agent runs the repository gate.

## Governing documents

- Thesis: [Project theses](../../00_theses.md)
- Product contract section: [message output](../../01_product_contract.md)
- Architecture or security invariant: CLI-only presentation state; no external operation
- Existing ADR: None

## Completion definition

The acceptance criteria have evidence, documentation is promoted, the readiness scenario is updated, the owning agent verifies the repository gate, and this temporary packet is removed.
