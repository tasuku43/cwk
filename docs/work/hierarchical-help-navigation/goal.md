# Work Goal: Hierarchical human help navigation

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: None

## Outcome

Human-readable help forms a bounded navigation network: root help presents
directly runnable utilities and each canonical top-level task namespace once,
namespace help presents only that namespace's exact commands, and exact command
help remains reachable through the natural trailing `--help` form and presents
the declared inputs needed for human invocation. Every view is derived from the
public catalog.

## Why now

The production root currently prints all 36 exact command paths in one flat
list. The user observed that entries such as all six `rooms` commands belong in
`cwk rooms --help`, and the runtime currently rejects that natural invocation
even though `cwk help rooms` can already select the namespace.

## Non-goals

- Renaming, adding, removing, or reclassifying public task commands.
- Changing task inputs, outputs, effects, authentication, or provider calls.
- Changing the schema-v3 machine-readable root index or non-help task contracts;
  the exact outcome index preserves the two-invocation discovery budget.
- Fuzzy selectors, abbreviated commands, shell completion, or implicit task
  execution from a namespace.
- Adding hand-maintained namespace descriptions or a second command registry.

## Acceptance criteria

- [x] Root text help lists every single-word command and every top-level
  namespace exactly once, preserving catalog-relative order within the direct
  and namespace sections, without listing multiword command paths or their
  summaries.
- [x] `cwk <namespace> --help` and `cwk help <namespace>` return the same
  namespace-scoped text view with relative command names and no commands from
  another namespace.
- [x] `cwk <exact-command> --help` and `cwk help <exact-command>` return the
  same exact command help, including catalog-derived input requirement,
  repeatability, source, values, reference kind, and description facts.
- [x] Every public multiword command is reachable from its root namespace in
  one namespace-help step, with no authentication or provider I/O.
- [x] Hostile input names from every source fail catalog validation before
  exact human help can emit terminal structure.
- [x] Unknown namespaces still fail as invalid input and point to root help.
- [x] The machine-readable root index and non-help task contracts remain
  unchanged; the `help` task's invalid-selector recovery names namespaces.
- [x] Governing documentation and CLI contract tests describe and enforce the
  two complementary discovery networks.
- [x] `task check` passes.

## Governing documents

- Thesis: `docs/00_theses.md`, Axiom 2
- Product contract section: `docs/01_product_contract.md`, Supported-outcome
  promise and Public runnable surface
- Architecture invariant: `docs/02_architecture.md`, Catalog as the public
  source of truth
- Security invariant: help must remain local, read-only, and credential-free
- Existing ADR: None; this is a projection and routing refinement within the
  existing catalog architecture

## Completion definition

The work is complete when both text-help layers and natural `--help` aliases
are catalog-derived, the agent-help compatibility contract is proven
unchanged, documentation and tests agree, `task check` succeeds, and the
reviewed change is committed.
