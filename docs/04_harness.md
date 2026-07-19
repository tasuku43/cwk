# Harness

The harness is the executable counterpart of the theses, product contract, architecture, security model, and release policy. Its goal is not to maximize the number of tools. Its goal is to make important regressions fail through one understandable interface.

## One gate, several profiles

`./scripts/check.sh` is the canonical check implementation. Every other entry point delegates to it.

| Profile | Task alias | Intended use | Includes |
|---|---|---|---|
| `fast` | `task check:fast` | Short local feedback loop | Formatting, architecture checks, capability/schema contracts, focused unit and contract tests |
| `full` | `task check` | Required pre-merge gate | Fast profile plus vet, race, generated-diff, and full test checks where applicable |
| `security` | `task security` | Security and dependency changes | Repository guard, module integrity, pinned static and vulnerability analysis |
| `release` | `task release:check` | Packaging and release changes | Artifact, metadata, checksum, Formula, and workflow contracts |
| `public` | `task public:check` | Bootstrap completion and public publication | Ready-profile identity, forbidden-data, required-file, license, capability/schema contracts, and public-boundary checks |

Direct invocation is supported for automation:

```sh
./scripts/check.sh fast
./scripts/check.sh full
./scripts/check.sh security
./scripts/check.sh release
./scripts/check.sh public
```

The canonical gate and release packager force module mode and neutralize ambient Go workspace, toolchain, experiment, FIPS, and flag settings before invoking Go. This prevents a local or CI `GOFLAGS` value from silently selecting no tests and keeps agent, developer, and workflow evidence on the same checked command set. A release fixture launches the public profile with hostile values and proves that its first Go-backed check observes only the sanitized contract.

