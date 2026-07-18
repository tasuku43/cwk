# Context: Positional message records

## Verified facts

- Repository profile is `ready` and the working tree is clean.
- The accepted flat adjacency format already emits one schema line and keeps
  every message in provider order on one physical line.
- `message-ref`, `sent`, and quoted `body` currently repeat labels on every
  message despite having fixed positions declared by the schema.
- Canonical identity is the message reference value, not the `message-ref=`
  spelling. A positional round-trip test can validate and pass the second field
  unchanged.
- Optional typed edges vary by message, so `reply=`, `to=`, and `quote=` remain
  labeled.
- The body is terminal-safe quoted and last, so spaces, newlines, and field-like
  external text cannot change record structure.
- Chatwork success remains text-only; existing JSON help/error contracts are not
  affected.

## Fixed grammar

```text
schema: #sequence message-ref actor sent [reply] [to] [quote] "body"
#10 2130329932361236480 a1 1784383835 reply=#8 to=a2 "message body"
```

The schema, not repeated per-record labels, assigns meaning. `#N` and actor
aliases remain document-local; the raw second field remains the canonical value
accepted by commands.

## Evidence

- `62fb36f` implements the positional renderer and its catalog, golden, hostile,
  deep-chain, canonical-round-trip, and active-readiness tests.
- The synthetic CLI produces `#3 9003 a2 1700000020 reply=#1 to=a1 "done"`
  without changing its typed semantic answer.
- `tiktoken==0.13.0` with `o200k_base` counts 330 tokens for the positional
  output, versus 365 for the labeled adjacency and 541 for the repeated
  baseline. Exact hashes and byte counts are in `token-measurement.md`.
