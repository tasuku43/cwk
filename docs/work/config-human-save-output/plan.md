# Work Plan: Human-readable config save result

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Keep the state calculation and save boundary unchanged. Replace only the CLI
success projection with Concept A, rename catalog success counts from
enabled/disabled to visible/hidden, combine stale and legacy removal into one
conditional cleanup count, and remove fingerprint from confirmed success.
Retain the existing fingerprint function for `doctor` and uncertain faults.

## Alternatives considered

### Aligned label/value list

Rejected because a three-row report adds visual weight without improving the
single decision a person needs after Enter: whether the display was saved and
what changed.

### Complete audit block

Rejected because zero-valued migration internals and a 71-character
fingerprint dominate the ordinary success path. Those details remain available
through `doctor` when reconciliation is actually needed.

## Design

### Public contract

- Exact command, role, effect, target, input, and exit behavior: unchanged.
- Confirmed-save final result suffix after terminal restoration: two fixed
  natural-Japanese lines plus one conditional cleanup line.
- Catalog fields: `status`, `visible`, `hidden`, `changed`, and conditional
  `cleaned`.
- Unchanged exit text, typed faults, and recovery commands: unchanged.
- Compatibility: deliberate pre-`v0.1.0` text-output and scoped agent field
  refinement; no released consumer contract exists.

### Layer changes

- Domain/application/infrastructure: none.
- CLI: success presentation, catalog declaration, and golden tests.
- Documentation/harness: move normal fingerprint ownership fully to
  uncertain-save and `doctor` reconciliation.

### Error and cancellation behavior

No error path changes. Output failure after confirmed persistence still routes
to `doctor`; unclassified mutation outcomes still contain the exact candidate
fingerprint and never claim success.

### Security and public boundary

No authority, credential, path, or side-effect boundary changes. The natural
words “表示” and “非表示” deliberately avoid suggesting that this cognitive
filter grants or revokes Chatwork permissions.

## Implementation slices

1. Pure success-output helper and exact transcript tests.
2. Catalog and lifecycle expectation updates.
3. Product, architecture, security, harness, readiness, the maintainer's
   shortened README, and Skill propagation.
4. Real synthetic PTY replay and repository gates.
5. Include the final result in the `v0.1.0` commit and release.

## Verification

- Focused: `go test ./internal/cli`.
- Transcript: temporary config home, interactive Enter save, exact final lines.
- Reconciliation: existing uncertain-save/doctor tests remain unchanged.
- Required: `task check`, plus security/release/public profiles for `v0.1.0`.

## Rollout and rollback

The change ships only in the first public release. Rollback before tag is an
ordinary code change; after tag, later output changes require a new version and
must not rewrite `v0.1.0`.

## Documentation promotion

Update product, architecture, security, harness, readiness, README, and the
repository `add-capability` skill. Preserve the earlier selector work packet as
historical evidence of the superseded first success projection.
