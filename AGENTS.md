# Agent Contribution Guide

This file is the only canonical operating policy for Codex and contributors in this repository. Do not create a second agent-policy copy.

## Read before changing anything

Read these documents in order:

1. [Project theses](docs/00_theses.md)
2. [Product contract](docs/01_product_contract.md)
3. [Architecture](docs/02_architecture.md)
4. [Security model](docs/03_security_model.md)
5. [Harness](docs/04_harness.md)

For an external API capability, also read [Authentication](docs/07_authentication.md), [External API Contracts](docs/08_external_api_contracts.md), and [Agent Readiness Validation](docs/09_agent_readiness_validation.md).

For release or publication work, also read [Public Repository](docs/05_public_repository.md) and [Release](docs/06_release.md).

## Bootstrap before capability work

When `.harness/project.json` has `profile: template`, use
[`$bootstrap-derived-cli`](.agents/skills/bootstrap-derived-cli/SKILL.md) before
adding a capability. It is the first-run Codex workflow: resolve the derived
identity, preview and apply the repository bootstrap, verify the result, and
make the first project-specific thesis and security decisions. Do not start
`$add-capability` until bootstrap reports `profile: ready` and the generic
product reasoning has been made concrete for the derived tool.

## Decision precedence

When instructions conflict, use this order:

1. Project theses
2. Security and architecture invariants
3. Accepted architecture decision records
4. The active work packet's goal and context
5. Its plan
6. Its task checklist

Do not silently work around a higher-level rule. If a requested change requires a thesis, trust-boundary, or public-contract change, update and review that decision before implementing the mechanism.

## Thesis lifecycle

Theses are working product hypotheses, not frozen slogans. Improve them through this loop:

1. **Seed.** Before broad implementation, write the smallest north star and theses that can choose the first slice. Mark unknowns instead of inventing certainty.
2. **Test with a minimal slice.** Build one end-to-end capability that is small enough to expose whether the vocabulary, boundaries, and enforcement are useful.
3. **Capture evidence.** Record repeated decisions, agent confusion, extra discovery steps, unsafe escape hatches, review friction, and user outcomes in the active `context.md`.
4. **Revise before routing around the thesis.** When code wants an exception or workaround, first decide whether the implementation is wrong or the thesis is incomplete. Do not normalize the workaround and leave the governing idea stale.
5. **Propagate.** A thesis revision must update affected product, architecture, security, Skill, catalog, and harness contracts in the same change.
6. **Repeat.** Early projects should expect frequent thesis revisions. Mature projects revise less often, but they still change when user, incident, compatibility, or maintenance evidence justifies it.

A thesis change is not complete when only `docs/00_theses.md` changed. Its consequences and mechanical enforcement must agree across the repository.

## Non-negotiable invariants

