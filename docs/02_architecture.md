# Architecture

Chatwork CLI uses four layers, a task-oriented command catalog, typed operation intent, and one composition root. The purpose is to keep product decisions separate from external-system details while giving side effects a narrow, testable path.

## Dependency direction

```text
internal/cli  ------> internal/app
      |                    |
      |                    v
      +------------> internal/domain <------ internal/infra

internal/domain does not depend on app, infra, or cli.
internal/app does not depend on infra or cli.
internal/infra does not depend on app or cli.
```

`tools/archlint` enforces this direction for production code. Tests may use dedicated helpers, but they must not create a production bypass.

## Layer responsibilities

### Domain

`internal/domain/` contains pure project vocabulary, values, validation, and invariants. It performs no network, filesystem, process, terminal, clock, or credential I/O.

The base template includes:

- `operation.Effect` with `EffectUnknown`, `EffectRead`, `EffectCreate`, and `EffectWrite`;
- `operation.TargetRef`, a stable description of the object or scope affected;
- `operation.Impact`, explicit cardinality, notification, access-change, and destructive declarations;
- `operation.Intent`, the declared effect, target, and impact presented to the execution boundary;
- `fault.Error`, stable failure kind, code, retryability, and next-action metadata;
- `page` and `apicall` envelopes for opaque cursors and finite call policy;
- secret-free authentication requirement and session metadata, including a private, non-serialized ephemeral binding passed to authenticated task ports;
- `doctor` domain values used by the default task.

Unknown effects and incomplete mutation intent are invalid domain states at an external-execution boundary.

### Application

`internal/app/` contains user-task use cases. A use case:

- interprets task-specific input;
- chooses the order of operations;
- handles deterministic composition and ambiguity rules;
- defines the smallest ports it needs;
- returns task-specific results rather than infrastructure response types.

The application layer depends on domain values and primitive types. It does not import infrastructure, parse CLI arguments, render terminal output, or construct transport requests.

`internal/app/doctorcmd` owns the local utility, while `internal/app/chatworkcmd`
owns the public provider tasks. `internal/app/samplecmd` remains only as an
explicitly constructed synthetic test fixture. Reusable application boundaries
include authentication gating, complete-or-no-result pagination, and
policy-neutral mutation invocation.

### Infrastructure

`internal/infra/` implements application or domain-facing contracts for external systems. Examples in a derived project may include HTTP adapters, filesystems, credential stores, clocks, subprocesses, or platform services.

Infrastructure owns protocol-specific validation and conversion. The Chatwork
API token and authorization header never leave this layer. Infrastructure does
not decide which public command should exist, how several adapters form a user
task, or how terminal output is presented.

The Chatwork API adapter is the sole production authentication adapter. It
reads `CWK_API_TOKEN` once from the command environment, retains the token in a
private process-local record, returns only secret-free authentication metadata
across its domain-facing boundary, and resolves that exact record immediately
before provider I/O. No production package owns OAuth protocol, browser,
configuration, or credential-store behavior. [ADR 0003](decisions/0003-chatwork-pat-only.md)
pins this reduced boundary.

`internal/infra/systemdoctor` is the diagnostic adapter. `chatworkapi` owns the
production provider boundary. `internal/infra/sampledata` is a deterministic
offline repository retained only for generic contract tests.

### CLI

`internal/cli/` owns:

- `CommandSpec` and `Catalog`;
- `CommandRole` and the structured `AgentContract` for capability, inputs, outputs, complete/paged cursor binding, prerequisites, failures, authentication, and mutation facts;
- argument parsing and task-level validation;
- help and public discovery;
- output and error presentation;
- the composition root that wires use cases to concrete adapters;
- the controlled handoff to side-effect execution.

For Chatwork output, including relationship-aware message results and the current headerless task projection, the layers divide responsibility further:

- Domain defines provider-neutral message, participant, recipient, reply, quote, context-coverage, and unresolved-reference values. It rejects impossible or internally inconsistent graphs but performs no parsing or rendering.
- Infrastructure decodes Chatwork wire DTOs and parses provider-specific message notation into typed facts. It preserves external text as untrusted data and never invents a reply from To, display names, prose, or temporal proximity.
- Application use cases select the bounded data required by one outcome, resolve only explicit relationships available within that bound, and return a typed task result with coverage and unresolved facts.
- CLI presentation projects that same typed result through a release-versioned text contract. The current headerless task projection starts with the result noun and emits only catalog-declared task facts, exact canonical references, task-relevant bounds/completeness/uncertainty, and trust framing for external text. For `messages list`, presentation assigns first-sender-order actor aliases, builds a canonical-message-to-provider-sequence index, and emits one flat record per input item in unchanged order. One fixed schema assigns the positional sequence, canonical message reference, actor, send time, and terminal-safe quoted body; only optional typed edges retain per-record labels. It never traverses or infers a thread: typed resolved replies become `reply=#N`, unresolved targets remain explicit, and aliases remain document-local. It adds no global version/task preamble, standalone provider coverage record, raw Chatwork notation as semantic structure, provider/wire extras, empty optional shells, or non-contract defaults. Presentation does not define relationship truth, completeness, identity, or task policy. Future candidate renderers must consume the same boundary.

