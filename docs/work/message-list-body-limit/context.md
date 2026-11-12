# Context: `messages list --max-body-bytes`

## Verified facts

- `messages list` performs typed filtering, source-window relation resolution, reply context selection, and optional exact relation closure before capsule presentation.
- `--max-bytes` is already removed in CLI before request construction and is measured after the capsule renderer, including its final newline.
- `capsule.RenderWithOptions` has an invocation-local options struct and can render byte-budget delivery plans without changing application/domain requests.
- `chatwork.Message.Body` is valid UTF-8 and has a provider limit of 65,535 bytes.
- External body text is escaped only at presentation, preserving printable meaning while preventing terminal/line-structure injection.

## Decision

- Introduce an optional presentation-only `--max-body-bytes` value in the inclusive range 0..65,535.
- Measure the raw UTF-8 byte string before escaping and keep the longest prefix ending at a UTF-8 rune boundary.
- Apply the cap to all body records actually emitted by `messages list`, including primary anchors, selected in-window reply context, and emitted supplemental relation context.
- Do not render omitted anchors or omitted relation closure merely to report truncation.
- Emit `body-truncated=true original-body-bytes=N` only on records whose original body exceeds the cap. The semantic result and every completeness/continuation field remain unchanged.
- When combined with `--max-bytes`, run body projection before escaped total-output measurement and anchor-prefix planning.
- Without `--max-body-bytes`, preserve existing output bytes exactly.

## Unknowns

- None for this implementation slice.
