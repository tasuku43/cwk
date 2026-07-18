# Project Theses

This is the first document to consult when a design choice is ambiguous. It states why Agentic CLI Foundry exists and the principles from which its product, architecture, security, and harness decisions follow.

A derived project must make these theses concrete. Renaming `agentic-cli-foundry` is not enough. Replace the generic users, outcomes, measures, examples, non-goals, and enforcement references with facts about the new tool. Preserve a template thesis only when it is genuinely true for that product.

Each thesis follows one causal chain:

```text
North Star or thesis
  -> consequences for product and engineering choices
  -> mechanical enforcement that detects regressions
```

If a statement has no observable consequence, it is not yet useful. If an important consequence has no enforcement, it remains an aspiration and must be labeled as such.

## Thesis lifecycle

The first theses are seeds: the smallest hypotheses needed to choose and review a minimal end-to-end slice. They should be decisive, but they are not assumed to be complete.

Use this continuous loop:

```text
seed a north star and minimal theses
  -> build the smallest vertical slice that can challenge them
  -> record repeated decisions, user outcomes, agent confusion, and friction as evidence
  -> revise the thesis before adding a code workaround or exception
  -> propagate consequences into product, architecture, security, Skills, catalog, and harness
  -> repeat with the next slice
```

Early in a project, thesis revisions should be frequent because every real slice reveals missing vocabulary and false assumptions. As the project matures, revisions should become less frequent, not forbidden. User behavior, incidents, compatibility pressure, and maintenance evidence remain valid reasons to change them.

Record evidence in the active work packet. A thesis revision is complete only when:

- the new statement explains the evidence;
- consequences and non-goals are explicit;
- affected durable documents and Skills agree;
- mechanical enforcement detects the old failure or workaround;
- compatibility and migration impact are reviewed.

Do not keep a thesis unchanged merely because code already exists. Do not change a mature thesis merely because one implementation would be easier without it.

## North Star

**A contributor or coding agent can turn a well-defined CLI idea into a small, safe, public-ready vertical slice without guessing the product vocabulary, architectural boundaries, side-effect policy, or completion gate.**

The template's success is measured by whether a new maintainer can answer, from repository evidence:

- Who is the tool for, and what outcome does it own?
- Which public commands exist, and how are they discovered?
- Where may domain, application, infrastructure, and CLI code depend?
- What can each operation affect, and where is that checked?
- Which command proves a change is complete?
- What must be reviewed before source or artifacts become public?

The template does not measure success by the number of included frameworks, commands, or integrations. A small coherent vertical slice is more valuable than a broad collection of optional mechanisms.

### Consequences

- The repository is runnable before customization.
- Documentation is part of the scaffold, not an afterthought.
- The default capability crosses every architectural layer and has contract tests.
- Product-specific integrations are omitted until a derived project can state their purpose and trust boundary.
- One catalog and one gate minimize competing sources of truth.

### Mechanical enforcement

- `go run ./cmd/agentic-cli-foundry --help`, `doctor`, `sample list`, and `sample read --id` exercise the default utility and discover/act slices.
- `cli.Catalog` contract tests keep public discovery and routing aligned.
- `tools/archlint` checks layer boundaries.
- `./scripts/check.sh full` is the canonical completion path.
- `./scripts/check.sh public` checks identity, licensing, and public-boundary policy.

## Thesis 1: Define the user outcome before the mechanism

A CLI command exists to deliver a user outcome, not to mirror a package, protocol, SDK, or vendor API.

### Consequences

- Command names use the user's task vocabulary.
- A single task may compose several adapters.
- A vendor method may remain internal even when an adapter exists.
- New transport flexibility is not accepted as a substitute for a missing product decision.
- Non-goals are recorded so agents do not “complete” the tool by exposing every available method.

### Mechanical enforcement

- Every `cli.CommandSpec` must name a documented public task.
- Catalog tests reject duplicate or undiscoverable command paths.
- Application use cases own orchestration; infrastructure adapters cannot register public commands.
- Work packets require a user outcome and non-goals before implementation tasks.

### Derived-project questions

- What sentence would a user say before reaching for this tool?
- Which outcome does the tool own from start to finish?
- Which vendor concepts must remain implementation details?
- Which superficially related tasks are deliberately unsupported?

## Thesis 2: Make discovery, execution, and interpretation predictable

