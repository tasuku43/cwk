# Chatwork CLI Design

- Status: Accepted and integrated
- Date: 2026-07-18
- Scope: Public Go CLI repository template

Historical note: this design records the original public template scaffold.
In the derived `cwk` product, the sample commands were later removed from the
public catalog and retained only as explicitly constructed offline test
fixtures after Chatwork workflows replaced their contract coverage.
Its generic OAuth/PAT discussion is likewise scaffold history, not a claim that
the current product supplies both methods. [ADR 0003](../decisions/0003-chatwork-pat-only.md)
makes `CWK_API_TOKEN` the sole Chatwork credential input and removes OAuth from
the current core.

## Problem

Starting a CLI from an empty repository is deceptively cheap. The expensive decisions appear later: product vocabulary mirrors an external API, architecture exists only in one maintainer's memory, side effects bypass inconsistent checks, agent instructions drift, CI duplicates local commands, and private assumptions reach a repository intended for publication.

An existing production CLI can demonstrate solutions, but copying it wholesale also copies product-specific code, organization identifiers, release assumptions, accumulated complexity, and private history. A useful template must preserve the reasoning and harness pattern without preserving the original product.

## Goals

- Provide a runnable default Go CLI, not invalid placeholder source.
- Make project thesis and product contract the first customization step.
- Demonstrate a utility slice and a discover-to-act four-layer slice.
- Make public commands, inputs, output schemas, prerequisites, failures, effects, intent, targets, and impact finite and inspectable.
- Let an agent discover a scoped task contract and a deterministic next action without reading source or interpreting error prose.
- Supply policy-neutral authentication, pagination, timeout, retry/idempotency, and mutation boundaries for API-backed CLIs.
- Provide one implementation for local, agent, CI, security, release, and public checks.
- Make a clean public boundary explicit before first publication.
- Remain small enough that a derived project can remove examples it does not need.

## Non-goals

- Reproducing a feature-rich production CLI.
- Selecting a concrete OAuth flow, PAT source, provider API, telemetry system, database, or credential store.
- Supporting arbitrary raw API execution.
- Guaranteeing security for integrations not present in the template.
- Automatically deciding legal rights, product thesis, or release support.

## Considered approaches

### Copy a mature CLI and replace names

This maximizes immediately available code and tests, but also imports product-specific architecture, private identifiers, history, platform assumptions, and a large deletion burden. Agents would struggle to distinguish deliberate generic policy from accidental inherited behavior. This approach was rejected.

### Provide only a minimal Go directory skeleton

This is easy to understand and safe to publish, but leaves the difficult decisions unresolved. Each derived project would reinvent agent policy, side-effect declarations, catalog consistency, checks, public review, and release structure. This approach was rejected as too weak.

### Provide a runnable thesis-and-harness template

This approach keeps a minimal vertical slice while including the durable reasoning, typed boundaries, catalog contract, machine-readable project policy, and public-release gates. It costs more than a bare skeleton but directly addresses the recurring startup risks. This approach was selected.

## Repository identity and bootstrap

The repository uses valid public defaults:

- module `github.com/tasuku43/cwk`;
- binary `cwk`;
- display name `Chatwork CLI`.

`.harness/project.json` stores validated derived identity and policy. `tools/bootstrap` previews and then performs exact replacement of those defaults. Exact runnable values are safer than placeholder syntax embedded in Go identifiers or filesystem paths.

Bootstrap changes its profile from `template` to `ready` only after successful application. That state means identity replacement completed; it does not assert that project-specific theses, security, or release review is complete.

## Documentation design

Numbered documents form a causal sequence:

```text
theses
  -> product contract
  -> architecture
  -> security model
  -> harness
  -> public repository boundary
  -> release model
  -> authentication boundary
  -> external API contracts
  -> agent-readiness scenarios
```

`AGENTS.md` is the only canonical agent and contribution policy. ADRs retain durable trade-offs and supersession history. Work packets separate goal, verified context, chosen plan, and task tracking; lasting conclusions are promoted before a packet closes.

