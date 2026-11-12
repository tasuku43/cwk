# Quote Candidate Matching

Retention: temporary

## Outcome

Preserve Chatwork quote metadata as an unresolved quote relation while exposing
only bounded, source-window message candidates when the exact quoted author and
whole send timestamp match. A candidate never proves message identity or a
reply edge.

## Non-goals

- No exact-message read or reply-context expansion from quote metadata.
- No fuzzy timestamp, nearby-time, text, name, or rendered-output matching.
- No new provider request or unbounded source traversal.

## Acceptance criteria

- Quote matching runs once over the complete original bounded source before
  filtering or byte delivery and its source provenance survives cloning.
- A candidate requires same-room sender identity and exact whole timestamp
  equality. Results retain canonical message references and optional visible
  source sequences under `source-window` scope with identity unproven.
- Missing timestamps, unknown relations/source, zero matches, and multiple
  matches remain unresolved. A unique source-window candidate is still not a
  resolved quote.
- Exact-message show uses only a recent source already fetched for replies or
  around; exact-only show makes no source request and has no candidate.
- Output reports unresolved replies and quotes separately, alongside unknown
  relation-set count. Coverage and access remain explicit.
- Shared semantic fixtures, answer keys, agent readiness, golden compatibility,
  and boundary tests cover matching and non-inference.