1. **The public CLI expresses user tasks.** Do not expose a vendor method, arbitrary route, or raw transport escape hatch as a shortcut.
2. **Discovery and action stay separate.** A `RoleDiscover` command may return candidates and opaque references. A `RoleAct` command requires at least one unique opaque reference and accepts it unchanged; it does not rediscover, normalize, decode, or reconstruct the identifier. Required-reference chains must lead back to an invocable producer rather than a closed cycle.
3. **The four-layer dependency direction holds.** Domain has no outward dependency. Application depends on domain. Infrastructure depends on domain contracts. CLI is the composition root.
4. **Every externally visible operation declares an effect.** Use `operation.EffectRead`, `operation.EffectCreate`, or `operation.EffectWrite`; unknown effects fail closed.
5. **Mutations declare intent, target binding, impact, and outcome.** A create command binds exactly one required opaque CLI input as `parent_input` and declares no `target_id_input`. A write command binds a required opaque CLI `target_id_input` whose reference kind equals `TargetKind`, plus an optional distinct opaque parent role whose input is also required when declared. `target_inputs` contains only those role-bound inputs. `operation.Intent`, `operation.TargetRef`, and every base `operation.Impact` dimension must identify what the operation can affect before an adapter performs it. After the action call, preserve valid structured outcome faults before generic cancellation; collapse every unclassified result to non-retryable `unclassified_mutation_outcome` with a read-only reconciliation action.
6. **Side effects cross one controlled boundary.** Do not give a command or use case an unrestricted executor, filesystem, process, or network client.
7. **The catalog is the public-command source of truth.** Routing, help, role, reference flow, and command tests derive from `cli.Catalog`; do not create a competing registry. Root agent help is an outcome/capability index only, with at most 512 encoded bytes per command entry; retrieve inputs, output, authentication, failures, mutation facts, and workflows through an exact-command or namespace selector. Recovery commands are exact catalog paths or `help <exact-path-or-namespace>`; do not append unchecked argv.
8. **Claims are executable.** When adding an invariant, add the type, lint, contract test, or release check that detects its violation.
9. **The public boundary stays clean.** Never add credentials, confidential URLs, private organization identifiers, real personal data, or copied private history.
10. **One gate decides completion.** Finish implementation work only when `task check` passes. Publication work also requires `task public:check`; release work requires `task release:check`.
11. **External calls are bounded and secret-free above infrastructure.** Propagate one context, declare pagination/call policy, and keep OAuth tokens, PATs, and credential-bearing types inside infrastructure.
12. **External text remains untrusted data.** Visible projection protects terminal and TSV/JSON structure by distinguishing backslashes, controls/formats, and Unicode line separators; it does not filter printable prompt-like meaning. Opaque references bypass display projection and retain their exact validated value.
13. **Supported agent outcomes own their semantics; presentation stays replaceable.** A supported task must not require `jq`, `grep`, a custom join/parser, raw Chatwork-notation interpretation, source inspection, or an exploratory API call. Typed results distinguish explicit resolved, explicit unresolved, and absent relations; To, quotes, names, proximity, and prose never fabricate reply edges. Candidate C is the accepted first presentation contract, not a product invariant or domain model. Compare future replacements in parallel against the same semantic fixtures on required understanding quality, canonical-reference use, token cost, tool steps, safety, and maintenance before changing that contract.

## Layer responsibilities

```text
internal/domain/   Pure vocabulary and invariants. No I/O.
internal/app/      User-task interpretation and ports owned by each use case.
internal/infra/    Concrete adapters. No product-policy decisions.
internal/cli/      Command catalog, arguments, presentation, and dependency wiring.
```

Application packages define the smallest port needed by their task. Infrastructure satisfies that port through Go's structural typing. Application code must not import infrastructure or construct transport-specific requests.

`cmd` and `internal/cli` have no third-party imports in the template. Keep provider SDKs and transports in `internal/infra`. If a derived project needs a presentation-only CLI parser or renderer, first accept an ADR or thesis consequence that explains the need and supply-chain tradeoff; then add only the exact package path to `allowedCLIThirdPartyImports` in `tools/archlint/main.go`, review its license and dependency delta, and add a negative test proving that sibling paths and effectful packages are still denied. Never add a wildcard, module prefix, SDK, or transport to that allowlist.

The default vertical slice is:

```text
cmd/cwk
  -> internal/cli
  -> internal/app/doctorcmd
  -> internal/domain/doctor and internal/domain/operation
  -> internal/infra/systemdoctor
```

The `sample list` and `sample read --id` pair follows the same layering through `internal/app/samplecmd`, `internal/domain/sample`, `internal/infra/sampledata`, and `internal/cli/sample.go`. It is the reference implementation for discover/act roles and exact opaque-ID flow.

## Working method

For a non-trivial change, create a directory under `docs/work/<change-name>/` starting from [the work-packet goal template](docs/work/_template/goal.md):

- `goal.md`: outcome, non-goals, acceptance criteria
- `context.md`: verified facts, constraints, and unknowns
- `plan.md`: chosen approach, alternatives, risks, and verification
- `tasks.md`: atomic checklist with evidence

Durable conclusions belong in theses, architecture, security, or an ADR. Do not leave lasting policy only in an implementation plan.

When the same design choice, workaround, or point of confusion appears twice, treat it as thesis evidence. Record it before adding another local special case.

Use this writing discipline:

- Production code explains **how**.
- Tests state **what** must remain true.
- Commit messages explain **why** the change exists.
- Code comments explain **why not** a plausible alternative.

