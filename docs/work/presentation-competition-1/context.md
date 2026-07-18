# Work Context: Select the next agent presentation by evidence

This file records verified behavior. Live identifiers, names, email addresses,
credentials, and private message content are intentionally excluded.

## Current behavior

- The integrated default is `cwk-task-projection/1`, a simple subtractive task
  projection implemented from candidate P's renderer seed. It is a replaceable
  public presentation contract, not a thesis or a benchmark winner.
- The projection emits catalog-declared task fields and validated canonical
  references directly. It removes the global alias policy, reference
  dictionary, generic result wrapper, task-irrelevant profile/default fields,
  zero limits that do not express a bound, and redundant authored coverage
  prose.
- Shared domain validation now rejects a reference whose structural kind does
  not match its semantic room, account, message, task, file, invite, request,
  recipient, reply, or quote position before presentation.
- A live isolated two-account room showed that explicit To and reply notation
  are parsed correctly. To was not strengthened into reply, and the reply
  resolved to the exact message in the returned window.
- Typed To, reply, and quote facts remain explicit. Raw Chatwork notation may
  remain inside a declared message body, but the projection neither emits it
  as additional semantic structure nor reparses it. External message and
  sender text remains visibly framed as untrusted data.
- Explicit zero, false, empty, absent, acknowledgement, parent, bound, and
  completeness facts remain present when the catalog contract distinguishes
  them from omission.
- Public command paths, inputs, effects, authentication, provider calls,
  structured failures, exit statuses, and JSON help are unchanged. The text
  schema change is intentionally breaking and is documented in
  [decision.md](decision.md).
- No external post-processing was needed during the live observation. That
  remains a hard eligibility condition.
- The frozen competition selected no eligible challenger. The owner-selected
  projection is a separate product and compatibility decision; candidate P
  supplied a seed but did not pass the frozen gates.

## Relevant structure

- Entry point: `cmd/cwk/main.go`
- Semantic result: `internal/domain/chatwork`
- Application use cases: `internal/app/chatworkcmd`
- Provider mapping and notation: `internal/infra/chatworkapi`
- Catalog: `internal/cli/chatwork_catalog.go`
- Presentation renderer: `internal/cli/capsule/capsule.go`
- Presentation tests: `internal/cli/capsule/capsule_test.go`
- Agent contract: `docs/09_agent_readiness_validation.md`
- Compatibility decision: [decision.md](decision.md)
- Frozen-result audit: [evaluation-audit.md](evaluation-audit.md)
- Retained evidence and commit bindings:
  [evidence/manifest.json](evidence/manifest.json)

## Constraints

- Candidates consume the same typed result; they cannot change provider
  parsing, relation truth, bounds, or answer keys to improve their score.
- Canonical references remain validated exact bytes and must be recoverable
  without decoding or reconstructing a display alias.
- External text stays visibly separated as untrusted data. Printable
  prompt-like meaning is not claimed to be sanitized.
- Complete and bounded/partial results remain distinguishable.
- Output changes cannot hide explicit zero, acknowledgement, uncertainty, or
  missing context when those values are part of the task contract.
- The competition adds no third-party CLI dependency.
- All committed fixtures are synthetic and publishable.

## Resolved decisions

- [x] Competition 1 did not establish a quality/token winner. The project
  owner selected a task-shaped projection as a product and compatibility
  direction rather than claiming a benchmark promotion.
- [x] The normalized ledger and relationship-first timeline remain losing
  experimental candidates; neither becomes a default or specialization in
  this work packet.
- [x] Raw Chatwork notation is not promoted into presentation-authored
  semantic structure. It may remain inside the declared untrusted body while
  reviewed typed To, reply, and quote facts carry relationship meaning.
- [x] One fixed task projection covers collections, mutation outcomes, and
  relationship-heavy messages. It uses catalog task selection rather than an
  output-size heuristic, adaptive summary, or agent-selected detail mode.

## Thesis evidence

- Repeated design decision or point of agent confusion: the generic renderer
  displays every populated field rather than the fields needed by the task.
- User outcome or friction observed in the minimal slice: a room list and a
  short message thread required scanning repeated defaults and aliases before
  reaching the actionable facts.
- Code workaround being avoided: adding more field-by-field omission branches
  without a task-output completeness contract and shared evaluation.
- Current theses already resolve the direction: semantics precede
  presentation, presentation is replaceable, and claims require executable
  evidence. The selected projection implements subtraction only after shared
  domain kind validation, so it does not route around that semantic boundary.

## Reproduction or observation

Live observation used an isolated, clearly named room containing only existing
test accounts. It exercised room creation, plain messages, explicit To,
explicit reply, file upload, and list retrieval. The commands used the normal
public CLI with the PAT supplied only to the command process. The transcript is
not retained because it contains live opaque identifiers; equivalent synthetic
fixtures must reproduce every semantic fact.

The synthetic runner was conformed against `codex-cli 0.145.0-alpha.18` and
`gpt-5.6-terra` in an empty temporary workspace. The first attempts exposed an
obsolete CLI pin, two newly declared usage fields, and the CLI's single-shell
wrapper around `cwk`; each mismatch failed closed before the frozen run. The
final run used only public `cwk` help and `rooms list`, exactly matched the
answer key, and passed deterministic transcript replay. No repository source,
Chatwork credential, live identifier, or live provider data was supplied to
the model.

The first scored recovery attempt exposed one remaining event-contract gap:
an intentional fixture exit code 6 is a completed observed command for the
benchmark, while Codex labels its command item `failed`. Runner v1 rejected it
before deterministic replay. All five submissions produced before that
discovery are retained outside the scored corpus. Amendment 1 accepts that
status only with a nonzero exit and restarts every candidate from the amended
common base; no candidate result from runner v1 is promoted into the score.

The completed frozen schedule produced 50 amended workflow runs. The strict
scorer found no eligible challenger. The audit then found that the thread
oracle omits a real simultaneous To relation and that the recovery prompt,
answer-key path, and cold-discovery command budget are ambiguous. The original
scores and invalidated attempts remain unchanged in the committed evidence;
[evaluation-audit.md](evaluation-audit.md) records the exact chronology and
resource calculations.

On 2026-07-19 the project owner separately chose the simple subtractive task
projection. Integration used candidate P commit
`b804f8efdaeb318e94b1b5e9d6144c00149e4674` as a seed, then added semantic
reference-kind enforcement and removed redundant coverage prose. This later
subtraction was not attributed retroactively to P's frozen benchmark result.

## Security and public-boundary notes

- Assets and side effects: temporary Chatwork rooms, messages, tasks, and one
  small synthetic text file were created in authorized test scope.
- Credentials: no token is accepted by argv or committed output. Evaluation
  and CI must use synthetic adapters only.
- Personal data: live account/profile fields must never enter fixtures, docs,
  snapshots, or benchmark artifacts.
- Dependencies and destinations: no new dependency or production destination
  was added by the candidates or selected projection.
- Publication: every candidate artifact and answer key must pass public
  boundary checks.
- Pending verification: the final `task check`, `task security`, and
  `task public:check` results and candidate-worktree cleanup are not yet
  recorded in this packet.

## Glossary

- **Baseline C0:** the frozen `cwk-context-capsule/1` competition baseline,
  retained as the protocol result but replaced by the later owner decision.
- **Task projection:** a renderer that emits only the declared facts needed for
  one catalog outcome.
- **Scope hoist:** emitting an exact command-known parent once rather than on
  every child record.
- **Semantic answer key:** presentation-independent facts that every eligible
  candidate must preserve.