`cmd/cwk/main.go` is a thin executable entry point. It should not contain product logic or construct adapters independently of the CLI composition root.

Production Go packages stay within `cmd/` and the four `internal/` layers. The `cmd/` entrypoint imports only context, operating-system signal handling, and `internal/cli`; process execution, network, filesystem, and third-party dependencies belong behind infrastructure ports. Repository-only programs live under `tools/` and cannot be imported by production packages.

CLI, application, and infrastructure code propagate the caller context instead of creating `context.Background()` or `context.TODO()`. The command entrypoint creates the signal-aware root and calls the context-only CLI boundary; a nil context produces a context-independent contract fault before dispatch. Infrastructure network code uses an explicitly constructed client governed by a finite call policy; package-level default HTTP clients and convenience calls are rejected by architecture lint.

## Chatwork authentication topology

Chatwork has one credential source behind the secret-free gate.

```text
command-process CWK_API_TOKEN
  -> CLI composition constructs the PAT infrastructure adapter
  -> application authentication gate receives a PAT-only secret-free requirement
  -> infrastructure issues one process-local binding for that token record
  -> application passes the binding unchanged to the Chatwork task port
  -> infrastructure resolves the exact record immediately before I/O
  -> x-chatworktoken on the fixed Chatwork API request
```

There is no authentication-method selection, probing, or fallback. A missing or
invalid `CWK_API_TOKEN` fails during composition before a provider task request.
The token determines the one account used by that command process and remains
private to infrastructure. `cwk` does not accept it in argv, persist it, or
expose login, status, logout, callback, or profile commands.

## Catalog as the public source of truth

`cli.Catalog` contains every public `cli.CommandSpec`. Routing, root help, command help, uniqueness checks, and catalog-wide effect tests derive from it.

Do not add a second command list for documentation or dispatch. Internal adapters and generated operations are not public merely because they exist. A capability becomes public only when a user-task use case and command specification deliberately expose it.

At minimum, catalog validation rejects:

- duplicate or empty command paths, and a command path that is also another command's word-boundary namespace prefix;
- missing summaries or handlers;
- `EffectUnknown`;
- `RoleUnknown`;
- malformed, duplicate, role-inconsistent, orphaned, or closed-cycle reference declarations;
- missing capability, input descriptions/allowed values, output field meanings/completeness, prerequisites, or recovery commands;
- argv input metadata whose `Required` value or ordered `AllowedValues` disagree with the small bracket/`a|b` usage grammar; non-argv sources are excluded from this syntax check;
- missing or inconsistent common runtime failures (`operation_canceled`, output `output_write_failed`, standard authentication-gate failures, and standard mutation-invoker contract/policy/unknown-outcome failures), or an unknown-outcome recovery that points to another mutation;
- recovery commands that are only catalog prefixes, contain unchecked argv, use an unknown help selector, or otherwise fall outside the exact-path/`help <path-or-namespace>` grammar;
- read commands with mutation metadata; reference-bound creates without exactly
  one required CLI parent binding; reference-bound writes without a matching
  required CLI existing-target binding; fixed-target mutations with inputs or
  a mismatched target; and mutations without complete impact;
- a root agent-index entry larger than 512 encoded bytes, which would let selection prose crowd detailed scoped contracts as the catalog grows;
- command metadata that cannot produce consistent help and routing.

## Command roles and opaque reference flow

`CommandRole` describes where a task sits in a user workflow:

| Role | Responsibility |
|---|---|
| `RoleUtility` | Repository or runtime operation that is neither candidate discovery nor unique-target action |
| `RoleDiscover` | Owns ambiguity and may emit opaque references with candidates |
| `RoleAct` | Operates on required declared opaque references, or on one exact catalog-declared fixed `tool_local` singleton when no selection exists |
| `RoleUnknown` | Invalid for a public command |

