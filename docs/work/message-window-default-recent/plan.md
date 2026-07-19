# Work Plan: Recent message window by default

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Resolve the optional window at CLI request assembly: initialize a
`messages list` request as recent, then let an explicit catalog-validated
`--window recent|changes` value override it. Keep the domain request shape,
application service, infrastructure `force` mapping, coverage types, and
renderer unchanged. Reorder the help values to default-first
`recent|changes`, remove redundant recent flags from common examples, and add
explicit differential regression coverage.

## Alternatives considered

### Remove `--window`

Rejected because provider differential retrieval is a valid bounded outcome
for polling callers. It should be explicit, not unavailable.

### Switch to recent only when `--limit` or `--sender` is present

Rejected because the room's current conversation is also the natural outcome
of the unfiltered command. Making omission depend on unrelated selection flags
would be harder for agents to predict.

### Change the infrastructure zero value

Rejected because the user-facing default belongs to CLI task interpretation.
The typed `ForceRecent` field and adapter should continue to map an explicit
boolean to the exact provider query without owning command-default policy.

## Design

### Public contract

- Command: existing `messages list`, `RoleAct`, `EffectRead`.
- Required identity: unchanged exact `--room <chatwork-room>`.
- Optional window: `recent|changes`; omission equals `recent`.
- Provider behavior: recent sends `force=1`; changes sends no `force` query.
- Output: unchanged bounded message result whose existing `window` field
  reports the selected source semantics.
- Compatibility: intentional pre-1.0 default change; explicit invocations keep
  their existing meaning.

### Layer changes

- Domain: none.
- Application: none.
- Infrastructure: no production change; retain request-mapping tests for both
  boolean values.
- CLI/catalog: default request initialization, default-first help vocabulary,
  end-to-end omission and explicit-changes tests.
- Documentation/harness: record the common recent outcome and explicit
  differential alternative.

### Data and control flow

```text
omitted --window -> CLI ForceRecent=true  -> one GET .../messages?force=1
--window recent  -> CLI ForceRecent=true  -> one GET .../messages?force=1
--window changes -> CLI ForceRecent=false -> one GET .../messages
```

The existing adapter maps the response to latest or differential coverage;
presentation renders that typed coverage without inspecting argv.

### Security and public boundary

No trust boundary changes. The same PAT-only read reaches the same fixed
Chatwork destination with unchanged bounds and one attempt. Synthetic tests
contain no live account data.

## Implementation slices

1. Work packet and failing omission/explicit-value contract tests.
2. CLI default and catalog help update.
3. Durable documentation, harness, Skill, and readiness propagation.
4. Focused/full verification, review, and commit.

## Verification

- Catalog and exact human/agent help tests for `recent|changes` and default
  wording.
- CLI runtime tests proving omission and explicit recent set `ForceRecent`,
  while explicit changes clears it before authentication/port execution.
- Infrastructure request tests retaining exact `force=1` versus absent query.
- Existing message semantic, selection, presentation, and canonical-reference
  tests remain green.
- Active readiness examples use omission for the common latest-window task and
  explicit changes only for differential retrieval.
- Required gate: `task check`.

## Rollout and rollback

No persisted state changes. Callers that require differential semantics must
add `--window changes`. Reverting the code restores the old default but would
also require reverting the public help and compatibility note.

## Documentation promotion

Update the theses, product contract, architecture, harness, README,
agent-readiness validation, and `$add-capability` so the default does not live
only in parser code or this packet.