Humans and agents should be able to discover a command, invoke it, and interpret its result without exploratory network calls or undocumented heuristics.

### Consequences

- Root help is a compact index; command help contains the exact usage and effect.
- Root agent help exposes only outcome-selection facts and a machine-readable scoped-help request; exact-command and namespace help expose invocation, output, authentication, failure, mutation, and workflow details.
- Help and dispatch derive from the same static catalog.
- Output shape, exit behavior, and error ownership are deliberate public contracts.
- Deterministic multi-step behavior belongs in an application use case rather than an agent prompt.

### Mechanical enforcement

- Catalog-wide help and routing contract tests run without external I/O.
- Agent-help shape and growth tests reject detailed contracts leaking back into the root index and prove an unknown outcome reaches scoped detail in at most two discovery invocations.
- Executable JSON-output contract tests compare renderer schema versions, envelopes, and item fields with the catalog declarations.
- Tests cover stable command paths, effects, examples, and negative input behavior.
- Use-case tests fix orchestration order and ambiguity handling.
- Public-contract changes are called out explicitly in pull requests.

### Derived-project questions

- What is the cheapest reliable path from root help to a successful command?
- Which output fields and exit statuses are stable?
- Which deterministic workflow should be one command rather than several agent steps?
- How does a user obtain the unique identifier required by an action?

## Thesis 3: Separate discovery from action and pass opaque IDs unchanged

Discovery owns ambiguity. Action owns one uniquely identified target. The opaque identifier emitted by a discovery command is accepted unchanged by the corresponding action command.

### Consequences

- Every public command has a `CommandRole`: `RoleUtility`, `RoleDiscover`, or `RoleAct`; `RoleUnknown` is invalid.
- A `discover` command may accept filters, return zero or more candidates, and emit stable opaque IDs.
- An `act` command requires at least one declared opaque reference and never chooses among candidates.
- An action does not search again, choose the “best” candidate, accept a copied resource URL as an implicit alternative, or reconstruct an identifier from display fields.
- The ID is not decoded, normalized, case-folded, unescaped, or reformatted between producer and consumer unless its domain type explicitly defines that transformation.
- Display labels may change without changing the reference contract.

### Mechanical enforcement

- `cli.CommandSpec` declares `Role`; reference kinds are attached once to structured input and output fields in its `AgentContract`.
- The catalog derives `ProducedRef{Kind, Field}` and `ConsumedRef{Kind, Argument}` projections from those fields, so routing, help, and reference-flow checks cannot drift across parallel registries.
- Catalog validation rejects an incomplete role/reference declaration.
- Agent help projects role and reference flow from the same catalog used by dispatch.
- Whole-catalog tests prove every consumed reference has a visible producer, every produced reference has a consumer, and no required-reference cycle is closed off from an invocable producer.
- Round-trip tests pass the exact opaque ID bytes emitted by discovery into the action command.
- Negative tests reject URLs, resource paths, control characters, and undocumented alternative reference forms before adapter execution.

The runnable proof is `sample list` -> `sample read --id`. It uses reference kind `sample`, producer field `id`, and consumer argument `--id`. The synthetic ID is `smp_` followed by exactly twelve lowercase hexadecimal characters. Validation rejects uppercase, partial IDs, names, URLs, whitespace, and resource paths without rewriting them.

### Derived-project questions

- Which command owns ambiguity and returns candidates?
- What opaque reference kind connects discovery to action?
- Is the action target truly unique, and where is that proven?
- Which tempting identifier conversions would couple the CLI to an external storage or URL format?
- If no in-tool producer exists, what product and catalog change is needed before exposing the action?

## Thesis 4: Declare side effects before executing them

An operation's effect, intent, and target are product facts. They must be known and validated before infrastructure performs the operation.

### Consequences

- `read`, `create`, and `write` are explicit domain values, not guesses derived from an HTTP verb or function name.
- Mutations carry an `operation.Intent` and `operation.TargetRef`.
- The public mutation contract binds declared CLI inputs to target roles: `create` consumes one opaque parent/scope reference and no pre-existing target ID; `write` consumes an opaque ID for the existing target and may also consume a distinct opaque parent/scope reference.
- `target_inputs` is the complete set of role-bound target inputs, not an unclassified list that can contain extra selectors.
- Unknown or inconsistent effects fail closed.
- Authentication, confirmation, audit, dry-run, and policy decisions can attach to one execution boundary.
- Adapters receive bounded inputs rather than unrestricted clients or executors.