Reference kinds live on `AgentContract.Output.Fields`, `AgentContract.Inputs`, and the top-level cursor field owned by `AgentContract.Pagination`. `ProducedRef` and `ConsumedRef` are compatibility projections, not a second declaration. A fixed target instead declares a stable kind, ID, description, and `tool_local` scope directly on the agent contract; it cannot coexist with produced or consumed target references. The catalog derives the reference graph, scoped workflows, next actions, and fixed-target facts from those fields. A shared reference kind is an explicit claim that every value of that kind is interchangeable at each matching input; use distinct kinds when fields or target roles are not interchangeable. Every reference kind needs a producer and consumer, and the required-reference dependency graph must be reachable from at least one command that can run without an unresolved required reference. Optional first-page cursors do not create a dependency. `CommandOutput.Fields` always describe values inside the declared JSON envelope; a public `paged` output owns its separate cursor name, string type, description, and shared opaque kind in `Pagination`. Its typed `completion: "empty_cursor"` rule makes an always-present empty string the sole completion marker. Human help may stay concise; scoped agent help exposes the exact graph, fixed target, pagination binding, and field meanings.

Agent-help schema version 3 separates selection from invocation detail. Root `help --format agent` is an `index` view containing only each command's path, top-level namespace, summary, capability ID, outcome, effect, and role. Each encoded command entry has a 512-byte catalog budget. Its `scope_request` names `commands[].path` and `commands[].namespace` as selectors and supplies the exact invocation template. `help <selector> --format agent` is a `scope` view containing global I/O/error contracts, complete selected `AgentContract` values, and reference workflows touching the selection. Its I/O contract marks external text as untrusted data, declares visible structural projection, and distinguishes validated exact opaque references. A known path therefore needs one help invocation; an unknown outcome needs the root index and one scoped invocation. Root size still grows with the number of outcomes, but detailed inputs, outputs, authentication, failures, mutations, and workflows do not multiply there.

Catalog `CommandOutput` metadata is executable compatibility data, not descriptive decoration. Generic CLI contract tests run each built-in JSON renderer and compare its `schema_version`, declared envelope, and every item key with the corresponding catalog declaration. Agent-help shape snapshots separately fix the intentionally different root and scoped views.

## Semantic and presentation boundary

Every presentation candidate consumes one typed semantic result; candidates do not parse one another's rendered output.

```text
Chatwork response
  -> infrastructure validates wire bounds and parses explicit notation
  -> domain values represent participants, messages, and typed relations
  -> application selects one bounded outcome context and declares coverage
  -> interchangeable CLI presentation candidates render the same facts
  -> evaluation compares agent understanding and resource cost
```

Relationship truth has three states:

| State | Meaning | Presentation requirement |
|---|---|---|
| explicit and resolved | Provider notation identifies a relation and the referenced object is available | Preserve it so the evaluation answer remains recoverable |
| explicit and unresolved | Provider notation identifies a relation but the referenced object is outside the bound or unavailable | Preserve the relation and unresolved/coverage state without fabricating the object |
| absent or unsupported | No provider fact establishes the relation | Do not imply that the relation exists |

Filtering and context selection are application outcome concerns. Provider pagination and notation parsing remain infrastructure concerns. Presentation owns only representation.

Candidate C (`cwk-context-capsule/1`) is the first stable public presentation baseline and retains that historical evidence. Competition 1 was inconclusive because benchmark defects made its promotion result non-authoritative. A P-derived task projection (`cwk-task-projection/1`) was selected afterward by an explicit owner compatibility decision, then hardened beyond the frozen candidate. The current default further removes its in-band schema/task preamble and standalone provider coverage line through a second explicit pre-1.0 compatibility decision. Neither decision describes P as the benchmark winner. Future alternatives may be developed in isolated worktrees against the same semantic fixtures, answer key, trust rules, canonical references, and output-boundary requirements. Candidate-specific schemas, grammars, ordering, shorthand, or visual hierarchy remain outside domain and application code, and raw experimental evidence remains immutable decision input.

Upstream coverage is separately pinned in `.harness/chatwork_api_v2.json`. That manifest may prove that every fixed official operation has a public task owner, but it cannot dispatch a request or generate a command. `cli.Catalog` remains the only public-command source of truth.

The same manifest pins the first implementation's reviewed resource ceilings and the exact upstream operation IDs in each confirmation class. Production code uses compile-time typed constants with those values; it does not load harness JSON at runtime. `tools/contractlint` detects drift in the independent evidence, while adapter, application, and CLI tests prove enforcement at the transport, list, upload, and rendered-output boundaries. Its current `coverage_status` is `complete`, so every one of the fixed 32 operations must retain at least one public capability owner.

The representative public graph is:

```text
rooms list
  RoleDiscover
  produces {kind: chatwork-room, field: room_ref}
       |
       | exact opaque value
       v
messages list --room <room-ref>
  RoleAct
  consumes {kind: chatwork-room, argument: --room}
```

The current task projection emits the exact canonical `room_ref` directly and
defines no display alias; only that canonical value is accepted by the action.
Historical candidate-C aliases were document-local and were never accepted by
commands. The former sample graph is absent from `DefaultCatalog` and remains
an offline test fixture for generic boundary checks.

