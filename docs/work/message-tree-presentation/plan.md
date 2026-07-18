# Plan: Flat message adjacency presentation

## Chosen approach

1. Extend the message-list semantic result with its exact requested room scope,
   bind it in infrastructure, and validate it against the request.
2. Add a presentation-independent synthetic conversation fixture and assertions
   for order, typed relations, missing targets, and canonical identity.
3. Replace only the `messages list` text renderer with a deterministic actor
   dictionary plus chronological adjacency list. Keep `messages show` and
   non-Chatwork JSON paths
   unchanged.
4. Update the catalog, golden/contract/hostile tests, and agent-readiness simulator
   to state the new public grammar and reference flow.
5. Measure the old repeated projection and new flat projection for the same synthetic
   fixture with one pinned tokenizer implementation and record raw counts.
6. Propagate the explicit compatibility decision through theses, product,
   architecture, harness/readiness documentation, then run repository gates.

## Output decisions

- Header: room scope, count, requested window, bound, completeness, and aggregate
  unresolved count appear once.
- Trust: `external-text=untrusted escaped` appears once.
- Schema: one line declares the positional node and optional edge fields.
- Actors: `aN account-ref=... name="..."` appears once per distinct sender,
  ordered by first sender occurrence.
- Nodes: one physical line per provider item, in unchanged provider order, starts
  with `#N` and carries the full canonical `message-ref` plus actor alias.
- Resolved in-window reply: `reply=#N`; indentation carries no semantics and is
  not emitted.
- Unresolved reply with target: `reply=?<message-ref>`; without target: `reply=?`.
- To: known actors use aliases; unknown actors retain canonical account refs.
- Quote: remains a separate typed field; unresolved state uses `?`.
- Body: one terminal-safe quoted field on the same line, preserving raw notation.
- Omitted derived fields: depth, thread, root, children, `relations=none`, and
  `state=resolved`.
- Forbidden compression: `prev`, relative offsets, whitespace-only edges, and
  any reference encoding that needs reconstruction.

## Alternatives rejected

- Reparse raw notation in presentation: violates the typed semantic boundary.
- Resolve missing parents by proximity/order: invents relations.
- Assign aliases to nameless recipients: violates the actor name requirement or
  fabricates external text.
- Use local message aliases as command identity: breaks canonical round-trip.
- Add a presentation switch: explicitly outside scope.

## Risks and controls

- Empty windows losing room scope: add and request-bind a semantic scope field.
- Inconsistent resolved relations: use only the application-resolved typed edge;
  the flat renderer performs no traversal and therefore cannot recurse or duplicate.
- Actor-name inconsistency: reject instead of choosing one silently.
- Hostile names/body breaking line structure: reuse the existing quoted projection
  and add structural-rune tests.
- Deep chains becoming quadratic: use constant-width records and test a synthetic
  50-message chain for no indentation and approximately linear output growth.
- Accidental changes to other outputs: exact snapshots for `messages show`, help,
  and representative JSON outputs.

## Verification

- Focused domain, application, infrastructure, capsule, CLI, and presentationeval tests
- Same-tokenizer before/after measurement on publishable synthetic data
- Agent-readiness semantic-answer and exact-reference scenario
- `task check`