The precedence order is theses, security/architecture invariants, accepted ADRs, work goal/context, plan, then tasks.

Theses follow a learning lifecycle rather than a one-time bootstrap step. The project seeds the minimum thesis, challenges it with a vertical slice, records repeated decisions and friction, revises the thesis before adding workarounds, and propagates the revision through all downstream contracts. Revisions are intentionally frequent early and remain evidence-driven after maturity.

## Code architecture

The default `doctor` command crosses four layers as a utility task. The synthetic `sample list` and `sample read --id` pair crosses the same layers to demonstrate discover-to-act composition.

- `internal/domain/operation` defines `Effect`, `Intent`, `TargetRef`, and explicit base `Impact`.
- `internal/domain/fault` defines stable failure kind, code, retryability, and recovery metadata.
- `internal/domain/authn`, `page`, and `apicall` define secret-free authentication, opaque pagination, timeout, attempt, and idempotency contracts.
- `internal/domain/doctor` defines pure diagnostic values.
- `internal/app/doctorcmd` owns the user-task use case and ports.
- `internal/app/authn`, `pagination`, and `execution` provide fail-closed gates without choosing a product-specific policy.
- `internal/infra/systemdoctor` provides a concrete system adapter.
- `internal/domain/sample`, `internal/app/samplecmd`, and `internal/infra/sampledata` provide the offline discover/act example.
- `internal/cli` owns `CommandSpec`, `Catalog`, routing, help, rendering, and wiring.
- `cmd/cwk` remains a thin executable.

Domain has no outward dependency. Application depends on domain. Infrastructure depends on domain-facing contracts without importing application. CLI is the composition root and may wire all layers.

`cli.Catalog` is the only public-command registry. Help, dispatch, uniqueness, role, reference flow, and effect checks derive from it. Internal adapter existence does not imply public exposure.

The catalog uses `CommandRole` with `RoleUtility`, `RoleDiscover`, and `RoleAct`; `RoleUnknown` is invalid. Structured input and output fields are the only source of reference declarations; produced/consumed compatibility fields, workflows, and next actions are derived from them. The sample list/read slice uses kind `sample`, field `id`, and argument `--id`, proving that discovery owns ambiguity and emits an opaque ID that action accepts without decoding, normalization, URL conversion, or reconstruction. Catalog and CLI tests enforce the producer-to-consumer reference flow.

Each command owns an `AgentContract` with a stable capability ID, outcome, described inputs and allowed values, output formats/fields/types/completeness, prerequisites, optional secret-free authentication requirement, declared failures, and an optional mutation contract. Nil collections mean unknown and fail validation; explicit empty collections mean none. Read commands cannot carry a mutation contract. Create/write commands require target inputs and a complete base impact declaration.

## Public and agent contracts

The default CLI provides human help and `cwk help --format agent`. Schema v3 makes the root response a compact outcome/capability index with a machine-readable scope request. Exact-command and namespace selectors return complete catalog contracts and derived workflows; the scoped command entries retain path, summary, usage, effect, role, produced/consumed references, and structured invocation/recovery detail. This keeps an unknown-outcome journey to two discovery invocations and a known-path journey to one without multiplying full contracts in root help.

Commands separate default human-readable TSV/text from declared JSON machine output. External text remains untrusted data: presentation visibly escapes backslashes, control/format runes, and Unicode line/paragraph separators while preserving printable JSON-looking and prompt-like content as data. This protects output structure but does not claim semantic prompt-injection prevention. Opaque references bypass display projection and preserve the exact validated value. Runtime failures use the same stable taxonomy declared in the catalog and can be rendered as structured JSON on stderr. Exit statuses distinguish input, authentication, permission, missing/ambiguous target, rate limit, temporary failure, rejection, cancellation, unsupported capability, contract failure, and internal failure.