## Operation effect and intent

Effect answers **what class of action occurs**:

| Effect | Meaning |
|---|---|
| `Read` | Observes state without intentionally changing it |
| `Create` | Creates a new object within a declared parent or scope |
| `Write` | Changes or removes an existing declared target |

Intent answers **what this invocation is allowed to affect**. A mutation intent includes a `TargetRef` and an `Impact`. Cardinality and the notification, access-change, and destructive dimensions must all be explicit; their zero values fail closed. Derived projects extend this base with domain-owned detail such as recipients, visibility changes, publication destinations, or workflow triggers.

`MutationContract` connects the public target declaration to runtime intent. For
a reference-bound `Create`, `parent_input` names the single opaque parent or
scope reference and `target_id_input` is absent because the object does not
exist yet. For a reference-bound `Write`, `target_id_input` names an opaque
reference whose kind equals `TargetKind`; an optional, distinct `parent_input`
can bind additional scope. `target_inputs` contains exactly these named roles.
A command-bound singleton instead requires one matching fixed target, an
explicitly empty `target_inputs`, and no input role; the effect determines
whether it is the create scope or existing write target. A free-form, optional,
non-CLI, duplicated, non-reference, remote-as-fixed, or mismatched binding fails
catalog validation rather than being treated as safe.

The catalog binding and runtime `TargetRef` are complementary checks: the former proves that an agent can supply an unambiguous target from the command contract, while the latter proves that the concrete invocation presented to policy and infrastructure has the declared target and impact. Neither replaces the other.

Do not infer effect from an HTTP method, command name, or adapter function. A `POST` may be a read-like query; a local file write is still a side effect without HTTP. The product declaration is authoritative and must agree with the concrete operation at runtime.

## Execution flow

```text
argv
  -> CLI selects one CommandSpec from Catalog
  -> CLI validates role/reference declarations, parses task input, and chooses presentation
  -> application use case interprets the user outcome
  -> domain validates Effect, Intent, TargetRef, Impact, auth, and API envelopes
  -> controlled execution boundary snapshots intent and applies derived policy
  -> infrastructure adapter performs one bounded logical operation
  -> application returns a task result
  -> CLI renders stable output and exit behavior
```

For mutations, validation failure must occur before the external side effect. `app/execution.Invoker` provides the common ordering and has no permissive default policy. Dry-run, human approval, OS authentication, authorization reuse, confirmation, and audit behavior remain derived policy rather than command-local conventions.

The Chatwork policy implementation derives one of three confirmation requirements from typed impact and the fixed operation contract: exact invocation only, exact `--confirm=access-change`, or exact `--confirm=destructive`. CLI parsing supplies the typed confirmation, application policy compares it with the snapshotted intent, and infrastructure never interprets the flag. Every logical provider operation permits one transport attempt. An unclassified post-action result routes only to a catalog-declared read-only reconciliation task.

## Error ownership

- Domain errors explain invalid values or invariants.
- Application errors explain task failure or ambiguity.
- Infrastructure errors map unstable upstream details into a stable `fault.Error` without leaking secrets.
- CLI maps fault kind, code, retryability, retry-after, and next actions to stable human and machine presentation and exit statuses.

The stable exit mapping is `0` success; `2` invalid input; `3` internal; `4` authentication; `5` permission; `6` not found; `7` ambiguous; `8` rate limited; `9` unavailable; `10` rejected; `11` canceled; `12` unsupported; and `13` contract violation. Success output is written to stdout, while text or schema-versioned JSON failures are written to stderr. A zero status requires a complete successful write. A failing `doctor` report may be rendered in full before its structured `rejected` failure so callers receive evidence without treating an incomplete result as success.

Do not render inside domain, application, or infrastructure packages. Do not make a use case parse human-facing error strings from an adapter.

## Adding a vertical slice

Add capabilities in this order:

1. User outcome and product-contract decision.
2. Utility, discover, or act role and any opaque reference flow.
3. Domain vocabulary, effect, intent, reference validation, and invariants.
4. Application input, result, use case, and owned ports.
5. Infrastructure adapter satisfying those ports.
6. CLI arguments, rendering, and one `CommandSpec`.
7. Unit tests at each layer and catalog/reference/execution contract tests.
8. Harness and documentation updates for any new invariant.

This order makes product intent reviewable before transport code creates momentum toward the wrong public API.

## Optional capabilities

Concrete schema generators, update checks, credential stores, OAuth libraries, telemetry, package-manager formulas, code signing, and platform-specific authorization are optional modules, not base-layer responsibilities. A derived project adds one only after documenting:

- the user outcome it enables;
- its trust and compatibility boundaries;
- dependency and maintenance cost;
- failure behavior;
- mechanical checks and release impact.
