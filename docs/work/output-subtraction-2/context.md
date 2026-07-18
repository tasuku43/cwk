# Work Context: Remove non-actionable success-output metadata

This file records verified facts and open audit questions. Live identifiers,
names, message bodies, credentials, and private provider output are excluded.

## Current behavior

- Chatwork successes are rendered by `internal/cli/capsule.Render` through one
  typed `chatwork.Result` switch.
- Every route currently starts with `cwk-task-projection/1 task=<task>`.
- Collection-like results currently emit a separate `coverage` line containing
  a semantic/provider kind, optional positive limit, completeness, and for
  message results an unresolved-relation count.
- Live output shows empty organization name/department components and repeats
  command identity already known from the invocation and result noun.
- The selected projection already omits aliases, icon URLs, empty room
  descriptions, zero coverage limits, duplicated coverage prose, and unrelated
  provider fields.

## Relevant structure

- Semantic result: `internal/domain/chatwork`
- Provider mapping: `internal/infra/chatworkapi`
- CLI catalog: `internal/cli/chatwork_catalog.go`
- Success renderer: `internal/cli/capsule/capsule.go`
- Active projection tests: `internal/cli/capsule/capsule_test.go`
- Evaluation fixtures: `tools/presentationeval`
- Governing output contract: `docs/00_theses.md` through
  `docs/04_harness.md`, `docs/08_external_api_contracts.md`, and
  `docs/09_agent_readiness_validation.md`

## Constraints

- Presentation may remove representation but not semantic truth.
- Canonical references remain exact reusable bytes and never become aliases.
- Bounds, incompleteness, uncertainty, explicit zero/false, and mutation
  acknowledgement remain visible when their absence could change an answer or
  next action.
- External text remains structurally framed as untrusted data.
- Historical Competition 1 evidence is immutable; only current contract,
  examples, active fixtures, and current evaluation inputs may change.
- The audit is finite: current result variants and representative live
  read-only outputs are examined once.

## Unknowns

- [ ] Which nonempty account, room, task, file, invite, and request profile
  fields change a supported task answer or declared next action?
- [ ] Whether `complete=true` must remain explicit for every complete collection
  or can be safely defaulted without increasing agent inference.
- [ ] Which scope references are truly duplicated by command input versus
  required for detached output and canonical next-command reuse?
- [ ] Which historical versus active snapshots contain the old schema marker?

## Thesis evidence

- Repeated decision: output metadata must justify an agent decision rather than
  only describe the renderer or provider implementation.
- Observed friction: one- and two-line results spend a large fraction of their
  tokens restating schema and command identity.
- Workaround avoided: route-local omission branches without one full-result
  audit and negative canaries.
- Governing direction: operational closure, semantics before presentation,
  replaceable presentation, and executable claims support a second subtractive
  compatibility decision without changing provider/domain semantics.

## Reproduction or observation

Representative read-only commands are run with the PAT supplied only by the
user's command environment. Exact live output is not committed. Mutation
routes are rendered from publishable synthetic `chatwork.Result` fixtures.

## Security and public-boundary notes

- No new side effect, credential source, destination, dependency, or file
  format is introduced.
- Live inspection is read-only and excludes incoming-request actions.
- Provider text remains untrusted and hostile-output tests remain mandatory.
- Only synthetic examples and fixtures may be committed.

## Glossary

- **preamble:** renderer/schema and task identity emitted before task data.
- **coverage kind:** the current label describing provider collection/window
  shape; distinct from actionable limit, completeness, and uncertainty facts.
- **closest task record:** the collection/message line whose noun already
  identifies what the bound or uncertainty describes.
