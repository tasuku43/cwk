# Work Plan: Select the next agent presentation by evidence

- Status: Complete
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)
- Decision: [decision.md](decision.md)
- Evaluation audit: [evaluation-audit.md](evaluation-audit.md)
- Evidence: [evidence/manifest.json](evidence/manifest.json)

## Chosen approach

Separate correctness repair from presentation optimization. First make every
catalog-declared task result observable with synthetic contract tests. Then
freeze a shared semantic fixture corpus and evaluation protocol. Implement the
baseline and challengers in isolated worktrees, preserve every result, and
accept the frozen scorer's conclusion when no candidate is eligible.

That competition is complete and selected no winner. The subsequent
implementation follows the owner's explicit compatibility decision: use P as
the seed for a simple subtractive task projection, harden semantic reference
kinds before presentation, and remove redundant coverage prose. This does not
reclassify P as having passed the frozen gates.

## Candidate concepts

### C0: Current context capsule

The compatibility baseline. It uses a document-level reference dictionary,
generic typed result blocks, explicit relationship states, and visibly framed
external text. It is expected to score well on explicitness and poorly on
task-irrelevant repetition.

### P: Task-shaped projection, selected implementation seed

Each catalog task owns a fixed projection containing only its declared output
facts. Known scope is hoisted once, absent optional facts are omitted, and
explicit zero is preserved where the task contract distinguishes it from
absence. It became the closest seed for the owner-selected product direction.
Its frozen score remained ineligible; the integrated contract includes later
domain hardening and subtraction that were not part of that score.

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
effect, impact, provider call, or failure contract. The owner accepted an
intentional text-schema compatibility break from `cwk-context-capsule/1` to
`cwk-task-projection/1`. The reviewed projection preserves the public stream,
exit, completeness, trust, and canonical-reference contracts; migration and
rollback behavior are recorded in [decision.md](decision.md).

### Layer changes

- Domain: no candidate-specific type or grammar; shared validation now
  enforces the contextual kind of every semantic reference before rendering.
- Application: no candidate-specific selection, joining, or omission.
- Infrastructure: no candidate-specific parsing or wire changes.
- CLI: shared task-result completeness repair followed by isolated renderers;
  the integrated renderer is a fixed catalog-task projection with redundant
  coverage prose removed.

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

1. **Completed:** add result-completeness tests and repair missing parent,
   explicit zero, acknowledgement, contact-name, and real quote-state
   projections.
2. **Completed:** freeze synthetic semantic fixtures, answer keys, prompts,
   scoring, token accounting, repetitions, and promotion gates.
3. **Completed:** create isolated candidate worktrees from the same reviewed
   commit.
4. **Completed:** implement C0 measurement and P, L, R, and J candidates
   without changing their shared semantic inputs.
5. **Completed:** run the pinned evaluation and retain raw, losing, failed,
   invalidated, scored, and static evidence. The frozen result is no eligible
   challenger, as documented in [evaluation-audit.md](evaluation-audit.md).
6. **Completed:** record the separate owner compatibility decision, integrate
   the P seed, harden domain reference kinds, remove redundant coverage prose,
   and add subtractive projection contract tests.
7. **Completed:** propagate governing documents, pass the repository,
   security, and public-boundary gates, and remove every clean experimental
   worktree while retaining its branch and evidence commit.

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
- Required profiles: `task check`, `task security`, and `task public:check`
  for committed publishable artifacts.

## Rollout and rollback

The experimental candidates have no rollout. The selected task projection is
an intentional breaking text-schema change with no persisted-state or provider
data migration. Mixed-version consumers branch on the first-line schema, and
rollback is the previous C0 binary. Candidate worktrees remain temporary and
must be removed only after their retained evidence and clean status are
verified.

## Documentation promotion

The selected-presentation text and compatibility version must agree across the
theses, product contract, architecture, security model, harness, external API
contract, agent-readiness validation, and capability skill. That propagation
and all required gates must be complete before this packet is accepted.
Competition measurements, defects, and execution details remain in this work
packet and do not turn the owner decision into a benchmark win.
