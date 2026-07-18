# Work Context: Remove non-actionable success-output metadata

This file records verified facts and open audit questions. Live identifiers,
names, message bodies, credentials, and private provider output are excluded.

## Baseline behavior before this change

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

## Completed result audit

The 33 routes are covered through the following projection cases. These are the
complete decisions for this pass:

| Projection case | Routes | Keep | Drop or condition |
| --- | --- | --- | --- |
| Account | `account show`, request accept | account/room references and display name | Never emit provider organization ID; emit human-readable organization name/department only when non-empty |
| Status | `account status` | unread, mention, and task counts including zero | None |
| Task collection | personal and room task lists | count, positive limit, completeness, task/room/account/message references, body, status, deadline including zero | Provider coverage kind/description |
| Single task | `room-tasks show` | the task facts above | collection count and single-operation coverage |
| Account collection | contacts and members | count, completeness, canonical references, name, member role | Empty organization shells and unrelated profile fields |
| Room collection | `rooms list` | count, completeness, room reference, name, type, role, unread/mention/task counts | Unrelated description/icon/activity counters |
| Single room | `rooms show` | one direct room record | collection count and single-operation coverage |
| Created/affected reference | create/update/delete task results | route-specific outcome and exact reusable reference | Generic task echo |
| Acknowledgement | room leave/delete and request reject | route-specific outcome plus typed target reference | Constant `acknowledged=true` and generic `target-ref` label |
| Membership counts | `members replace` | all role counts including zero | None |
| Message collection | `messages list` | count, task-level `recent`/`changes` window, positive limit, completeness, unresolved count, message/scope/sender references, sender/time, typed relations, body | Provider coverage kind/description, constant To resolution state, provider relation external IDs |
| Single message | `messages show` | one direct message record and typed relation states | collection count, aggregate uncertainty, and single-operation coverage |
| Room-scoped creation | message/task/file create | created reference(s), count where cardinality varies, parent room reference | None |
| Read state | mark read/unread | route-specific outcome, unread and mention counts including zero | Generic result label |
| File collection/single file | file list/show | count/bounds for list; file/room/uploader/message references, name and size; requested non-empty download URL on show | Empty or list-only download URL, single-operation coverage |
| Invite link | show/create/update/delete | route-specific outcome, invitation reference, public state; URL/approval/description when enabled and present | Empty enabled-only fields while disabled |
| Contact request collection | request list | count, positive limit, completeness, request/account references, name, non-empty message | Empty message |

`complete=true` remains explicit because it distinguishes a documented complete
collection from a bounded provider window without a defaulting rule. Repeated
room references remain on items: the current domain/application contract does
not prove that every provider item has the command's room, and detached items
need the exact parent for follow-up actions. `limit-time=0`, false state counts,
and `unresolved-relations=0` also remain explicit absence/state facts.

For message lists, the provider-oriented coverage kind is replaced by
`window=recent|changes`. Removing it entirely would make the latest-window and
differential-window tasks indistinguishable.

## Resolved boundaries and follow-up

- Active renderer, golden/tests, catalog, README, governing documents, and the
  capability Skill are updated. Completed Competition 1 evidence and frozen
  evaluation protocol remain unchanged history.
- Current success-text compatibility is carried by the binary/release,
  documented field contract, catalog, and goldens rather than an in-band schema
  marker repeated on every result.
- The room `tasks` field may need a separate semantic audit against provider
  `task_num` versus `mytask_num`. That changes field meaning and is intentionally
  outside this presentation-only subtraction pass.

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
