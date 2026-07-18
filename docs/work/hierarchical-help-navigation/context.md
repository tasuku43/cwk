# Work Context: Hierarchical human help navigation

## Pre-change behavior

- `go run ./cmd/cwk --help` prints 36 exact command entries under one
  `Commands:` heading, including all six `rooms` paths.
- `go run ./cmd/cwk help rooms` succeeds and selects the six `rooms` commands,
  proving that word-boundary namespace selection already exists.
- `go run ./cmd/cwk rooms --help` fails with `unknown_command` because routing
  attempts to match a catalog command before interpreting a namespace help
  suffix.
- `go run ./cmd/cwk rooms list --help` succeeds because exact command routing
  has a special one-argument help case.
- `go run ./cmd/cwk help --format agent` emits schema-v3 JSON with one compact
  outcome entry per exact command and a selector for exact path or namespace.

## Implemented observation

- Root text help is 836 bytes, down from the observed 2,553-byte flat root
  (1,717 bytes, or 67.3%, removed) while retaining all 13 direct or namespace
  navigation entries.
- `cwk rooms --help` and `cwk help rooms` are byte-identical 599-byte views
  with six relative commands, their exact summaries, usage, and forward/back
  navigation.
- `cwk rooms list --help` and `cwk help rooms list` are byte-identical and link
  back to the `rooms` namespace plus the exact machine contract.
- Unknown `missing --help` retains `unknown_command`, exit 2, and the root
  `help` next action.
- Schema-v3 agent root, `rooms` namespace, and `rooms list` exact output sizes
  remain 9,568, 185,329, and 22,505 bytes respectively, matching the
  pre-change observations.
- `messages list --help` is a 1,509-byte human view that now exposes repeatable
  sender OR selection, its 100-reference bound, and one-hop reply context from
  catalog input metadata; basic invocation discovery no longer requires opening
  the 53,078-byte complete machine contract.
- A post-implementation review measured the same views with `o200k_base`: root
  text 200 tokens, `rooms` namespace text 144, `rooms list` exact text 91,
  `messages list` exact text 323, agent root 1,987, `rooms list` exact agent
  5,575, `rooms` namespace agent 49,777, and `messages list` exact agent 14,268.
  The exact human message view deliberately spends 136 more tokens than its
  pre-input-projection form so repeatability and context semantics do not
  require the 14,268-token complete machine contract.
- The final independent architecture, semantics, and UX reviews found no
  blocking issue after nested-namespace, hostile-input, exact-input, and lazy
  authentication coverage was added.
- The Go 1.26.5 full repository gate passes, including hygiene, architecture,
  contract, unit, security, vulnerability, release, public-boundary, and
  reproducibility checks.

## Relevant structure

- Entry point: `internal/cli/cli.go` parses global options and dispatches the
  longest exact catalog path.
- Domain rule: command path syntax and operation effects are unchanged.
- Application use case: none; help is a local CLI utility.
- Infrastructure boundary: none; `New` initializes Chatwork lazily and help
  must not resolve `CWK_API_TOKEN` or call the provider.
- CLI catalog or presentation: `internal/cli/help.go` owns text and agent help;
  `Catalog.Select` already selects exact paths and word-boundary namespaces.
- Existing tests and harness checks: `internal/cli/help_test.go` fixes root,
  namespace, exact, schema, and size behavior; `internal/cli/chatwork_pat_test.go`
  proves production help does not resolve the PAT.

## Constraints

- `cli.Catalog` remains the only public-command source of truth.
- A namespace is a selector and navigation node, not an executable command.
- Root renders direct commands before namespaces and preserves curated catalog
  order within each section; namespace command order follows the catalog.
- Human text help may be hierarchical, but machine root help must retain exact
  outcomes so an unknown outcome still needs at most root plus one scoped
  request and a known path needs one scoped request.
- No new dependency, credential source, destination, external I/O, or mutable
  state is introduced.
- Exact input projection makes every input name structural terminal text, so
  catalog validation now rejects invalid UTF-8, Unicode whitespace,
  control/format characters, and line separators for non-argv sources as well
  as flags and positional arguments.

## External facts

None. The behavior and decision are wholly repository-local.

## Unknowns

- [x] Whether namespace selection already exists: verified through
  `Catalog.Select` and `cwk help rooms`.
- [x] Whether agent root help should also be grouped: no; its exact outcome
  entries are the routing information that satisfies the existing bounded
  machine-discovery contract.
- [x] Whether namespace descriptions need a new registry: no; canonical names
  plus catalog-derived command counts provide topology without duplicated
  semantic metadata.
- [x] Whether exact text help needs separate flag prose: no; the existing typed
  `AgentContract.Inputs` supplies the concise human input projection.

## Thesis evidence

- Repeated design decision or point of agent confusion: a flat exhaustive root
  list duplicates namespace prefixes and obscures the next narrowing action.
- User outcome or friction observed in the minimal slice: the user expected
  `rooms --help`, but it failed while root presented every `rooms` leaf.
- Code workaround or exception being considered: hard-coded namespace lists or
  summaries were rejected because they would compete with `cli.Catalog`.
- Current thesis that resolves it: Axiom 2 requires bounded, non-guessing task
  discovery; text and machine surfaces may use different projections when
  both remain bounded and deterministic.
- Downstream impact: thesis, product, architecture, harness wording, README,
  routing, and CLI tests must agree. Security and external API contracts do not
  change.

## Reproduction or observation

```sh
go run ./cmd/cwk --help
go run ./cmd/cwk rooms --help
go run ./cmd/cwk help rooms
go run ./cmd/cwk rooms list --help
go run ./cmd/cwk help --format agent
```

Observed on macOS with Go 1.26.5. The commands use no credentials or live data.

## Security and public-boundary notes

- Assets and side effects involved: stdout/stderr only; all help routes are
  `EffectRead` and local.
- Credentials or confidential data involved: none; PAT resolution must remain
  lazy and uncalled.
- Structural input-name boundary: hostile non-argv names fail catalog
  validation before help rendering; tests cover escape, format, separator,
  whitespace, and invalid UTF-8 values.
- New dependencies, destinations, files, processes, or generated content:
  none.
- External schema provenance, publication rights, and drift evidence: not
  applicable.
- Pagination, timeout, retry, idempotency, and cancellation facts: no provider
  operation occurs; the existing context and complete-write behavior remains.
- Publication and licensing concerns: none.

## Glossary

- **root text help**: human output from `cwk --help` or `cwk help`.
- **namespace**: a canonical word-boundary prefix such as `rooms` that groups
  exact catalog commands but is not itself executable.
- **exact command help**: the detailed human view for one runnable catalog
  path, such as `rooms list`.
- **agent root index**: schema-v3 machine JSON containing exact outcome entries;
  it is intentionally distinct from the human namespace index.
