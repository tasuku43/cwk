# Work Context: Select the next agent presentation by evidence

This file records verified behavior. Live identifiers, names, email addresses,
credentials, and private message content are intentionally excluded.

## Current behavior

- The accepted baseline is `cwk-context-capsule/1`; it is a replaceable public
  presentation contract, not a thesis.
- `rooms list` emits a complete collection and exact room references, but also
  emits a global alias policy, verbose complete-coverage prose, icon URLs,
  empty descriptions, and several zero/default counters that are not required
  to select a room for the next task.
- A live isolated two-account room showed that explicit To and reply notation
  are parsed correctly. To was not strengthened into reply, and the reply
  resolved to the exact message in the returned window.
- The same message output repeated the known room scope, identical sender
  identity, `updated=0`, and three separate empty relation states for each
  message. It also retained the raw provider To/reply notation after emitting
  typed relations, making the agent read the same relationship twice.
- The provider-generated room-creation message is returned as a long notation
  string. It is safely framed as untrusted data, but it is expensive and does
  not expose a task-shaped interpretation.
- `messages send`, task creation, and file upload return only the created
  object reference even though their catalog output also declares the parent
  room reference.
- File listing emits a blank download URL. Task listing emits a zero deadline
  and an empty deadline type when no deadline exists.
- Generic account rendering can expose profile fields irrelevant to contact or
  member selection, increasing both token cost and personal-data exposure.
- No external post-processing was needed during the live observation. That
  remains a hard eligibility condition.

## Relevant structure

- Entry point: `cmd/cwk/main.go`
- Semantic result: `internal/domain/chatwork`
- Application use cases: `internal/app/chatworkcmd`
- Provider mapping and notation: `internal/infra/chatworkapi`
- Catalog: `internal/cli/chatwork_catalog.go`
- Baseline renderer: `internal/cli/capsule/capsule.go`
- Baseline tests: `internal/cli/capsule/capsule_test.go`
- Agent contract: `docs/09_agent_readiness_validation.md`

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

## Unknowns

- [ ] Whether a task-shaped projection or a relationship-first projection
  produces the best quality/token frontier across both lists and messages.
- [ ] Whether a normalized ledger wins only for large collections and should
  remain an experimental specialization rather than the default grammar.
- [ ] Whether raw Chatwork notation can be omitted from the default message
  projection after all task-relevant typed facts are proven present.
- [ ] Whether one presentation serves mutation outcomes, collections, and
  relationship-heavy messages without adding unpredictable adaptive rules.

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
  evidence. No thesis revision is currently required.

## Reproduction or observation

Live observation used an isolated, clearly named room containing only existing
test accounts. It exercised room creation, plain messages, explicit To,
explicit reply, file upload, and list retrieval. The commands used the normal
public CLI with the PAT supplied only to the command process. The transcript is
not retained because it contains live opaque identifiers; equivalent synthetic
fixtures must reproduce every semantic fact.

## Security and public-boundary notes

- Assets and side effects: temporary Chatwork rooms, messages, tasks, and one
  small synthetic text file were created in authorized test scope.
- Credentials: no token is accepted by argv or committed output. Evaluation
  and CI must use synthetic adapters only.
- Personal data: live account/profile fields must never enter fixtures, docs,
  snapshots, or benchmark artifacts.
- Dependencies and destinations: no new dependency or production destination
  is proposed.
- Publication: every candidate artifact and answer key must pass public
  boundary checks.

## Glossary

- **Baseline C0:** the accepted `cwk-context-capsule/1` renderer.
- **Task projection:** a renderer that emits only the declared facts needed for
  one catalog outcome.
- **Scope hoist:** emitting an exact command-known parent once rather than on
  every child record.
- **Semantic answer key:** presentation-independent facts that every eligible
  candidate must preserve.
