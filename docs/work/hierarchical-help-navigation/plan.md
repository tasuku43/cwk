# Work Plan: Hierarchical human help navigation

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Project the same catalog into two bounded human views. Root text help partitions
catalog entries into directly runnable single-word commands and first-seen
top-level namespaces, renders direct commands first, and preserves
catalog-relative order within each section. Namespace text help uses
`Catalog.Select`, strips the selected prefix for local readability, and
preserves catalog order. Before normal command matching, a trailing `--help` or
`-h` is normalized to the existing `help <selector>` task only when the
preceding words are a valid exact path or namespace. Exact human help renders
the existing typed input metadata, so repeatable and reference-bearing inputs
do not require machine JSON or a second prose registry.

## Alternatives considered

### Keep the flat root and only add `rooms --help`

This fixes the surprising rejection but leaves the exhaustive root list and
duplicated namespace prefixes that prompted the request.

### Add curated namespace descriptions

Descriptions could make root richer, but storing them outside command specs
would create a second registry, while repeating them on every command would add
validated duplication. Canonical namespace names and derived command counts
provide a smaller truthful index; exact summaries remain in namespace help.

### Group schema-v3 agent root help too

This would remove the exact outcome facts agents currently use to choose a
task, forcing root, namespace, and exact requests for an unknown outcome. It
would violate the existing two-invocation budget, so machine help remains
unchanged.

## Design

### Public contract

- `cwk --help` and `cwk help` show local commands plus top-level namespaces.
- `cwk <namespace> --help` and `cwk help <namespace>` show only commands in
  that namespace, using names relative to the namespace.
- `cwk <exact-command> --help` and `cwk help <exact-command>` show the existing
  exact task contract plus its catalog-derived human input facts.
- Namespace nodes have no operation effect, role, inputs, output, or handler;
  only the existing `help` utility executes.
- Schema-v3 agent help, command execution, error format, exit codes, and all
  provider workflows remain compatible.

### Layer changes

- Domain: none.
- Application: none.
- Infrastructure: none.
- CLI and catalog: add catalog-derived text index projection, namespace text
  renderer, exact input projection, and safe trailing-help normalization; align
  the help task's invalid-selector recovery wording with namespace support.

### Data and control flow

```text
validated argv
  -> trailing help selector validation through Catalog.Select
  -> existing local help command
  -> root, namespace, or exact catalog projection
  -> complete stdout write
```

### Error and cancellation behavior

Only a valid catalog path or word-boundary namespace is normalized. Unknown or
partial prefixes continue through normal routing and fail with the existing
typed invalid-input error and root-help next action. Context cancellation and
complete-write behavior remain owned by the existing CLI boundary.

### Security and public boundary

The change performs no external I/O and does not initialize authentication.
No private data, dependency, schema, destination, or persistence is added.
Catalog validation rejects unsafe structural runes and invalid UTF-8 from every
input source before exact help renders a name.

## Implementation slices

1. Add failing root/namespace/alias/compatibility tests.
2. Implement catalog-derived root and namespace text projections.
3. Normalize valid trailing namespace/exact `--help` forms.
4. Project exact human inputs and direct machine navigation from the catalog.
5. Promote the durable discovery distinction to governing documents and README.
6. Run focused tests, manual transcript checks, and `task check`.

## Verification

- Unit and contract tests: `go test ./internal/cli`.
- Negative side-effect tests: existing and extended PAT-laziness help tests.
- Opaque-reference and complete-pagination tests: unchanged; full gate covers
  them.
- Structured output, hostile-output, and recovery tests: agent schema snapshots
  and full gate remain unchanged.
- Agent-readiness scenario and discovery-round-trip count: machine help remains
  root plus one scope for unknown outcomes; human `rooms` path is root plus one
  namespace request.
- Manual observation: root, namespace suffix, namespace help selector, exact
  suffix, exact selector, unknown namespace, and agent root JSON.
- Required profiles: `task check`.
- Generated-diff or artifact checks: included by the full gate.

## Rollout and rollback

This is a pre-1.0 human text-help contract change with no external state. Safe
rollback restores the flat renderer, removes namespace suffix normalization,
and removes exact input/navigation projection. The stricter catalog input-name
validation and corrected `help` recovery may remain as independent hardening;
if reverted too, their hostile and recovery tests must be reverted with them.
Command execution, the root agent index, and non-help scoped contracts are
unaffected.

## Documentation promotion

- Axiom 2: distinguish hierarchical human discovery from exact-outcome machine
  discovery.
- Product contract: state the root/namespace/exact human path.
- Architecture: document catalog-derived grouping and suffix normalization.
- Harness: add the human navigation contract and enforcement claim.
- README: show the natural namespace and exact help invocations.
