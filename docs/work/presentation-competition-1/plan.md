# Work Plan: Select the next agent presentation by evidence

- Status: Proposed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Separate correctness repair from presentation optimization. First make every
catalog-declared task result observable with synthetic contract tests. Then
freeze a shared semantic fixture corpus and evaluation protocol. Implement the
baseline and challengers in isolated worktrees, reject any candidate that
weakens semantics or safety, and compare only eligible candidates on agent
quality and resource cost. Promote one reviewed winner or retain C0.

## Candidate concepts

### C0: Current context capsule

The compatibility baseline. It uses a document-level reference dictionary,
generic typed result blocks, explicit relationship states, and visibly framed
external text. It is expected to score well on explicitness and poorly on
task-irrelevant repetition.

### P: Task-shaped projection

Each catalog task owns a fixed projection containing only its declared output
facts. Known scope is hoisted once, absent optional facts are omitted, and
explicit zero is preserved where the task contract distinguishes it from
absence. This is the leading general-purpose hypothesis.

### L: Normalized ledger

Repeated identities and values are emitted once in typed ledgers and records
refer to them by local display keys. This may minimize large-list bytes but
requires additional mental joins and is therefore a distinct hypothesis.

### R: Relationship-first timeline

Message output prioritizes source order and typed To/reply/quote edges, with
actors and known room scope declared once. This may improve thread
understanding but is not assumed to fit non-message tasks.

### J: Typed semantic JSON control

A compact, stable JSON projection preserves the same facts without a custom
text grammar. It provides a machine-familiar control for measuring whether a
text presentation earns its maintenance cost.

## Design

### Public contract

This packet changes no command, role, reference kind, authentication method,
effect, impact, provider call, or failure contract. Candidate output is
experimental until review accepts it. The accepted result must preserve the
current public stream, exit, completeness, trust, and canonical-reference
contracts and receive an explicit schema/compatibility decision.

### Layer changes

- Domain: no candidate-specific type or grammar.
- Application: no candidate-specific selection, joining, or omission.
- Infrastructure: no candidate-specific parsing or wire changes.
- CLI: shared task-result completeness repair followed by isolated renderers
  that consume the same semantic boundary.

### Data and control flow

```text
synthetic provider fixture
  -> reviewed typed semantic result and exact answer key
  -> one isolated candidate renderer
  -> direct agent task, with no external reconstruction
  -> exact answer scoring plus token/byte/tool/latency measurements
```

### Error and cancellation behavior

Candidates receive identical success or structured failure inputs and cannot
change retryability, next actions, cancellation, partial-result behavior, or
exit mapping. Output-write failure remains non-zero with no partial success.

### Security and public boundary

Only synthetic fixtures and identifiers are committed. Candidate code adds no
credential source, network call, filesystem state, parser subprocess, or
dependency. Hostile external text is shared across candidates and remains
untrusted data.

## Implementation slices

1. Add failing result-completeness tests and repair missing parent, explicit
   zero, acknowledgement, contact-name, and real quote-state projections.
2. Freeze synthetic semantic fixtures, answer keys, prompts, scoring, token
   accounting, repetitions, and promotion gates.
3. Create isolated candidate worktrees from the same reviewed commit.
4. Implement C0 measurement and P, L, R, and J candidates without semantic
   changes.
5. Run the pinned evaluation and retain raw per-run evidence.
6. Review the result, accept one candidate or retain C0, then implement only
   the accepted contract on the integration branch.
7. Update durable documentation, golden/compatibility tests, and gates.

## Verification

- Unit and contract tests: all result variants, explicit zero/absence, and
  catalog field completeness.
- Opaque-reference tests: exact next-command identity in every candidate.
- Hostile-output tests: controls, bidi/zero-width, line separators, existing
  escapes, JSON-like and prompt-like printable data.
- Relationship tests: To is not reply; resolved/unresolved/absent reply and
  quote states remain distinguishable; code-like copied notation creates no
  relation.
- Agent readiness: exact answers, zero post-processing, bounded discovery and
  tool steps.
- Required profiles: `task check`, plus `task public:check` for committed
  publishable artifacts.

## Rollout and rollback

Experimental worktrees have no rollout. If a winner is accepted, compatibility
and migration behavior are decided before integration. Until then C0 remains
the default and rollback is unnecessary.

## Documentation promotion

An accepted result must update the selected-presentation text and compatibility
version in the theses, product contract, architecture, security model, harness,
and agent-readiness validation. Measurements and temporary execution details
remain in this work packet.
