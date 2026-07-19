# Work Plan: Persistent command selection

- Status: Complete
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Add catalog-declared configurability and derive an ordered active `Catalog`
from a validated explicit allowlist before help normalization or command
matching. Keep full-catalog validation as the release contract and add a
view-specific closure check. Compose a domain profile, application port/service,
and platform file adapter in production only. Expose read-only `config show`
and fixed-target write `config edit`; the latter uses a bounded line selector
and crosses `execution.Invoker` only after `save`.

## Alternatives considered

### Remove disabled commands from `DefaultCatalog`

Rejected because contract lint, capabilities, API coverage, and future
re-enabling would lose their single source of truth.

### Persist disabled paths

Rejected because newly released commands would silently become visible. An
enabled allowlist keeps a curated profile stable across upgrades.

### One exact interactive `config` command

Rejected because it leaves no truthful read-only configuration reconciliation
task and conflicts with the established namespace navigation model. The
`config` namespace keeps the root compact while `show` and `edit` separate
effects.

### Treat selection as authorization

Rejected. The same local principal can edit or delete the preference, and
provider authentication and mutation policy remain the actual controls.

### Raw-terminal checkbox TUI

Rejected for portability, cancellation, dependency, and non-TTY agent costs.
The line protocol provides deterministic selection without terminal takeover.

## Layer changes

- Domain: exact-path selection profile and bounded validation.
- Application: load/save port, profile service, and explicit-save local policy.
- Infrastructure: strict platform file resolution and decode/encode;
  same-directory replacement with rename/open-directory sync on Unix and an
  explicit portable Windows limitation.
- CLI: catalog configurability/view validation, production composition,
  pre-dispatch loading, `config show`, selector rendering, and stable faults.
- Harness/docs: capability ledger, thesis consequence, product/architecture/
  security contracts, readiness scenario, README, and skill guidance.

## Verification

- Domain/app/infra unit tests for validation, load states, policy, strict JSON,
  bounds, modes, symlinks/special files, platform replacement, and
  phase-sensitive cancellation.
- CLI contract tests for selection protocol, repair, dependency rejection,
  same-instance re-enable, all help forms, workflows/recoveries, and output.
- Negative I/O tests proving disabled commands do not resolve PAT or call the
  provider and enabling does not bypass authentication or confirmation.
- Cross-build Windows resolver coverage and Unix XDG/macOS semantics.
- Agent-readiness replay for hiding contact-request management and later
  restoring it without guessing paths from normal help.
- Required completion gate: `task check`.

## Commit slices

1. Domain/application/infrastructure persistence with tests.
2. Catalog view plus config commands and CLI contract tests.
3. Governing documentation, readiness evidence, and closure.
