# Work Goal: Remove redundant labels from fixed message records

- Status: Accepted
- Owner: Project owner and Codex
- Target: Current implementation cycle
- Related ADRs: None; this refines the accepted flat adjacency contract

## Outcome

`messages list` uses one fixed schema to identify the positional sequence,
canonical message reference, actor alias, Unix send time, and quoted body. Each
message stops repeating `message-ref=`, `sent=`, and `body=` while preserving
the exact values, optional typed edges, one physical line, and direct canonical
reference reuse.

## Non-goals

- Removing message references, send times, bodies, actors, or typed relations
- Changing `messages show`, semantic models, relation resolution, or escaping
- Making optional reply, To, or quote fields positional
- Adding another presentation or a format switch

## Acceptance criteria

- [ ] The schema is exactly `#sequence message-ref actor sent [reply] [to]
  [quote] "body"` and every message record conforms to it.
- [ ] Message records contain none of `message-ref=`, `sent=`, or `body=`.
- [ ] Every canonical message reference is accepted unchanged by a downstream
  `--message` input.
- [ ] Provider order, reply/To/quote semantics, unresolved targets, hostile-text
  framing, and one-record-per-physical-line behavior remain unchanged.
- [ ] Active semantic/readiness fixtures and goldens use the positional grammar.
- [ ] The same pinned tokenizer measures the updated after-golden against the
  unchanged pre-adjacency baseline.
- [ ] `task check` passes and the change is committed.

## Completion definition

The work ends when the renderer, schema, tests, active evidence, token record,
and public documentation agree on one positional grammar and the full repository
gate passes.
