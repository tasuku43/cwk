# Work Plan: Remove non-actionable success-output metadata

- Status: In progress
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Audit the current typed success result once, by result variant rather than by
provider endpoint. Use representative live read-only commands to catch real
empty/default noise and synthetic fixtures to cover mutations and uncommon
states. Remove only fields that do not affect the task answer, next canonical
action, completeness/uncertainty, explicit state, or trust boundary.

The project owner has already selected the first rules: remove the schema/task
preamble; remove the standalone coverage line and coverage kind; retain
actionable limit/completeness/uncertainty on the closest task record. Additional
removals require evidence from the finite audit.

## Alternatives considered

### Keep the self-describing preamble

This helps detached output identify an old grammar, but every current consumer
already has the invocation/result noun and no supported workflow branches on
the line. The owner chose lower normal-path cost; compatibility remains in
documentation, tests, help, and release version rather than repeated data.

### Add terse aliases or a second compact mode

This could reduce bytes further but creates another grammar, discovery choice,
and identity risk. It is outside this subtraction pass.

## Design

### Public contract

Command and semantic contracts stay unchanged. Only successful Chatwork text
presentation changes. A collection line owns its count and any actionable
positive limit, completeness state, and task-specific uncertainty. Noncollection
results begin directly with their result noun or mutation outcome. External
text remains `untrusted`, and all emitted references remain canonical.

### Layer changes

- Domain: no semantic changes expected.
- Application: no use-case changes expected.
- Infrastructure: no provider or notation changes expected.
- CLI/catalog: renderer field selection, active text contract tests, and any
  catalog field removal justified as task-irrelevant.

### Error and cancellation behavior

Unchanged. This work changes successful Chatwork text only. Structured errors,
retryability, next actions, exit mappings, cancellation, and complete-write
behavior remain fixed.

### Security and public boundary

Trust framing, canonical identity, stdout/stderr ownership, output bounds, and
secret exclusion remain mandatory. Live evidence is never committed.

## Implementation slices

1. Freeze the result-variant audit and failing active contract tests.
2. Remove the preamble and fold actionable coverage into task records.
3. Remove audited empty/task-irrelevant fields and align catalog declarations.
4. Update current documentation/examples/evaluation fixtures without rewriting
   historical evidence.
5. Replay live read-only outputs, agent help/journey, and required gates.

## Verification

- Unit/contract: all 33 routes, every result variant, determinism, and exact
  catalog-field agreement.
- Negative canaries: removed preamble/kind/profile fields stay absent.
- Semantic safety: canonical references, relationships, explicit zero/false,
  bounds/completeness/uncertainty, acknowledgements, and hostile text remain.
- Runtime: representative live read-only outputs, no retained transcript.
- Agent readiness: root/scoped help budget and direct result interpretation.
- Required profiles: `task check`, `task security`, `task public:check`.

## Rollout and rollback

This is an intentional pre-1.0 breaking text-presentation change with no
provider or local-state migration. Rollback is the previous binary. Consumers
must rely on documented semantic fields/help rather than the removed preamble.

## Documentation promotion

Update the theses, product contract, architecture, security model, harness,
external API contract, agent-readiness validation, capability Skill, README,
and current migration record. Preserve completed historical work packets and
raw competition evidence as history.