`cwk sample list` emits lowercase `id<TAB>name` or its declared JSON equivalent; `cwk sample read --id <sample-id>` emits `id<TAB>name<TAB>content` or JSON. The ID is canonical `smp_` plus twelve lowercase hexadecimal characters and is passed unchanged. List traversal uses the same opaque-page and complete-or-no-result contract required of a real API adapter.

These contracts are deliberately small but must be tested as public behavior.

## Side-effect design

Every command declares `EffectRead`, `EffectCreate`, or `EffectWrite`; `EffectUnknown` cannot execute. Mutations require typed intent, a target reference, and explicit cardinality/notification/access-change/destructive impact before crossing `app/execution.Invoker`. The invoker snapshots the declaration, checks it against catalog/parser expectations, applies one injected policy, rechecks cancellation, and then invokes one logical action. Missing policy, mismatched target/impact, denial, and cancellation make zero mutation attempts.

Effect is not inferred from transport. The template does not decide whether the injected policy performs human approval, OS authentication, dry-run, authorization, or authorization reuse. Those decisions remain in the derived thesis and security model. Negative tests prove invalid or mismatched mutation state produces zero side-effect attempts.

## External API foundation

The template includes only cross-project invariants:

- OAuth/PAT requirements and session metadata contain no credential-bearing value; an authentication gate proves failure-before-downstream-call ordering.
- OAuth protocol machinery is not reimplemented. A derived OAuth project reviews and adds `golang.org/x/oauth2` behind infrastructure; PAT-only projects do not inherit it.
- Opaque page cursors are forwarded byte-for-byte. Exhaustive traversal has page/item limits, loop detection, cancellation, and no partial success.
- Every adapter call declares finite timeout, total attempt count, and idempotency. Retry of an unsafe operation is invalid; keyed retry reuses one logical-operation key.
- Stable faults hide upstream causes from public output while retaining `errors.Is/As` behavior.
- Wire DTOs remain in infrastructure and publishable schema fixtures are checksum/provenance/license bound.

The template deliberately does not include a generic HTTP client, provider SDK, OAuth login command, credential store, backoff formula, or generated API command surface.

## Fixed and derived boundary

| The template fixes | A derived project must decide | Why the boundary sits here |
|---|---|---|
| Thesis-learning loop and decision precedence | Concrete north star, users, outcomes, measures, and early thesis revisions | The reasoning process repeats; the product answer does not |
| Four layers and import/context/I/O checks | Domain vocabulary, ports, adapters, and optional justified dependencies | Direction prevents recurring coupling without predicting a provider |
| Catalog, agent contract, roles, capability IDs, and opaque-reference graph | Actual task names, filters, references, workflows, and unsupported coverage | Agents need one finite source; API coverage is not a product |
| Stable output/fault/completeness metadata | Stable formats, fields, limits, and product-specific failure codes/actions | Machine composition needs a contract; meanings belong to the task |
| Read/create/write, target, base impact, and one policy-neutral invoker | Approval, dry-run, OS auth, authorization, reuse, and domain-specific impact | Omission must fail closed, but safety policy differs by consequence |
| Secret-free OAuth/PAT gate and reviewed-library rule | Grant/PAT source, storage, refresh, scopes, tenant/account selection, logout/revoke | Credential containment repeats; identity UX and provider behavior do not |
| Opaque pagination and explicit call/idempotency policy | Exhaustive/paged UX, budgets, backoff, vendor error mapping, schema update cadence | Silent truncation/unsafe retry are universal; operational budgets are not |
| One gate, public guard, transactional bootstrap, immutable release machinery | Support matrix changes, signing/provenance, package managers, disclosure contacts | Convergence and non-overwrite are reusable; release promises require ownership |

The detailed API version of this table is maintained in [External API Contracts](../08_external_api_contracts.md). If a derived project needs to cross the boundary, it first revises its thesis/security decision and records the exception in an ADR; it does not add an unreviewed local bypass.

## Harness design

`./scripts/check.sh` implements five profiles:

