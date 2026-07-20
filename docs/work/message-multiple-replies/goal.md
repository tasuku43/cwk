# Work Goal: Preserve multiple explicit message replies

- Status: Accepted
- Owner: Product owner
- Target: Current development cycle
- Related ADRs: None

## Outcome

An agent reading `messages list` can follow every valid explicit Chatwork reply target from a message, including two or more `[rp]` tags, without reparsing the untrusted body. Resolved and unresolved targets remain typed and `relation-state=unknown` is reserved for genuinely unparseable or contradictory notation.

## Why now

Black-box production evidence showed that a second valid `[rp]` tag currently discards the whole typed relation set. This breaks reply-chain reconstruction even when every target is inside the returned 100-message window and makes `unknown-relation-sets` overstate uncertainty.

## Non-goals

- Infer replies from To, quotes, display names, prose, or time proximity.
- Follow cross-room references or add provider pagination.
- Change single-reply rendering.
- Change the relation-fetch limit or authentication boundary.

## Acceptance criteria

- [x] Two or more valid `[rp]` tags produce an ordered typed reply collection and retain all To recipients.
- [x] Multiple in-window targets render as `reply=[#a,#b]` without `relation-state=unknown`.
- [x] One reply continues to render as `reply=#a`.
- [x] Mixed resolved and unresolved targets retain each target using the existing `?` convention, and relation closure considers every same-room target.
- [x] `--context replies` includes every direct typed parent/child neighbor for the selected anchors.
- [x] Unknown and unresolved counts reflect the typed facts rather than the former single-slot limitation.
- [x] Exact-command agent help and durable schema documentation describe one-or-many reply values and the meaning of `relation-state`.
- [x] `task check` passes.

## Governing documents

- Thesis: Axioms 1, 3, 4, and 8
- Product contract section: Agent-output axioms and bounded relation closure
- Architecture or security invariant: Infrastructure parses notation; domain owns relation truth; application resolves; CLI only renders
- Existing ADR: None

## Completion definition

The change is complete when semantic fixtures, parser/domain/application/presentation tests, agent contract, durable documentation, and full repository gates agree.