CI is the completion authority. A local hook may run `fast` to reduce latency, but it must call this script and must not claim equivalence to `full`. The Codex `Stop` hook resolves its script from the Git root so it also works from a subdirectory; after the user completes [Codex's project-hook trust review](https://learn.chatgpt.com/docs/hooks), a failed fast gate returns a structured `continue: false` result that tells the agent to repair and rerun the canonical command.

## Harness components

### `.harness/project.json`

This file is the machine-readable source for template identity, bootstrap state, exact runnable defaults, and repository policy. The bootstrap tool validates it before replacement and changes its profile from `template` to `ready` only after successful application.

`binary_name` is a portable lowercase executable basename. Validation rejects the case-insensitive Windows device names `CON`, `AUX`, `PRN`, `NUL`, `COM1` through `COM9`, and `LPT1` through `LPT9`; adding `.exe` does not make those names extractable on Windows. This is part of the default release-matrix contract, not a naming-style preference.

Policy that must be reviewed by both humans and tools belongs here when it is finite and structural, such as forbidden private identifiers or expected module and binary names. Product reasoning remains in documentation.

### `tools/bootstrap`

Bootstrap performs validated exact replacement of `github.com/tasuku43/cwk`, `cwk`, and `Chatwork CLI`. It does not search-and-guess arbitrary names.

Always preview first:

```sh
go run ./tools/bootstrap --dry-run
go run ./tools/bootstrap
```

Bootstrap failure must leave the repository in a diagnosable state and must not claim the project is ready. Bootstrap changes identity; it cannot complete theses, threat models, or release promises.

### `.agents/skills/bootstrap-derived-cli`

`$bootstrap-derived-cli` is the first-run Codex workflow for a derived repository. It does not implement a second replacement engine: it resolves missing identity decisions, invokes `tools/bootstrap` in preview-then-apply order, verifies the resulting module/import/command paths and gates, then requires a project-specific thesis and security handoff before `$add-capability`. `tools/repoguard` requires both the Skill instructions and their Codex interface metadata, while the Skill's workflow delegates mechanical safety to the same bootstrap and check commands used by humans and CI.

For a newly derived repository, the Skill leaves provider authentication,
credential storage, side-effect approval, user tasks, and release ownership to
that project's theses and security model. In this derived `cwk` product, ADR
0003 has already selected process-local PAT only. A `ready` profile proves only
that identity replacement completed.

### `tools/archlint`

Architecture lint checks production dependency direction, rejects unclassified production packages, and keeps each `cmd/` entrypoint limited to argument/stream handoff, signal cancellation, the CLI composition root, and process exit. It merges Go package information for the native build and every release target on Linux, macOS, and Windows, so a platform-specific file cannot hide a forbidden dependency from the host CI platform. Source checks reject detached application, infrastructure, and CLI contexts, default HTTP clients, application-layer `fmt` presentation/scanning calls, built-in `print`/`println` in domain, application, CLI, and command packages, authentication-binding issuance outside infrastructure, and command-entrypoint access outside the narrow selector allowlist. Domain and application packages cannot import `log`, `log/slog`, or Cgo. Reviewed user-facing presentation belongs in CLI and must use its injected streams; observability and native integration are explicit derived-project infrastructure policies. Any allowed exception must be narrow, named, and tested.

The template also rejects every third-party import from `cmd` and `internal/cli` by default. Vendor SDKs, authenticated transports, and other effectful clients belong in `internal/infra`, where third-party imports remain available and the dependency/security gates review them. A derived project may allow a CLI parser or renderer only by adding its exact package path to `allowedCLIThirdPartyImports` in `tools/archlint/main.go`. The same change must include an accepted ADR or thesis consequence, license and dependency review, and a regression test proving that sibling packages, module-wide prefixes, SDKs, and transports remain rejected. Wildcards and prefix allowlists are not valid exceptions.

### `tools/repoguard`

Repository guard checks public-boundary and repository-shape policy, including bootstrap state, forbidden identifiers, likely secrets, invalid or leftover identity, and required public files. A derived project extends its policy when it adds credentials, private migrations, generated content, or publication constraints.

### `tools/contractlint`

Contract lint validates the executable catalog before checking two repository ledgers:

- [`.harness/capabilities.json`](../.harness/capabilities.json) records supported and deliberately unsupported user capabilities without copying command paths. Each public capability ID must appear in at least one `AgentContract.CapabilityID`, every catalog capability must be public, and an `internal`, `deferred`, or `excluded` entry must remain absent from the catalog and explain why.
- [`.harness/schemas.json`](../.harness/schemas.json) pins publishable external-schema fixtures by repository-relative path and exact SHA-256 digest. Each entry also records provenance and license. An explicit empty array is valid before the project adopts an external schema.

Both ledgers are strict JSON and must themselves be regular files reached without symbolic links. Unknown or duplicate object keys, duplicate IDs, malformed lowercase dot IDs, trailing values, and implicit `null` lists fail. Capability command paths remain owned only by the catalog; adding them to the ledger creates forbidden duplication rather than useful documentation.

The optional runtime command-selection allowlist does not change this check.
`contractlint` always validates the complete `DefaultCatalog` against the
capability and Chatwork-operation ledgers. The active attention view is a
catalog-derived local projection, not a reason to mark a supported capability
internal, deferred, excluded, or uncovered.

The derived project also owns `.harness/chatwork_api_v2.json`, a fixed upstream-operation snapshot rather than a public-command registry. Its exact 32 operation IDs and method/path pairs are pinned to the official 2026-07-18 documentation index and map only to capability IDs. Contract validation rejects a same-sized substituted operation set, missing or duplicate operation IDs, method/path drift, unknown capabilities, or a Chatwork-backed capability with no upstream owner. The manifest is now `coverage_status: complete`, so any operation without at least one public capability owner fails the gate. Returning it to `planned` would reopen, not complete, this work. Future provider additions require a new reviewed snapshot decision; they do not silently extend the active goal.

The manifest also pins the numeric implementation contract: 20-second metadata/read timeout, 60-second upload timeout, one attempt, 8 MiB successful response body, 64 KiB provider error body, 16 MiB complete output, 10,000 aggregate list items, the five reviewed provider operations with documented 100-item limits, and 5 MiB upload input. Its mutation policy fixes exact-invocation as the default, the precise operation-ID sets requiring `--confirm=access-change` or `--confirm=destructive`, and read-only reconciliation for uncertain outcomes. `contractlint` validates those exact values and sets. Runtime code does not read this manifest; boundary-specific tests compare independently typed production policy with the same accepted decisions.

Message boundary tests pin `200/204/404` both with and without the two official
limitation headers, reject every value other than the sole official `true`,
keep limitation-summary prose private, and prove partial/all restriction never
renders as normal empty or not-found. Parser fixtures cover each official To,
reply, quote, and complete code region plus malformed/unclosed/contradictory
forms; malformed notation preserves escaped body and sibling records while
producing an unknown whole relation set with no partial facts.

Mutation boundary tests require `rooms create --account` to reach `/me` and
`POST /rooms` through the same private binding, prove an exact mismatch or
preflight failure makes zero room-create calls, and inspect the room form for
absence of owner/account fields. They also reject generic/mismatched bindings,
overlong names, and unknown icon presets before room I/O. Invite-link fixtures require code XOR
regeneration, approval, and nonempty description before authentication; they
pin the complete PUT form, code alphabet/length, explicit regeneration-only
omission, create description support, exact result-reference binding, and zero
calls for every empty/partial replacement.

Boundary tests independently pin the provider-specific rate-limit contract:
one strict official `x-ratelimit-reset` within five minutes, provider `Date`
as the duration baseline with a local-clock fallback, no
`Retry-After` fallback, exact bounded room-posting error classification to 10
seconds, unknown timing for all unproved cases, and distinct read/mutation
retryability and recovery. Fault validation permits advisory `retry_after` on
only a non-retryable rate-limit fault; CLI snapshots require text `unknown`
and JSON `null` when timing is absent. The transport attempt ceiling remains
one, so these tests do not introduce an adapter retry loop.

Capability status has a narrow meaning:

| Status | Meaning |
|---|---|
| `public` | At least one catalog command exposes this supported user capability |
| `internal` | The implementation may use it, but no public command may expose it |
| `deferred` | The product may add it later, but it is unsupported now |
| `excluded` | The current product contract deliberately does not support it |

Several commands may share one public capability ID when discover and act commands form one user workflow. Conversely, one command declares exactly one primary capability; splitting a command across unrelated outcomes is a product-design signal, not a ledger shortcut. Non-public entries require a reason so an agent does not mistake absence for an implementation gap.

Schema paths must be canonical repository-relative paths below a `testdata` directory. Every path component is inspected without following symbolic links, and the target must be a regular file. A digest mismatch requires reviewing the upstream change and updating the manifest deliberately; the tool never rewrites a digest. `repoguard public` separately checks the same fixture content for public-repository policy, so a matching digest is not permission to publish a secret or unlicensed material.

Run the focused check with:

```sh
task contracts:check
```

The same tool runs in `fast`, therefore in `full`, and directly in `public`. There is no CI-only capability or schema interpretation.

When adding an external API, first record every considered user capability in the capability ledger, including deliberately deferred and excluded outcomes. Promote an ID to `public` only in the same change that adds a validated catalog contract. When vendoring an upstream schema or response fixture, record its source and publication license, compute the digest from the exact bytes, and add adapter contract tests. A schema digest proves identity, not compatibility: tests must still fail when a reviewed upstream change violates the domain mapping.

### Tests

The test suite has complementary levels:

- Domain tests fix pure invariants.
- Application tests fix task interpretation, orchestration, and ambiguity behavior.
- Authentication, pagination, and mutation-boundary tests prove rejection/cancellation before downstream calls, exact secret-free authentication binding, complete standard runtime-fault declarations, and complete-or-no-result behavior.
- Chatwork PAT-only composition tests prove that `CWK_API_TOKEN` is the sole
  credential input, every requirement admits only `pat`, and a missing or
  invalid token makes zero provider task requests. They also prove that the
  removed `CWK_AUTH_METHOD` value cannot select another path.
- Chatwork authentication-binding tests use synthetic tokens and two isolated
  clients to reject missing, stale, wrong-client, and cross-session bindings;
  no automated test reads a developer credential or contacts live Chatwork.
- Chatwork secret-canary tests prove that the token reaches only the exact
  `x-chatworktoken` request header and never argv, output, errors, logs,
  snapshots, fixtures, or persistent configuration.
- Catalog pagination tests require an exact optional-input/top-level-string-output opaque cursor binding, typed empty-cursor completion, and JSON-only presentation for `paged` results, and forbid that binding for `complete` results. Renderer fixtures reject an omitted, null, or non-string cursor.
- Infrastructure tests fix protocol conversion and boundary failure.
- CLI tests fix routing, help, rendering, and exit behavior.
- Human-help navigation tests require one root entry per direct command or
  top-level namespace, catalog-relative ordering within the direct and
  namespace sections, exact namespace membership/counts, no leaf duplication at
  root, relative namespace listings, natural trailing-help equivalence, and
  unchanged unknown-command faults.
- Command-selection catalog tests keep exactly `help`, `doctor`, `version`, and
  `config` always-on; prove missing state enables every current configurable
  Chatwork command while a saved allowlist keeps later additions off; and make
  root, namespace, exact, trailing, agent, recovery, workflow, and routing
  views agree. They reject a visible consumer without a reachable producer and
  a visible recovery action whose command is hidden, without auto-enabling
  either dependency.
- Command-selection TTY tests drive the single `config` selector through a
  synthetic terminal. They require textual `[read]`, `[create]`, and `[write]`
  effect badges even when color is unavailable, catalog-derived color spans
  that never replace those labels, deterministic Up/Down movement, ASCII and
  fragmented UTF-8 U+3000 Space toggling, Enter-only save, and
  q/Escape/Ctrl-C cancellation. Renderer tests require the exact command path
  to begin at one display column for read, create, and write rows. An invalid
  active view remains inside the selector with an actionable diagnostic and
  zero writes. Every exit path restores the alternate screen, cursor, output
  mode, and raw input mode; restoration completes before an Enter-confirmed
  save, and restoration failure prevents that save. Non-TTY stdin or stdout
  produces `interactive_terminal_required` without ANSI output or persistence.
  Width/height tests also prove a hidden or truncated exact command disables
  movement, toggle, and save while retaining a non-saving exit.
- Command-selection storage tests cover XDG behavior on macOS/Linux, AppData on
  Windows, strict bounded JSON, stale paths, modes, symbolic links, special
  files, Unix rename/directory durability, Windows replace-existing behavior,
  pre-Enter q/EOF/context cancellation, and unchanged prior bytes after an
  invalid selection. Repair tests distinguish malformed content from unsafe or
  inaccessible storage, reject false empty root help, migrate formerly
  selectable `doctor` and `version` entries without reintroducing them as
  choices, and preserve confirmed success under late cancellation. Disabled
  invocation must be `unknown_command` with zero PAT resolution and zero
  provider calls; re-enabling must not bypass the original authentication or
  mutation confirmation contract.
- Command-selection reconciliation tests add one bounded `command-selection`
  check to always-on `doctor`. Synthetic default, saved, stale, legacy,
  malformed, unsafe, and unavailable states fix its state/source,
  enabled/disabled/stale/legacy counts and deterministic ordered SHA-256
  fingerprint. An uncertain save records expected `source=saved` and its
  candidate fingerprint; tests require both to match the subsequent doctor
  result and reject the same fingerprint reported as `source=default`, so an
  uncertain write is inspected without a second mutation or false success.
  Scoped agent help publishes the exact dynamic error-message grammar, and a
  JSON error fixture must match it byte-for-byte around the fingerprint.
- Config success-output tests fix the two-line natural-Japanese
  visible/hidden/change transcript, require cleanup prose only for a nonzero
  stale-plus-legacy count, and reject internal key/value labels or a fingerprint
  on confirmed success. Fingerprint tests remain on the uncertain fault and
  `doctor` reconciliation path.
- Terminal-adapter tests require both stdin and stdout to be terminals, cover
  setup rollback, short/control-write failure, idempotent restoration, sizing,
  ready input, and blocked-read cancellation followed by a successful later
  read. They compile the pinned `x/term` plus `x/sys` platform implementation
  for Darwin, Linux, and Windows. Unix coverage fixes bounded descriptor polling
  and reading; Windows coverage fixes a locked reader thread, exact thread-
  handle cancellation through `CancelSynchronousIo`, join-before-return,
  VT-output enablement, and restoration of the prior console mode. Cancellation
  must leave no reader that can consume a later invocation's terminal input.
- Formula audit tests begin with a deliberately owner-only `0600` rendered
  Formula and require the isolated tap copy to be exact non-executable `0644`
  before strict Homebrew audit. This preserves source bytes while preventing
  runner umask from making a valid Formula unreadable to Homebrew.
- Exact human-help tests derive required, repeatable, source, allowed-value,
  reference-kind, and description facts from catalog inputs, including the
  multi-sender message selection contract and the inclusive 1..100 primary
  message limit.
- Catalog hostile-input tests reject invalid UTF-8, terminal controls, Unicode
  format controls, line separators, and whitespace in non-argv input names
  before exact help can render them.
- Agent-help shape and size-growth tests keep root discovery index-only while scoped help retains the complete invocation and recovery contract.
- Localization contract tests require Japanese summaries, outcomes, input and
  output descriptions, prerequisites, and recovery reasons for every public
  catalog command while separately proving that command paths, flags, allowed
  values, fault kinds/codes, JSON keys, and opaque references remain unchanged.
- JSON-output contract tests compare each built-in renderer's schema version, envelope, and item keys with its catalog `CommandOutput` declaration, and enforce the always-present string cursor for any paged probe.
- Adversarial output tests keep TSV/JSON records and stdout/stderr ownership intact across controls, Unicode format/line separators, existing backslashes, and printable prompt-like data while preserving opaque IDs exactly.
- Catalog tests scan every public command for completeness and unique paths.
- Catalog syntax tests reject command/namespace prefix collisions, usage/`Required`/`AllowedValues` drift, and missing common runtime failure declarations.
- Reference-graph tests connect discover producers to act consumers by kind and exact field/argument declarations.
- Opaque-ID round-trip tests pass discovery output unchanged into action input.
- Negative tests prove rejection before side effects.
- Release tests inspect actual artifacts and metadata, not only workflow text.
  Workflow checks additionally enumerate every Formula-job step start and
  validate the exact checksum, toolchain, render/audit, shared-tap token,
  checkout, staging, and pull-request steps. They confine the App action and
  each secret to one reviewed workflow-wide occurrence; fix repository and
  requested permissions, read-only audit-job source token, zero
  source-repository permissions in the token-bearing job, non-persisted
  checkouts, audit-job artifact upload, fresh publish-runner dependency, the
  absence of tagged-source checkout or checked-out code execution in that
  privileged runner, reviewed workflow and Formula-job fields without ambient
  `env` or `defaults`, non-symbolic Formula destination paths, exact
  Formula-only PR path and conventions, and
  render/audit-before-write ordering. Negative workflow mutations prove that
  broader scope, another token/path/base, wildcard staging, an extra permission
  or token consumer, duplicate secret use, ambient runtime injection, symbolic
  destination acceptance, changed Formula-source binding, or ignored audit/PR
  failure fails. The App's external installation maximum remains a named
  manual release-owner review.
- Shared semantic fixtures and answer keys fix relationship, identity, bounds, coverage, uncertainty, and hostile-text facts independently of presentation.
- Relationship tests prove that To, quote, time proximity, display names, and layout-looking content do not fabricate reply edges.
- Bounded message-selection tests prove sender OR precedes newest-N selection,
  typed send time chooses anchors with later provider position breaking ties,
  reply context follows the limit, and rendered records retain provider order
  and source sequences. They distinguish requested limit and candidate count
  from the provider `source-limit`, allow explicit context to exceed N, reject
  invalid values before authentication/I/O, reject a source above declared
  coverage before selection, and keep local policy out of the one documented
  `force` request. CLI/runtime tests additionally prove omitted and explicit
  `recent` emit the latest-window request while explicit `changes` retains the
  differential request and output.
- No-post-processing agent transcripts fail if a supported task requires `jq`, `grep`, a custom join, raw notation parsing, source inspection, or an exploratory provider call.
- Presentation competitions pin fixtures, agent/model versions, prompts, repetitions, invocation budgets, answer scoring, token accounting, and latency measurement before candidate implementation.
- Candidate reports retain per-worktree correctness, next-action/reference, token, tool-step, byte, latency, reviewability, maintenance, benchmark-defect, and audit evidence. A selected presentation receives golden and compatibility tests only after an explicit compatibility decision; an inconclusive benchmark is never relabeled as a win.

A global coverage percentage is not a substitute for these contracts. Add tests at the boundary where a future regression would otherwise pass unnoticed.

## Claims-to-checks discipline

Every strong statement should identify its enforcement path.

| Claim type | Preferred enforcement |
|---|---|
| Layer dependency | Go-aware architecture lint and import-boundary tests |
| Finite domain state | Types, constructors, and table-driven negative tests |
| Catalog completeness | Whole-catalog contract tests |
| Discover-to-act composition | Reachable reference-graph validation, required act references, and byte-preserving round-trip tests, including positional `files list` file/room values passed unchanged to `files show` while `absent` is rejected as identity |
| Command-bound singleton action | Fixed-target catalog kind/ID/scope validation, no-reference exclusivity tests, and matching mutation-target tests |
| Side-effect ordering | Fake adapter counters and failure-before-I/O tests |
| Mutation outcome classification | Structured-fault-first/cause-stripping tests, non-retryable unclassified outcome fallback, and read-only recovery validation |
| Authentication precondition | Secret-free session contract, zero-downstream-call tests, and catalog validation of every standard gate fault's code/kind/retryability |
| Authentication binding | Opaque JSON-excluded/fmt-redacted binding type, infrastructure-only issuance lint, exact pass-through tests, and two-client/stale-binding adapter fixtures |
| Sole Chatwork PAT input | CLI/composition tests requiring `CWK_API_TOKEN`, PAT-only requirement snapshots, obsolete-selector non-effect, and zero-call missing/invalid-token failures |
| PAT process-local binding | Synthetic two-client/stale-binding fixtures, exact unchanged binding pass-through, fixed-destination header tests, and no persistent credential source |
| Authentication secret exclusion | Token-canary scans across argv rejection, stdout, stderr, structured faults, logs, snapshots, fixtures, test diagnostics, and repository state |
| Pagination completeness | Cursor loop/budget/cancellation tests, retryability/catalog agreement, and no-partial-result assertion |
| Public paged continuation | Catalog validation of one exact same-kind optional input/top-level output binding, JSON-only presentation, and agent-help/reference-workflow projection |
| Retry safety | Timeout/attempt/idempotency validation and adapter contract tests |
| Chatwork message truthfulness | Status/header matrix, private-summary canary, normal-zero/partial/all/restricted/not-found distinctions, relation unknown/absent tests, and hostile escaped-body list continuity |
| Chatwork room identity and invite replacement | Same-binding `/me` exact-match tests with zero-POST failures; owner-free room form; pre-auth code/XOR/full-field validation; exact PUT form and regeneration omission tests |
| Chatwork rate-limit evidence | Strict header/body parsing tests, five-minute plausibility bounds, read/mutation catalog signatures, advisory-timing validation, and text/JSON unknown-timing snapshots |
| Agent recovery | Catalog fault declarations, exact-path/help-selector executable grammar tests, and structured error snapshots |
| Hierarchical human discovery | Catalog-derived direct-command/namespace partition, unique section-relative ordering and namespace counts, selector round-trip, no root leaf leakage, namespace-size growth, trailing-help equivalence, exact input projection, and hostile non-argv name rejection tests |
| User-selected command attention view | Complete `DefaultCatalog` contract lint plus configurable-leaf metadata, an exact-path ordered active view shared by every help/routing/recovery/workflow projection, exactly four always-on commands (`help`, `doctor`, `version`, `config`), actionable required-reference/recovery closure validation, the single TTY selector's textual effect badges and key-state tests, Enter-only persistence, natural-Japanese confirmed-save golden output without recovery internals, all-exit terminal restoration, context-responsive platform reads with no abandoned input consumer, typed non-TTY failure, invalid-view retention, legacy local-command migration, strict platform storage with Unix durability and explicit Windows limits, uncertain-fault/doctor count and fingerprint reconciliation, disabled zero-PAT/provider-call tests, and re-enable tests that retain existing security policy |
| Bounded agent root discovery | Fixed root-index shape, 512-byte per-command entry validation, and 100-command growth/selection tests |
| External text structure | Visible-projection unit/E2E tests plus scoped I/O trust metadata; printable meaning remains explicitly out of scope |
| Agent command certainty | Root/scoped help round-trip tests plus task transcripts with no command probing or prose scraping |
| Supported outcome completeness | Transcript assertion of zero external post-processing and declared provider/context coverage |
| Context relationship truth | Presentation-independent typed fixtures and negative inference tests for To, quote, names, proximity, and missing references |
| Presentation eligibility | Shared semantic answer key, canonical-reference/coverage/trust checks, determinism, and zero external post-processing |
| Presentation selection | Parallel-worktree comparison with pinned agent tasks, model/tool versions, repetitions, token accounting, latency, and raw per-candidate results |
| Presentation decision provenance | Retained raw runs, score summaries, audit findings, and benchmark-defect records that distinguish an experiment result from a later owner compatibility decision |
| Current success text | All-route and golden tests require the headerless task projection. Seven reviewed homogeneous collections require exactly one trust/schema prelude even when empty and one provider-order physical line per item with stable canonical positions and optional suffixes; `files list` fixes `message_ref` as canonical-or-`absent`. `messages list` additionally requires one room/trust/fixed-schema header with the provider bound named `source-limit`, a document-local actor dictionary, positional canonical message/time/body values without repeated labels, and flat provider-order adjacency records. Tests preserve migration history without claiming a Competition 1 winner |
| Subtractive task projection | Catalog/result field checks and negative canaries allow only declared task facts, exact canonical references, task-relevant bounds/completeness/uncertainty, and external-text trust framing. Message actor aliases are allowed only as document-local compression with canonical dictionary entries; semantic raw-notation records, wire extras, derived thread metadata, and non-contract defaults fail |
| Bounded message selection | Domain/application truth tables, adapter-request guards, scoped-help/runtime tests, and active synthetic agent scenarios require omitted or explicit `recent` to select the latest bounded window, explicit `changes` to select differential retrieval, exact sender OR inputs with machine-readable repeatability and a 100-reference bound; optional primary `--limit` 1..100; sender predicate then typed-send-time newest-N selection with later-position tie-break then direct typed one-hop reply expansion; context allowed beyond N; provider-order gapped source sequences; source/candidate/requested-limit and anchor/context distinction; one documented `force` request with no pagination; pre-I/O invalid-input and over-bound-source rejection; canonical-reference reuse; and zero rendered-text filtering or raw-notation inference |
| Token efficiency | Pareto comparison among quality-eligible candidates followed by a selected-format non-regression budget |
| Public capability coverage | Exact bidirectional match between capability ledger and catalog `CapabilityID` values |
| Fixed Chatwork API coverage | Strict 32-operation snapshot plus bidirectional operation-to-public-capability validation |
| Fixed Chatwork resource bounds | Exact typed snapshot values plus transport, provider source-cardinality, aggregation, upload, and output boundary tests; local message selection cannot hide a response above declared coverage |
| Chatwork mutation confirmation | Exact operation-ID policy sets plus typed invoker/CLI zero-call tests |
| External schema compatibility | Vendored fixture, generator, and drift test |
| Secret or private-data exclusion | Repository policy, scanner, and synthetic fixtures |
| Reproducible generation | Regenerate and require a clean diff |
| Artifact integrity | Build, inspect, checksum, and install tests |
| Shared Homebrew tap publication | Release lint validates the workflow and Formula-job field allowlists, every Formula-job step start, and exact checksum/toolchain, render/audit/artifact upload, fresh-runner artifact validation, token, checkout, staging, and PR shapes; confines the App action and each secret to one reviewed workflow-wide occurrence; fixes the audit-to-publish job dependency, public tap repository, requested permissions, read-only audit-job source token, zero source-repository permissions in the token-bearing job, absence of ambient `env`/`defaults`, tagged-source checkout, or checked-out code execution there, exact audited Formula binding/path, non-symbolic destination paths, PR branch/title prefixes, pinned actions, and render/audit-before-write order; negative mutation fixtures prove alternate/extra fields or steps, external secret/action use, runtime injection, symbolic-path acceptance, same-runner code execution, representative drift, and ignored failures are rejected; the release owner manually confirms the external App installation maximum |
| Documentation command | Execute or parse the canonical snippet where practical |
| Japanese default user experience | Whole-catalog natural-language validation, human help/TUI/error snapshots, stable-identifier regression tests, and repository checks for the active Japanese entry documents and GitHub templates |

If no practical mechanical check exists, state the manual review step and why automation is not reliable. Do not describe a manual convention as mechanically guaranteed.

## Adding an invariant

1. State the invariant and the failure it prevents in the governing document.
2. Identify the smallest code mutation that would violate it.
3. Put validation at the narrowest shared boundary.
4. Add a test or lint fixture that fails for the mutation.
5. Give the failure an actionable message with file, rule, and next step.
6. Add the check to the appropriate `scripts/check.sh` profile.
7. Confirm local Task and CI paths exercise the same implementation.

Do not add a grep that checks only whether a function name exists when the real claim concerns behavior. Prefer types, AST analysis, runtime validation, and contract tests in that order of semantic strength.

## Generated and automated changes

Generation is allowed when it reduces hand-maintained duplication without making the public product dynamic at runtime.

- Inputs and tool versions are reviewed and pinned.
- Generated output is committed only when repository policy requires it.
- Regeneration is deterministic.
- Generated code cannot register public commands implicitly.
- Generated schema fixtures must retain reviewed provenance and license metadata and an exact manifest digest.
- Automated updates use pull requests and the same profiles as human changes.
- A passing generator does not classify a new capability or side effect on behalf of a reviewer.

## Failure handling

A failed check is a work item, not an obstacle to bypass. Fix the implementation or, when policy is wrong, update the governing decision and its enforcement together. Do not:

- delete a negative test without replacing its guarantee;
- add a broad lint exclusion;
- switch a pinned tool to `latest` to obtain a passing result;
- make CI and local checks silently diverge;
- suppress output that a contributor needs to act on the failure.

Record nondeterministic failures with inputs, platform, and logs in the active work packet before changing timeouts or retries.

## Completion rules

- Ordinary implementation: `task check`
- Security boundary or dependency: `task check` and `task security`
- Public repository change: `task check` and `task public:check`
- Release or packaging change: `task check` and `task release:check`
- First public release: all profiles, plus the manual review in [Public Repository](05_public_repository.md)