- `fast` for formatting, architecture, and focused tests;
- `full` as the pre-merge gate;
- `security` for repository, dependency, and security analysis;
- `release` for artifact and Formula contracts;
- `public` for bootstrap and publication readiness.

Task aliases, hooks, and CI delegate to this script. `tools/archlint` enforces the complete production package set, layer/import boundaries, caller-context propagation, and explicit HTTP-client construction. `tools/contractlint` binds catalog capability IDs to the public/internal/deferred/excluded ledger and verifies external-schema fixture digests. `tools/repoguard` enforces repository identity, file shape, likely secrets, and public policy. Claims are added with a failing fixture or test, not by documentation alone.

Bootstrap plans and validates every edit and rename before writing, rejects symbolic links, stages replacements, and rolls back a failed commit. Release creation is immutable: an existing release is not overwritten. The release profile builds and inspects all supported targets, checks checksums, and renders the Formula through the same scripts used by CI.

## Public boundary

The template is authored as public material and uses synthetic data. A derived public repository starts with clean history; it never copies a private `.git` directory. Bootstrap is followed by a complete diff, history, secret, identifier, license, security-contact, fixture, dependency, and release review.

The MIT license applies to the template. A derived project makes an explicit license decision before publication.

## Release design

The default pure-Go matrix covers Linux `amd64`/`arm64`, macOS `amd64`/`arm64`, and Windows `amd64`. A SemVer tag publishes byte-for-byte reproducible archives for identical source, tag, revision, target, and pinned Go toolchain, plus `checksums.txt`. The release harness fingerprints release inputs around two full-matrix passes with separate Go build caches and through the final Formula checks, reports source drift with the changed paths, and requires matching archive digests. Every matrix build and Formula render checks out the exact revision resolved by preflight; the audited stable Formula is then staged on current `main` through a reviewable pull request. Prereleases do not update stable Formula metadata.

The base release does not claim signing, notarization, or external provenance. A derived project must document and enforce those controls if required.

## Implementation sequence

1. Seed and revise the project thesis, then define user outcomes and non-goals.
2. Complete command agent contracts, discover/act reference flow, output/failure compatibility, and security impact before transport implementation.
3. Add pure domain values and application ports/gates.
4. Add one infrastructure adapter with publishable schema fixtures and exact authentication/call policy.
5. Register the user task in the catalog and capability ledger; never generate public commands from API coverage.
6. Add boundary, negative, output, recovery, and scenario tests.
7. Run the canonical full/security/public/release gates and audit the resulting public diff.

## Validation

The initial implementation is accepted when:

- default module and binary build and run;
- human and scoped agent help match the catalog, capability ledger, and structured error contract;
- doctor output and exit contracts pass;
- sample role, TSV/JSON output, exhaustive pagination, reference-graph, exact ID round-trip, and invalid-ID contracts pass;
- architecture lint rejects forbidden/unclassified dependencies, detached application/adapter contexts, and implicit default HTTP clients;
- authentication mismatch, unknown effect, invalid mutation intent/impact, policy denial, and cancellation fail before downstream effects;
- bootstrap dry-run uses the same complete preflight as apply, and bootstrap reaches a validated ready profile without partial application;
- full, security, release, and public profiles pass;
- all documentation links resolve and terminology is consistent;
- no organization-specific or confidential identifier exists in tracked content.
- project-collaboration and team-chat scenarios reach discovery, execution, interpretation, and a structured recovery action within the documented round-trip budget.

## Risks

- **The template becomes a framework.** Keep optional integrations out until a derived product needs them.
- **Documents drift from code.** Bind finite claims to catalog, architecture, repository, and release checks.
- **Agent policy fragments.** Keep `AGENTS.md` as the only policy source of truth.
- **Public check creates false confidence.** Preserve a manual ownership and confidentiality review.
- **Runnable defaults survive into a release.** Ready-profile and public checks reject unfinished identity.
- **Release surface grows faster than support.** Require platform and compatibility decisions before changing the matrix.
