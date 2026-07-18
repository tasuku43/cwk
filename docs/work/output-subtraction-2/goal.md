# Work Goal: Remove non-actionable success-output metadata

- Status: Complete
- Owner: Project owner
- Target: One bounded post-completion output audit
- Related ADRs: None

## Outcome

Every one of the 33 public Chatwork task success routes emits the smallest
direct task result that still preserves the facts an agent needs to understand
the outcome, reuse exact canonical references, recognize bounded or incomplete
context, distinguish uncertainty, and supervise mutations safely. The repeated
schema/task header, a standalone coverage record and its provider-oriented kind,
empty provider profile components, and any other field proven irrelevant in
this single audit are absent.

## Why now

Live use of `account show`, `account status`, `contacts list`, `rooms list`, and
`personal-tasks list` showed that the first subtractive projection still emits
format identity, command identity, provider coverage vocabulary, and empty
organization components that repeat known context or do not change an agent's
next decision. The project owner explicitly chose another subtraction pass and
asked for the remaining commands to be inspected under the same rule.

## Non-goals

- Changing command paths, inputs, roles, effects, reference identity,
  authentication, provider calls, or failure presentation.
- Removing positive limits, completeness/partiality, unresolved relations,
  explicit zero/false values, mutation outcomes, or external-text trust framing
  when they affect interpretation or a next action.
- Parsing or rewriting raw Chatwork message bodies.
- Adding a new compression grammar, output mode, adaptive detail level, or a
  second optimization/benchmark round.
- Performing live mutations or contact-request actions for presentation
  inspection; mutation results use synthetic fixtures.

## Acceptance criteria

- [x] All 33 public Chatwork success routes and every domain result variant are
  represented in an explicit keep/drop/conditional audit.
- [x] Normal success output has no schema/task preamble and no standalone
  coverage record or provider-oriented coverage kind.
- [x] Collection bounds/completeness and message unresolved-relation facts are
  retained in the closest task record without making partial output look
  complete.
- [x] Empty and task-irrelevant fields identified by the audit are absent,
  while canonical references, semantic relationships, explicit zero/false,
  mutation outcomes, and trust framing remain mechanically enforced.
- [x] Root/scoped agent discovery, commands, authentication, failures, effects,
  stdout/stderr ownership, and exit behavior are unchanged.
- [x] Live read-only representative commands need no post-processing and match
  the reviewed contract; synthetic fixtures cover mutation outcomes.
- [x] `task check`, `task security`, and `task public:check` pass.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1, 3, 4, 5, and 8
- Product contract: `docs/01_product_contract.md`, Agent-output axioms
- Architecture: `docs/02_architecture.md`, Semantic and presentation boundary
- Security: `docs/03_security_model.md`, Output and terminal safety
- Validation: `docs/09_agent_readiness_validation.md`

## Completion definition

This work is complete after one finite audit of the current 33 success routes,
the selected removals and required facts are enforced by active tests, the
public compatibility decision and examples are updated, live read-only and
synthetic mutation evidence agree, all required profiles pass, and no live
output, credential, or temporary diagnostic remains in the repository. It does
not continue into another optimization round.