Observe runtime-only behavior before changing it. Add bounded diagnostics, reproduce the behavior, record the evidence in `context.md`, and then implement the smallest verified fix.

## Adding a command or capability

1. Define the user outcome and test it against the current theses. Revise a weak thesis before adding a code-level exception.
2. Classify the command with `RoleUtility`, `RoleDiscover`, or `RoleAct`; declare reference kinds on structured inputs and output fields so produced/consumed edges are derived.
3. Prefer extending an existing task when the outcome is the same.
4. Add or refine domain vocabulary and its invariants.
5. Add an application use case with task-specific input, output, and ports.
6. Implement a concrete infrastructure adapter behind those ports.
7. Register one complete `cli.CommandSpec` in `cli.Catalog` and derive routing, scoped help, capability, role, reference flow, output, prerequisites, and recovery metadata from it. Keep the root agent index limited to path, namespace, summary, outcome, capability, effect, and role; verify detailed metadata only in scoped help. A `complete` output has no public pagination binding; a `paged` output is JSON-only and binds one optional opaque cursor argument or flag to one same-kind, always-present top-level string cursor beside the JSON envelope, with typed `empty_cursor` completion.
8. Declare `Effect`, `Intent`, `TargetRef`, and `Impact`. Bind create scope through `MutationContract.parent_input`; bind a write's existing target through `target_id_input` and any distinct scope through optional `parent_input`. Make missing, unbound, mismatched, or inconsistent values fail before the side effect.
9. For an external API, bind the secret-free authentication requirement, declare every standard `app/authn.Gate` fault plus any provider-specific fault in the catalog, issue `BindingID` only inside infrastructure, pass the validated session's non-serialized binding unchanged through each authenticated task port, resolve and revalidate that exact infrastructure authentication record immediately before I/O, and declare pagination/call policy, provider fault mapping, and publishable schema fixtures before enabling live I/O. Never pass credential-bearing clients or provider types into application code.
   For the fixed Chatwork first implementation, `.harness/chatwork_api_v2.json` pins one attempt; 20-second metadata/read and non-upload timeout; 60-second upload timeout; 8 MiB success, 64 KiB provider-error, 16 MiB output, 10,000-item aggregate, documented 100-item endpoint, and 5 MiB upload ceilings. Ordinary exact creates/updates need no additional confirmation; the reviewed access-change set requires exact `--confirm access-change`, and the destructive set exact `--confirm destructive`. Unknown mutation outcomes are non-retryable and reconcile through an exact read-only task. Production uses typed constants and boundary tests rather than loading the harness manifest at runtime.
10. Add unit, contract, opaque-reference round-trip, negative-path, hostile-output, recovery, and public-boundary tests in proportion to risk.
    For a relationship-rich Chatwork read, first add presentation-independent semantic fixtures, answer keys, negative relationship-inference tests, canonical-reference tests, and a zero-external-post-processing agent transcript. The first complete implementation uses the accepted candidate-C context capsule and adds golden tests for that contract. Select any future replacement only through a worktree competition with pinned agents, prompts, repetitions, token accounting, quality floor, and comparable raw evidence.
11. Propagate any thesis change through product, architecture, security, Skill, and harness documents.
12. Run `task check` and replay the relevant agent-readiness scenario.

## Verification commands

```sh
task check:fast
task check
task security
task release:check
task public:check
```

The underlying interface is `./scripts/check.sh fast|full|security|release|public`. Codex hooks call that interface and do not reimplement the checks.

Do not weaken a check merely to make a change pass. If a check encodes the wrong policy, update the governing document and test the new policy as part of the same reviewed change.

## Scope and safety

- Preserve unrelated user changes in a dirty worktree.
- Do not rewrite Git history, delete data, publish, or create releases unless the task explicitly requires it.
- Do not fetch or embed external content without verifying its license and integrity.
- Use synthetic fixtures such as `example.com`, deterministic timestamps, and non-secret tokens.
- Keep repository documentation in English unless the derived project's theses explicitly establish another language policy.
- Security reports follow [SECURITY.md](SECURITY.md), never public issues containing sensitive details.
