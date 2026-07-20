# Work Plan: Preserve multiple explicit message replies

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Replace the semantic single reply pointer with an ordered reply-relation slice. Parse every complete `[rp]` tag, resolve each edge independently, traverse every unique same-room target under the existing budget, and render the existing scalar form for one value or compact bracket form for multiple values.

## Alternatives considered

### Keep the first reply only

This would avoid a model change but silently discard proven provider facts and leave thread reconstruction incomplete.

### Add a second presentation-only parser

Parsing raw body text in the renderer would violate layer ownership, trust boundaries, and the no-post-processing outcome.

## Design

### Public contract

`messages list` and `messages show` retain the `reply` field. One edge renders exactly as today. Two or more render in provider notation order as `reply=[value,value]`. Each value uses the existing resolved local sequence, resolved supplemental `message-ref:`, or unresolved `?message-ref` grammar. `to` continues to list every deduplicated explicit recipient. A complete multi-reply set is not unknown.

### Layer changes

- Domain: message owns an ordered `Replies []Relation` collection and validates every reply plus duplicate/unknown-state invariants.
- Application: resolve, select one-hop neighbors, clone, enqueue, mark, and validate all reply edges.
- Infrastructure: parser accumulates every complete `[rp]` relation and response mapping carries the slice.
- CLI and catalog: render scalar/list reply values; update field descriptions and semantic fixtures.

### Data and control flow

Provider body → notation parser → ordered typed replies/recipients → domain validation → source-window resolution → filter/context selection → bounded recursive closure → scalar/list projection.

### Error and cancellation behavior

A malformed recognized tag still discards all partial relation facts for that message and marks the set unknown. Valid but unavailable targets remain unresolved or become typed relation gaps. Existing provider faults, cancellation, and all-or-no-result rules remain unchanged.

### Security and public boundary

No credential, destination, effect, dependency, or persistent state changes. Raw provider text never becomes presentation-authored structure without validated notation parsing.

## Implementation slices

1. Failing parser and semantic-output fixtures
2. Domain reply collection and validation
3. Application selection and recursive closure
4. Projection and agent contract
5. Durable documentation and harness evidence

## Verification

- Unit and contract tests: parser, domain, application, renderer, runtime, agent help
- Negative side-effect tests: unchanged read-only/PAT checks
- Structured output and hostile-output tests: malformed later tag remains unknown with no partial facts
- Agent-readiness scenario: multi-reply branch reconstructed with zero raw-body parsing
- Manual observation: synthetic end-to-end multiple in-window replies
- Required profiles: `task check`

## Rollout and rollback

This is an intentional pre-1.0 semantic/output correction. Single-reply output is byte-compatible; only formerly unknown valid multi-reply messages gain typed `reply` and `to` fields and reduce unknown counts. Rolling back restores the known data-loss defect.

## Documentation promotion

Update theses, product, architecture, security, external API contract, harness, agent-readiness validation, catalog descriptions, and the add-capability skill.
