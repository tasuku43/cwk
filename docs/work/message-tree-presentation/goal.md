# Work Goal: Make bounded message conversations directly legible

- Status: Accepted
- Owner: Project owner and Codex
- Target: Current implementation cycle
- Related ADRs: None; this is an explicit pre-1.0 text-contract decision

## Outcome

`messages list` renders one bounded Chatwork message window as a deterministic
reply tree with a document-local actor dictionary. An agent can recover provider
sequence, distinguish To from reply, identify unresolved parents, and pass every
canonical message reference unchanged to the next command without external
processing.

## Why now

The current flat projection repeats room, sender, trust, and relation-state
syntax on every message. The project owner has selected a concrete tree grammar
that preserves the typed semantic answer while reducing that repetition.

## Non-goals

- Inferring relations from message bodies, names, order, proximity, quotations,
  or meaning
- Removing raw Chatwork notation or rewriting message bodies
- Shortening, encoding, or reconstructing canonical references
- Changing `messages show`, other command presentations, or machine-oriented
  JSON/error/help contracts
- Adding presentation variants, switches, generalized benchmarks, summaries,
  or topic classification
- Maximizing token reduction beyond removal of clear repetition

## Acceptance criteria

- [ ] One room record, one external-text trust declaration, and one entry per
  known sender appear before a deterministic message tree.
- [ ] Every message retains its canonical reference and original provider
  sequence number; resolved reply edges determine indentation and branching.
- [ ] To and reply are separate, may coexist, and only typed relations are used.
- [ ] Unresolved reply targets retain an available canonical reference without
  being attached to a guessed node; target absence never fabricates a reference.
- [ ] Actor aliases are deterministic display-local aids only. Unknown To targets
  remain exact canonical account references because no name may be invented.
- [ ] Existing terminal-safe quoting protects actor names and bodies, with one
  global trust declaration and no repeated `untrusted:` prefix.
- [ ] Synthetic semantic, golden, hostile-text, edge-shape, and exact-reference
  tests cover the owner-specified matrix without real Chatwork data.
- [ ] A same-tokenizer before/after token measurement is recorded for one
  representative synthetic fixture.
- [ ] Scoped agent-readiness evidence proves reply-chain understanding and exact
  next-command reference reuse without external processing.
- [ ] `task check` passes and the work is committed in reviewable slices.

## Governing documents

- Thesis: Axiom 3 (semantics precede presentation), Thesis 1 (bounded context),
  Thesis 3 (canonical references)
- Product contract section: Anchor semantic outcome and data-presentation contract
- Architecture or security invariant: typed boundary before presentation;
  external text remains untrusted data
- Existing ADR: None

## Completion definition

The work ends when every acceptance criterion has recorded evidence, affected
thesis/product/architecture/readiness/harness contracts agree, `task check`
passes, and no live data, credentials, temporary diagnostics, or unrelated
changes remain.