### Mechanical enforcement

- Domain constructors and validation reject unknown or incomplete mutation intent.
- The catalog requires a declared effect for every public command.
- Catalog validation rejects a read with mutation metadata, a create without exactly one required CLI `parent_input` or with a `target_id_input`, and a write whose required CLI `target_id_input` is absent, unbound, non-opaque, or a different reference kind from `TargetKind`. It also rejects duplicate or extra target inputs and an invalid optional parent role.
- Architecture lint prevents application code from importing concrete infrastructure.
- Negative tests prove that validation failure occurs before the side effect.

### Derived-project questions

- What assets can each command read, create, change, delete, notify, or publish?
- Which exact opaque input supplies a create's parent or a write's existing target, and does its reference kind match the declared role?
- Is one target reference sufficient, or does the product need a typed multi-target impact model?
- Which effects require human authorization or a dry-run preview?
- What evidence proves that rejection happens before external I/O?

## Thesis 5: Turn important claims into executable contracts

Documentation explains a claim, but a repeatable check preserves it across contributors and agents.

### Consequences

- Architecture, security, compatibility, generated-code, and release promises identify their checks.
- A new invariant is incomplete until its failure mode is tested.
- CI invokes the repository's scripts instead of recreating policy in workflow YAML.
- Generated updates fail visibly when they introduce an unclassified change.
- Exceptions include a reason and a regression test.

### Mechanical enforcement

- `./scripts/check.sh` owns the `fast`, `full`, `security`, `release`, and `public` profiles.
- Task aliases, local hooks, and CI delegate to that script.
- Tool versions and third-party actions are pinned according to repository policy.
- `task check` is the pre-merge gate; higher-risk operations add their named profile.

### Derived-project questions

- Which current claims rely only on reviewer memory?
- What is the smallest mutation that would violate each invariant?
- Can the check produce an actionable failure message?
- Is the same implementation exercised locally and in CI?

## Thesis 6: Treat public safety as a design boundary

Once source or history reaches a public remote, confidentiality cannot be restored by deleting a later commit. Public readiness begins at repository creation.

### Consequences

- Derived repositories start with clean history rather than copying a private `.git` directory.
- Runnable public defaults replace organization-specific placeholders.
- Fixtures use synthetic identities and data.
- License, disclosure channel, dependency rights, and release behavior are decided before publication.
- Private URLs, organization names, credentials, and internal operating procedures are prohibited in all tracked and generated content.

### Mechanical enforcement

- `.harness/project.json` records identity and public-boundary policy.
- `tools/repoguard` checks forbidden identifiers, secrets, placeholders, required community files, and repository readiness.
- `task security` scans source and configuration.
- `task public:check` is required before the first public push and public release.

### Derived-project questions

- Was any file or Git object copied from a private source?
- Who owns the code and documentation, and which license applies?
- Which identifiers or domains must never appear publicly?
- What private vulnerability-reporting channel exists?

## Thesis 7: Keep one maintainable path through the repository

The project should not depend on one maintainer remembering parallel registries, duplicated policy, or undocumented release steps.

### Consequences

- `AGENTS.md` is the only agent-policy source of truth.
- `cli.Catalog` is canonical; help and dispatch do not maintain separate lists.
- `scripts/check.sh` is canonical; Task, hooks, and CI do not duplicate commands.
- Durable decisions live in theses, numbered docs, or ADRs; active implementation state lives in work packets.
- Dependencies are added only when their safety and maintenance value exceeds their ongoing cost.

### Mechanical enforcement

- Contract tests compare every derived view with its source of truth.
- Repository guard checks required documentation and bootstrap state.
- Documentation links and command snippets are checked where practical.
- Completion requires the same full gate regardless of whether a human or agent made the change.

### Derived-project questions

- Where are the current duplicate sources of truth?
- Which recurring judgment needs a Skill, generator, or lint?
- What can a new maintainer safely remove?
- Which maintenance task still requires private knowledge?

## Mature thesis changes

A mature thesis is allowed to change, but not as an incidental implementation edit. Propose the new statement, present user, incident, compatibility, or maintenance evidence, identify the consequences, update the enforcement path, and record migration impact in an ADR. The burden is evidence and repository consistency, not age alone.
