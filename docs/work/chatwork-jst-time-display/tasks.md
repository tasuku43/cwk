# Work Tasks: Optional JST message timestamps

## Understand

- [x] Read governing theses, product, architecture, security and harness documents.
- [x] Observe current exact help byte count and renderer path.
- [x] Record verified bounds and constraints in `context.md`.
- [x] Confirm presentation-only scope in `goal.md`.

## Decide

- [x] Select fixed UTC+09:00 formatting independent of host zone data.
- [x] Keep Unix `send_time` and default output unchanged.
- [x] Classify `--jst` as an optional CLI presentation flag with no effect or reference flow.

## Implement

- [x] Add fixed-offset formatter with pre-conversion year bounds and tests.
- [x] Add list/show renderer option and context coverage tests.
- [ ] Add catalog inputs, conditional field declarations, and exact-help compaction.
- [ ] Strip the flag before typed request construction and preserve it through byte-budget redraw.
- [ ] Add CLI request/provider guard, help, active-catalog, and combined budget tests.
- [ ] Update durable documentation, skill, and readiness scenario.

## Verify

- [x] Focused capsule tests pass. Evidence: `go test ./internal/cli/capsule -run 'Test(FormatSendTimeJST|RenderMessagesListJST|RenderMessagesShowJST)' -count=1`.
- [ ] CLI/catalog tests pass. Evidence:
- [ ] `task check` passes. Evidence:
- [ ] Relevant readiness scenario updated and replayed. Evidence:
- [ ] Temporary packet removed after promotion.
