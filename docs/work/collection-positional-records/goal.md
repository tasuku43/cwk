# Work Goal: Apply fixed schemas to repeated collection records

- Status: Accepted
- Owner: Project owner and Codex
- Target: Current implementation cycle
- Related ADRs: None; this propagates the accepted fixed-schema principle

## Outcome

The seven homogeneous Chatwork read collections emit their header, external-text
trust declaration, and fixed schema once, followed by provider-order positional
records. Agents retain canonical references, explicit zero/false/absent values,
quoted external text, bounds, and completeness without repeated per-record field
labels or trust prefixes.

## In scope

- `contacts list`
- `rooms list`
- `members list`
- `personal-tasks list`
- `room-tasks list`
- `files list`
- `contact-requests list`

`files list` is mandatory because file, room, account, and optional message
references form a frequently queried fixed record.

## Non-goals

- Single-record show commands, account/status, or mutation results
- Message presentation changes beyond the already accepted positional schema
- Removing canonical references, bounds, completeness, zeros, `absent`, or text
- Changing semantic/domain/application/infrastructure behavior
- Adding output switches or alternative schemas

## Acceptance criteria

- [ ] Every in-scope collection always emits exactly one fixed schema and one
  `external-text=untrusted escaped` declaration, including empty collections.
- [ ] Every item is one physical provider-order line conforming to its schema.
- [ ] Canonical references remain directly reusable without reconstruction.
- [ ] Optional contact organization and request message remain explicit without
  shifting required positions; missing file message remains `absent`.
- [ ] Hostile names/bodies cannot inject records or schema lines.
- [ ] Existing single-record and machine-readable contracts remain unchanged.
- [ ] Representative multi-item fixtures show a token reduction under one pinned
  tokenizer; semantic accuracy remains the eligibility condition.
- [ ] `task check` passes and changes are committed in coherent slices.

## Completion definition

The work ends when all seven renderers, goldens/contracts, hostile and canonical
tests, token evidence, documentation, and the final repository gate agree on one
fixed schema per collection.
