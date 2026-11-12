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
| `public` | `task public:check` | Public publication | Ready identity, forbidden-data, required-file, license, capability/schema contracts, and public-boundary checks |

Direct invocation is supported for automation:

```sh
./scripts/check.sh fast
./scripts/check.sh full
./scripts/check.sh security
./scripts/check.sh release
./scripts/check.sh public
```

The canonical gate and release packager force module mode and neutralize ambient Go workspace, toolchain, experiment, FIPS, and flag settings before invoking Go. They unset `GOROOT` so the Go binary selected from `PATH` supplies its own standard library; synthetic fixtures provide an invalid `GOROOT` canary and prove it is removed before the first Go invocation. This prevents a local or CI `GOFLAGS` value from silently selecting no tests and keeps agent, developer, and workflow evidence on the same checked command set. Before any Go-backed profile work, `scripts/check.sh` validates the profile name, reads the required version from the `go` directive in `go.mod`, and checks the PATH-selected `go version` with numeric major/minor/patch comparisons. The selected Go must meet or exceed that directive. A missing or older version fails with Japanese guidance naming the required and detected versions and directing the contributor to install the required version or newer and put its `go` command on `PATH`. The gate retains `GOTOOLCHAIN=local` and does not download or switch toolchains. Fake-Go fixtures verify this preflight and that hostile ambient settings are sanitized before the first Go invocation.

Profiles are composable entry points, not a nesting hierarchy. `full` includes
`fast` and the additional implementation checks, but it does not invoke
`security`, `release`, or `public`. Those profiles remain mandatory when the
completion rules below select them. Release preflight invokes every required
profile explicitly, so it cannot depend on an undocumented transitive check.

CI is the completion authority. Pull-request CI runs `full` and the
security/public boundary profiles in parallel. The repository installs no
automatic Codex Stop hook: a per-turn fast run added latency, was sensitive to
the PATH-selected Go binary, and did not prove completion. A contributor may
configure local automation, but it must delegate to a named profile here and
must not claim equivalence to a profile it did not run.

## Harness components

### `.harness/project.json`

This file is the machine-readable source for project identity, exact runnable defaults, and repository policy. The retained `ready` profile records that identity derivation completed; it is not a product, security, or release-readiness claim.

`binary_name` is a portable lowercase executable basename. Validation rejects the case-insensitive Windows device names `CON`, `AUX`, `PRN`, `NUL`, `COM1` through `COM9`, and `LPT1` through `LPT9`; adding `.exe` does not make those names extractable on Windows. This is part of the default release-matrix contract, not a naming-style preference.

Policy that must be reviewed by both humans and tools belongs here when it is finite and structural, such as forbidden private identifiers or expected module and binary names. Product reasoning remains in documentation.

### `tools/archlint`

Architecture lint checks production dependency direction, rejects unclassified production packages, and keeps each `cmd/` entrypoint limited to argument/stream handoff, signal cancellation, the CLI composition root, and process exit. It merges Go package information for the native build and every release target on Linux, macOS, and Windows, so a platform-specific file cannot hide a forbidden dependency from the host CI platform. Source checks reject detached application, infrastructure, and CLI contexts, default HTTP clients, application-layer `fmt` presentation/scanning calls, built-in `print`/`println` in domain, application, CLI, and command packages, authentication-binding issuance outside infrastructure, and command-entrypoint access outside the narrow selector allowlist. Domain and application packages cannot import `log`, `log/slog`, or Cgo. Reviewed user-facing presentation belongs in CLI and must use its injected streams; observability and native integration are explicit derived-project infrastructure policies. Any allowed exception must be narrow, named, and tested.

The template also rejects every third-party import from `cmd` and `internal/cli` by default. Vendor SDKs, authenticated transports, and other effectful clients belong in `internal/infra`, where third-party imports remain available and the dependency/security gates review them. A derived project may allow a CLI parser or renderer only by adding its exact package path to `allowedCLIThirdPartyImports` in `tools/archlint/main.go`. The same change must include an accepted ADR or thesis consequence, license and dependency review, and a regression test proving that sibling packages, module-wide prefixes, SDKs, and transports remain rejected. Wildcards and prefix allowlists are not valid exceptions.

### `tools/repoguard`

Repository guard checks public-boundary and repository-shape policy, including the ready identity, forbidden identifiers, likely secrets, invalid or leftover template identity, required Codex guidance, and required public files. The template-only bootstrap implementation and Skill are intentionally absent after identity derivation. `repositoryPaths` and `checkFilesystemShape` read cached Git-index paths with `git ls-files --cached -z` and fail closed if that query is unavailable or malformed. The shape check rejects indexed `.claude/` components and `CLAUDE.md` basenames case-insensitively at any depth. Untracked regular local paths with those names are excluded from the publishable-content candidate set, so their local links, paths, text, and secret canaries are not read by publication scanners. Once staged or tracked, they remain candidates, receive the Claude-policy shape issue, and go through the ordinary scanners. The full working-tree shape walk still descends into local directories and rejects symbolic links, special files, bootstrap residue, and root build artifacts regardless of index state.

### `tools/contractlint`

Contract lint validates the executable catalog before checking two repository ledgers:

- [`.harness/capabilities.json`](../.harness/capabilities.json) records supported and deliberately unsupported user capabilities without copying command paths. Each public capability ID must appear in at least one `AgentContract.CapabilityID`, every catalog capability must be public, and an `internal`, `deferred`, or `excluded` entry must remain absent from the catalog and explain why.
- [`.harness/schemas.json`](../.harness/schemas.json) pins publishable external-schema fixtures by repository-relative path and exact SHA-256 digest. Each entry also records provenance and license. An explicit empty array is valid before the project adopts an external schema.

Both ledgers are strict JSON and must themselves be regular files reached without symbolic links. Unknown or duplicate object keys, duplicate IDs, malformed lowercase dot IDs, trailing values, and implicit `null` lists fail. Capability command paths remain owned only by the catalog; adding them to the ledger creates forbidden duplication rather than useful documentation.

The optional runtime command-selection allowlist does not change this check.
`contractlint` always validates the complete `DefaultCatalog` against the
capability and Chatwork-operation ledgers. The active dispatch scope is a
catalog-derived local projection, not a reason to mark a supported capability
internal, deferred, excluded, or uncovered.

The derived project also owns `.harness/chatwork_api_v2.json`, a fixed upstream-operation snapshot rather than a public-command registry. Its exact 32 operation IDs and method/path pairs are pinned to the official 2026-07-18 documentation index and map only to capability IDs. Contract validation rejects a same-sized substituted operation set, missing or duplicate operation IDs, method/path drift, unknown capabilities, or a non-excluded Chatwork-backed capability with no upstream owner. An `excluded` Chatwork capability may deliberately record an outcome for which that reviewed upstream snapshot has no operation; its nonempty reason is the evidence rather than an invented route. The manifest is now `coverage_status: complete`, so any operation without at least one public capability owner fails the gate. Returning it to `planned` would reopen, not complete, this work. Future provider additions require a new reviewed snapshot decision; they do not silently extend the active goal.

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

Message send/update boundary tests require exactly one literal, stdin, or file
body source. They pin the 65,535-byte UTF-8 ceiling, a 65,536-byte-limited stdin
read, byte-for-byte preservation of newlines/Chatwork notation/shell
metacharacters, Unix cancellation of a waiting inherited pipe without closing
it or consuming later input, Windows exact-thread cancellation and join,
infrastructure-only context-carrying regular-file access with its own
65,536-byte read ceiling, typed `invalid_input`
faults, private body/path/read-error redaction, and zero provider HTTP calls for
every missing, unreadable, empty, oversized, NUL-containing, or invalid UTF-8
source.

Typed message-send probes pin `--to` to at most 100 unique canonical account
refs in input order and `--reply-to` to one canonical exact message ref. They
verify both refs remain ordered preflight intent while `--room` remains the
only mutation parent, malformed/duplicate/over-bound inputs fail before stdin,
authentication, or provider access, and hand-written notation is never parsed
or deduplicated. Infrastructure fixtures pin the exact official To/reply
prefix order, byte-for-byte original body suffix, and the 65,535-byte expanded
UTF-8 boundary. Application probes require no extra GET without `--reply-to`;
with it they require exactly one same-context/same-binding GET followed by one
POST, and zero POSTs for provider 404, malformed or mismatched reply results,
preflight cancellation, unreadable body files, or an oversized expanded body.
For a reply send, missing, unreadable, empty, invalid-UTF-8, and raw oversized
body files must fail through infrastructure preflight with zero GET and POST
calls; the exact bytes are read once and carried in an opaque one-use handle.
An integrated local-server case returns the exact reply successfully, then
rejects the expanded body before HTTP and observes one GET and zero POSTs.
Another case sends the exact form and original manual-notation suffix after one
same-binding GET; a lost POST response remains unknown and non-retryable. Once
POST dispatch may have begun, the pre-existing
non-retryable unknown-outcome tests remain authoritative.

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
- The opt-in scheduled live smoke workflow is separate from these offline
  checks. When explicitly enabled and supplied a dedicated test-account PAT,
  it runs only `account show`, `account status`, and `rooms list` using a
  temporary selection profile. It suppresses all task stdout and stderr and
  records only fixed task names and success/failure. This is a narrow live
  read check, not a provider-wide compatibility test or a source of fixtures.
- Chatwork authentication-binding tests use synthetic tokens and two isolated
  clients to reject missing, stale, wrong-client, and cross-session bindings;
  no ordinary repository gate reads a developer credential or contacts live
  Chatwork.
- Chatwork secret-canary tests prove that the token reaches only the exact
  `x-chatworktoken` request header and never argv, output, errors, logs,
  snapshots, fixtures, or persistent configuration.
- Chatwork User-Agent tests prove that the lazy CLI factory passes `Version`
  into infrastructure, only `dev` or up to 128 ASCII bytes of release-compatible
  SemVer reaches `User-Agent: cwk/<version>`, invalid input falls back to
  `cwk/dev` without echo, and every task route (including `/me` and uploads)
  keeps the exact PAT binding with that header.
- Catalog pagination tests require an exact optional-input/top-level-string-output opaque cursor binding, typed empty-cursor completion, and JSON-only presentation for `paged` results, and forbid that binding for `complete` results. Renderer fixtures reject an omitted, null, or non-string cursor.
- Infrastructure tests fix protocol conversion and boundary failure.
- CLI tests fix routing, help, rendering, and exit behavior.
- Room-task deadline tests preserve positive decimal Unix seconds, convert
  strict whole-second offset-bearing RFC3339 to the same positive Unix second,
  and reject fractions, offset-free/unknown-offset timestamps, invalid dates or
  offsets, nonpositive values, and int64 overflow before authentication or
  provider calls. Equivalent offset spellings must produce identical request
  limits; existing `--limit-type` values and provider form fields remain exact.
- `messages resources` tests perform requested files reads before requested
  room-task reads, match typed same-room message references, retain source count,
  100-item limit, completeness, and unknown absence independently per source,
  preserve reusable canonical references, and return no partial result after a
  failure. Body notation cannot infer resource kind or message existence.
- Text-help navigation tests require one root entry per direct command or
  top-level namespace, catalog-relative ordering within the direct and
  namespace sections, exact namespace membership/counts, no leaf duplication at
  root, relative namespace listings with every leaf's full usage and required
  flags, equivalent bare/explicit/trailing namespace help, and unchanged
  unknown-command faults for non-namespace paths.
- Command-selection catalog tests keep exactly `help`, `doctor`, `version`, and
  `config` always-on; prove missing state exposes only that control plane while
  a saved allowlist keeps later additions off; and make
  root, namespace, exact, trailing, recovery, workflow, and routing views agree.
  They reject a visible consumer without a reachable producer and
  a visible recovery action whose command is hidden, without auto-enabling
  either dependency.
- First-run command-selection tests require root human help to explain that
  `config` is unset and why the reduced view lowers token use and selection
  mistakes. Root text help contains only the four control commands; known
  configurable execution and scoped-help paths return
  `command_selection_required` with exact `config` recovery and zero PAT or
  provider calls. With a saved valid profile, exact disabled canonical paths,
  catalog-declared aliases, and exact scoped/trailing help return non-retryable
  `command_disabled` with canonical-path identity and exact `config` recovery;
  read, create, and write handlers, PAT resolution, and provider ports receive
  zero calls. Exact bare catalog namespaces produce the same bounded namespace
  help as `help <namespace>` without provider calls; saved views show disabled
  names only, including disabled-only namespaces. Unknown and incomplete
  non-namespace paths remain `unknown_command`. Re-enabling restores the
  ordinary command contract. A non-TTY config recovery reports
  `interactive_terminal_required` and makes no write.
- Disabled-command help tests require root navigation to retain disabled-only
  namespaces and namespace help to list disabled canonical paths in a separate
  name-only `無効なコマンド` section. Disabled usage, summaries, examples, and
  workflows are absent; active namespace details and typo suggestions use only
  the active view. Unsafe/unavailable selection retains complete-catalog
  static-help fallback, and unconfigured state remains distinct from a saved
  empty selection.
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
  Windows, resolve-once support for an existing configuration-home symbolic
  link, rejection of broken aliases and symbolic-link owned targets, strict
  bounded JSON, stale paths, modes, special files, Unix rename/directory
  durability, Windows replace-existing behavior,
  pre-Enter q/EOF/context cancellation, and unchanged prior bytes after an
  invalid selection. Repair tests distinguish malformed content from unsafe or
  inaccessible storage, reject false empty root help, migrate formerly
  selectable `doctor` and `version` entries without reintroducing them as
  choices, and preserve confirmed success under late cancellation. A saved-profile exact disabled invocation must be `command_disabled` with
  zero handler, PAT-resolution, or provider calls; exact namespaces still enter
  their active-view help, and unknown/incomplete non-namespace paths remain
  `unknown_command`. Re-enabling must not bypass the original
  authentication or mutation confirmation contract.
- Command-selection reconciliation tests add one bounded `command-selection`
  check to always-on `doctor`. Synthetic missing, saved, stale, legacy,
  malformed, unsafe, and unavailable states fix its state/source,
  enabled/disabled/stale/legacy counts and deterministic ordered SHA-256
  fingerprint. An uncertain save records expected `source=saved` and its
  candidate fingerprint; tests require both to match the subsequent doctor
  result and reject an unconfigured empty-selection fingerprint, so an
  uncertain write is inspected without a second mutation or false success.
  The internal catalog failure contract validates the exact dynamic
  error-message grammar, and runtime JSON/text fixtures must match it
  byte-for-byte around the fingerprint. Missing
  state reports `state=unconfigured source=missing` with zero enabled paths.
- Config success-output tests fix the two-line natural-Japanese
  enabled/disabled/change transcript, require cleanup prose only for a nonzero
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
- Exact text-help tests derive flag name, type, required, required-one-of, or
  optional state, the complete one-line description, combination constraints,
  and at most two examples from catalog inputs. They prove each rendered input
  line ends with its full catalog description, remove optional examples from
  the end when needed, and enforce a 2,048-byte encoded ceiling for every command,
  including the multi-sender message selection contract and the inclusive
  1..100 start-index/count bounds, plus inclusive-since/exclusive-until and
  fixed-Tokyo `YYYY-MM-DD|today|yesterday` day selection.
- Full-catalog and active-view validation reject a command whose complete
  no-example exact-help projection cannot fit the 2,048-byte ceiling, including
  when filtering changes which catalog-backed recipes are visible.
- Catalog hostile-input tests reject invalid UTF-8, terminal controls, Unicode
  format controls, line separators, and whitespace in non-argv input names
  before exact help can render them.
- Root help tests derive up to three distinct active leaf outcomes per
  namespace, de-duplicate summaries, count remaining outcomes, restrict exact
  help routes to catalog paths, gate the link-discovery guide on active messages
  resolve-link and the separate show route on active messages show, keep disabled
  outcomes hidden and disabled-only entries name-only, and
  report normal/static-fallback UTF-8 byte lengths without imposing a root-wide
  hard limit. Catalog tests reject missing, non-terminal-safe, multiline, or
  over-192-byte summaries; synthetic English summaries remain valid.
- Help shape and size tests keep root discovery index-only, require namespace
  help to expose each visible leaf's complete usage, required flags, and
  catalog-declared required one-of input groups, and
  restrict exact help to the reviewed invocation facts. They reject public
  contract dumps, enforce the exact 2,048-byte ceiling, and require
  `help ... --format agent` to fail with `--help` migration guidance.
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
- Local message-link discovery tests pin the exact 95-byte-max ASCII grammar,
  malicious and malformed rejection without raw-input echo, no auth or network,
  canonical room/message reference production and act-input round trip, all
  command-selection states, active-view help routes, resolver-only guidance to
  ask a person to enable show in root help, marked static fallback projection
  without asserting selection, and absence of implicit selection expansion. Synthetic command tests are not an agent transcript.
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
  render/audit-before-write ordering. They also fix the stable-only manual
  recovery trigger, quoted tag-to-revision binding, read-only recovery job,
  exact non-draft six-file Release metadata, checksum verification, and
  data-only handoff into the same Formula jobs. Negative workflow mutations prove that
  broader scope, another token/path/base, wildcard staging, an extra permission
  or token consumer, duplicate secret use, ambient runtime injection, symbolic
  destination acceptance, changed Formula-source binding, or ignored audit/PR
  failure fails. The App's external installation maximum remains a named
  manual release-owner review.
- Shared semantic fixtures and answer keys fix relationship, identity, bounds, coverage, uncertainty, and hostile-text facts independently of presentation. The active message-adjacency fixture also covers an exact quote author/whole-timestamp match as one unresolved, source-window-scoped candidate with an unproven identity.
- Quote-candidate tests require exact same-room author and whole timestamp matches, provider-order preservation, unique versus multiple states, missing timestamp, complete no-match, unavailable/partial unknown behavior, unknown relation-set rejection, and canonical message references. They prove candidates are computed before filtering/delivery, hidden targets lose only their optional sequence, visible source sequences are validated, cloning preserves provenance, and exact-only show makes no recent call while replies/around reuse their existing read. Overflow fails before filtering; no quote creates a reply edge or extra exact read.
- Relationship tests prove that To, quote, time proximity, display names, and layout-looking content do not fabricate reply edges; every valid multiple `[rp]` edge remains ordered, resolves independently, and renders as one scalar or a compact list without becoming unknown. Quote candidates use exact same-room author and whole-timestamp evidence, remain unresolved even when unique, preserve candidate order/refs through selection and byte delivery, and never trigger an extra provider call.
- Room-name candidate tests require one ordinary rooms provider read, Unicode
  simple-case-folded rune-window substring matching, provider-order all-match
  output, query/source/candidate/completeness metadata, and canonical decimal
  room references. They cover zero/one/many matches, Japanese and mixed case,
  no normalization/trim/ranking/multi-rune expansion, zero query leakage, and
  a no-`grep` flow into an exact action. Negative action tests reject names,
  leading-zero decimals, `rid...`, kind-prefixed text, and browser URLs before
  authentication/I/O. Member-name tests remain case-sensitive.
- Bounded message-selection tests prove sender OR and half-open period
  membership precede typed-time ranking,
  later provider position breaks equal-time ties, one-based start index and
  maximum count choose primary anchors, reply context follows that selection,
  and rendered records retain provider order and source sequences. They
  distinguish candidate count, applied start index, requested count, actual
  items per page, and next start index from provider `source-limit`, allow
  omitted context to default to replies for active selections, explicit none to
  opt out, context without a primary selection to name the offending flag and
  alternatives before authentication/I/O, context to exceed the requested count, reject invalid values before
  authentication/I/O, reject a source above declared
  coverage before selection, and keep local policy out of the one documented
  provider request. Fixed-clock tests prove Tokyo `today`/`yesterday`, exact
  effective day/bounds, and pre-I/O rejection of offset-free, fractional,
  conflicting, empty, or reversed periods. Context may cross the primary period
  only through an explicit direct reply edge while anchors remain distinct.
  `force` request. CLI/runtime tests additionally prove omitted and explicit
  `recent` emit the latest-window request while explicit `changes` retains the
  differential request and output.
- Exact-message reply-context tests prove omitted/`none` performs one exact
  request, while explicit `replies` performs that exact request followed by one
  same-binding force-recent list request whose local context and relation
  budget fields are cleared. Fixtures cover zero and multiple children,
  provider-order source sequences, an anchor also present in the source,
  parent-chain and child coexistence, `--resolve-relations 0`, incomplete and
  partially restricted source evidence, and strict typed same-room reply-edge
  matching. Source-window relation state is reset and re-resolved against the
  visible anchor/children so hidden targets cannot be labeled resolved without
  separate parent evidence. Body/To/quote/name/time canaries must not create a child. Domain
  negatives reject wrong anchor/room/target, unresolved edges, invalid source
  sequences, and over-bound coverage; capsule tests require anchor-first output
  with separate reply-context and canonical reply-child records.
- Byte-bounded message-delivery tests validate `--max-bytes` as a
  CLI/presentation-only integer from 1 byte through 16 MiB and reject duplicate,
  malformed, zero, negative, and over-bound values before authentication/I/O.
  Request spies prove the option enters neither the domain request nor provider
  query, and that the invocation makes the same one list request and relation
  reads as its unbudgeted twin. Boundary fixtures choose the longest
  newest-first anchor prefix while preserving provider-order output, whole
  messages, direct reply context, and reachable relation closure as atomic
  units. They cover shared context, UTF-8, exact byte boundaries, final newline,
  and no split body/line/relation record. Metadata assertions fix exactly
  `max-bytes`, `output-complete`, delivered-anchor `output-items`,
  always-present eligible-anchor `output-total-items`, and conditional
  `output-next-start-index` plus remaining-slice `output-next-count`; require
  that delivery is the absolute final line;
  and count its terminating newline inside the hard budget. Continuation tests
  prove the next index is the next newest-first anchor rank with no repeated or
  skipped anchor. Budgeted relation fixtures require
  `relation-resolution targets` to equal the emitted relation target count
  while `fetch-attempts` retains the complete invocation's already executed
  total. Provider `complete` and selection `next-start-index` must remain
  unchanged. A budget
  below the fixed envelope or first atomic unit returns
  `output_budget_too_small` with zero stdout bytes. Control goldens prove that
  omitting the flag remains byte-for-byte identical and emits no delivery
  metadata.
- Per-message body-limit tests validate `--max-body-bytes` as a CLI-only strict
  ASCII decimal value from 0 through 65,535 and reject signs, malformed input,
  overflow, and out-of-range values before authentication/provider I/O. The
  typed request and provider call remain unchanged. Renderer and semantic
  fixtures cover Japanese and emoji boundaries, limits below one rune, zero,
  original raw-byte lengths, escaped backslash/control/bidi text, preserved
  contains matches, canonical refs, reply edges, source order, selected
  context, and emitted relation-context. The cap affects each actually emitted
  message body only after typed selection and relation work; source, selection,
  and output-delivery completeness remain unchanged. Combined `--max-bytes`
  tests measure escaped output after body projection, including the final
  delivery line and newline. Omitted-option goldens remain byte-for-byte stable.
- Optional JST-display tests cover fixed-offset RFC 3339 output independent of
  the process timezone, exact inclusive/exclusive bounds for local years 0000
  and 9999, adjacent out-of-range seconds, and both `int64` extremes before
  conversion. List and show fixtures cover primary rows, reply/around rows, and
  emitted relation context; quote candidates and reachability boundaries gain
  no timestamp. CLI spies prove `--jst` is removed before typed request and
  provider construction, adds no call, and remains active through combined
  `--max-body-bytes`/`--max-bytes` redraw. Omitted-option goldens stay
  byte-for-byte unchanged, and catalog/active-view help remains within 2,048
  encoded bytes.
- Deprecated-alias tests prove `messages list --limit` and `--count` bind the
  same typed input and output, the alias succeeds with the exact stderr
  migration notice for at least two minor releases, and conflicting spellings
  fail before I/O. Unknown-flag tests require a deterministic catalog-derived
  did-you-mean hint and prove no suggested argv is rewritten or executed.
- Command-alias tests derive every finite spelling from `CommandSpec`, prove
  canonical handler/intent identity, post-active-view normalization with no
  hidden canonical-path disclosure, selector aliases in space/dot forms,
  redundant help-flag normalization, active-view fail-closed behavior, and
  catalog rejection of duplicate/path/boundary and alias-to-alias prefix
  collisions. Routing tests also
  cover dotted selectors, help flags in every post-command position, zero
  authentication/I/O, and full-catalog static help under only the declared
  invalid/unsafe/unavailable selection faults. Root and namespace fallback
  assertions require the static-catalog/`doctor` notice, while exact help keeps
  its bounded invocation-only bytes and ordinary tasks retain the load fault.
- Routing-suggestion tests reject mistyped leading root options before command
  selection or authentication, limit their hints to the fixed global option
  set, and require `help --hlep` to name `cwk --help`. Unknown-command tests
  require deterministic canonical-path guidance from only the active catalog,
  and zero handler execution or disabled/full-catalog path disclosure. The
  nearest candidate is suggested only when its declared effect is `read`; if
  that nearest candidate is `create`, `write`, or `unknown`, no suggestion is
  shown and a farther read is not promoted.
- No-post-processing agent transcripts fail if a supported task requires `jq`, `grep`, a custom join, raw notation parsing, source inspection, or an exploratory provider call.
- Presentation competitions pin fixtures, agent/model versions, prompts, repetitions, invocation budgets, answer scoring, token accounting, and latency measurement before candidate implementation.
- Candidate reports retain per-worktree correctness, next-action/reference, token, tool-step, byte, latency, reviewability, maintenance, benchmark-defect, and audit evidence. A selected presentation receives golden and compatibility tests only after an explicit compatibility decision; an inconclusive benchmark is never relabeled as a win.
- The frozen Competition 1 runner/scorer is not part of the ongoing gate.
  Current `tools/presentationeval` tests retain only active semantic fixtures;
  the explicitly preserved raw reports remain immutable historical evidence,
  and a future competition defines its own pinned runner in its work packet.

A global coverage percentage is not a substitute for these contracts. Add tests at the boundary where a future regression would otherwise pass unnoticed.

## Claims-to-checks discipline

Every strong statement should identify its enforcement path.

| Claim type | Preferred enforcement |
|---|---|
| Layer dependency | Go-aware architecture lint and import-boundary tests |
| Finite domain state | Types, constructors, and table-driven negative tests |
| Catalog completeness | Whole-catalog contract tests |
| Discover-to-act composition | Reachable reference-graph validation, required act references, and byte-preserving round-trip tests, including `rooms list --query` candidates passed unchanged as positive decimal `<room-id>` values while names/URLs/`rid...`/kind-prefixed values are rejected, and positional `files list` file/room values passed unchanged to `files show` while `absent` is rejected as identity |
| Command-bound singleton action | Fixed-target catalog kind/ID/scope validation, no-reference exclusivity tests, and matching mutation-target tests |
| Side-effect ordering | Fake adapter counters and failure-before-I/O tests |
| Mutation outcome classification | Structured-fault-first/cause-stripping tests, non-retryable unclassified outcome fallback and post-`Do` cancellation outcomes, infrastructure-minted pre-`Do` cancellation with zero transport calls and narrowly retryable recovery, valid success preserved across late cancellation, and read-only recovery validation |
| Chatwork error guidance | Transport and service tests distinguish typed read deadlines from caller cancellation by fixed Japanese message while preserving `operation_canceled` kind/code/exit; ordinary I/O remains `chatwork_unavailable`; mutation taxonomy is unchanged; 403 membership wording is fixed and limited to room targets/scopes; confirmation faults reveal only the exact catalog `--confirm` value, retain `mutation_rejected` recovery, and make zero auth/provider calls |
| Authentication precondition | Secret-free session contract, zero-downstream-call tests, and catalog validation of every standard gate fault's code/kind/retryability |
| Authentication binding | Opaque JSON-excluded/fmt-redacted binding type, infrastructure-only issuance lint, exact pass-through tests, and two-client/stale-binding adapter fixtures |
| Sole Chatwork PAT input | CLI/composition tests requiring `CWK_API_TOKEN`, PAT-only requirement snapshots, obsolete-selector non-effect, and zero-call missing/invalid-token failures |
| Chatwork client identification | Strict version validator and safe fallback, lazy CLI version-passage test, and all-task local-server fixtures checking the exact User-Agent with the authenticated binding |
| PAT process-local binding | Synthetic two-client/stale-binding fixtures, exact unchanged binding pass-through, fixed-destination header tests, and no persistent credential source |
| Authentication secret exclusion | Token-canary scans across argv rejection, stdout, stderr, structured faults, logs, snapshots, fixtures, test diagnostics, and repository state |
| Pagination completeness | Cursor loop/budget/cancellation tests, retryability/catalog agreement, and no-partial-result assertion |
| Public paged continuation | Catalog validation of one exact same-kind optional input/top-level output binding, JSON-only runtime presentation, and internal reference-workflow validation |
| Retry safety | Timeout/attempt/idempotency validation and adapter contract tests |
| Room-task deadline input | Strict CLI parser and exact-help tests; legacy positive Unix seconds stay unchanged, equivalent whole-second RFC3339 offsets map to one Unix second, and invalid/fractional/offset-free/unknown-offset/nonpositive/overflow inputs make zero auth/provider calls while preserving the provider integer and `limit_type` form contract |
| Message resource lookup | A fixed expected-command-sequence contract test passes fixture canonical refs unchanged through `chatworkcmd.Service` and the real capsule renderer; it does not run the CLI or observe an agent transcript, so its zero-external-processing budget is not a measurement. The separate `httptest` integration proves exact unfiltered read requests, DTO `message_id` retention, shared caller context/authentication binding, one attempt per source, files-before-room-tasks order, per-source 100-item/incomplete/unknown-absence metadata, 403/429 all-or-error with no partial output, zero writes, and no raw-notation inference. A qualifying agent-readiness transcript requires the production CLI fixture route and a bounded Codex runner to record discovery, command calls, external tools, and writes. |
| Chatwork message truthfulness | Status/header matrix, private-summary canary, normal-zero/partial/all/restricted/not-found distinctions, relation unknown/absent tests, and hostile escaped-body list continuity |
| Chatwork room identity and invite replacement | Same-binding `/me` exact-match tests with zero-POST failures; owner-free room form; pre-auth code/XOR/full-field validation; exact PUT form and regeneration omission tests |
| Typed message delivery | Ordered unique `--to` refs bounded at 100; exact optional same-room reply target; preflight refs separate from room mutation scope; same-context/same-binding GET-before-POST with no GET when absent; exact official notation prefix plus original-body byte suffix; invalid body-file source proves zero GET/POST, invalid preflight/final expansion proves zero POST, including GET-success/body-overflow integration; exact-form success and lost-POST-response unknown-outcome fixtures |
| Read-only send outcome reconciliation | Exact `messages send` recovery path and active-profile closure; literal/file body and typed-prefix equality through the common byte builder; body-file validation before reads; at most one exact reply GET, one current-account GET and one latest 100-message read, same context/binding, zero POSTs; the existing request/list result still crosses the app boundary, but preparation returns no file bytes and matching returns only booleans with temporary list-body references dropped; always-present account_ref including zero candidates; every sender_ref equals it; only canonical exact matches by the current account; identity failures stop before list; other-author decoys are excluded; outcome always unknown; zero/one/many, Unicode/line-ending byte, hostile author, restricted/empty/out-of-window source, invalid-input zero-provider-call, request/client/binding identity, one-use/zeroize, body/path-free result/fault/log, and active no-post-processing readiness fixtures |
| Chatwork rate-limit evidence | Strict header/body parsing tests, five-minute plausibility bounds, read/mutation catalog signatures, advisory-timing validation, and text/JSON unknown-timing snapshots |
| Agent recovery | Catalog fault declarations, exact-path/help-selector executable grammar tests, and structured error snapshots |
| Hierarchical text discovery | Catalog-derived direct-command/namespace partition, unique section-relative ordering and namespace counts, selector round-trip, bounded exact root outcome routes, namespace leaf usage/required-flag projection, trailing-help equivalence, complete input-description equality, optional-example reduction, exact 2,048-byte/content/example limits, retired-agent-format rejection, and hostile input-name rejection tests |
| User-selected command dispatch scope | Complete `DefaultCatalog` contract lint plus configurable-leaf metadata, an exact-path ordered active scope, exactly four always-on commands (`help`, `doctor`, `version`, `config`), control-only first-run dispatch and `command_selection_required` recovery with zero PAT/provider calls, saved-disabled canonical/alias/exact-help `command_disabled` with zero handler/PAT/provider calls, unknown/incomplete-path distinction, disabled-only root namespace and name-only namespace listing without disabled usage/examples/workflows, active-only typo suggestions, unsafe static-help fallback, actionable required-reference/recovery closure validation, the single TTY selector's textual effect badges and key-state tests, Enter-only persistence, natural-Japanese enabled/disabled/change output, all-exit terminal restoration, context-responsive platform reads with no abandoned input consumer, typed non-TTY failure, invalid-view retention, legacy local-command migration, resolve-once configuration-home alias support plus strict owned-target platform storage with Unix durability and explicit Windows limits, uncertain-fault/doctor count and fingerprint reconciliation, and re-enable tests that retain independent authentication, reference, provider-permission, and mutation-confirmation contracts |
| Bounded agent discovery | Root and namespace text-help journeys, namespace-complete leaf usages and required flags, exact-help complete descriptions and 2,048-byte/two-example ceilings, oversized no-example catalog rejection, forbidden full-contract canaries, explicit retired-format failure, runtime structured recovery, and active-view consistency tests |
| Room-name candidate discovery | Domain/application/CLI/presentation tests require one complete rooms provider read, Unicode simple-case-folded substring filtering with no normalization/trim/ranking/multi-rune expansion or automatic selection, explicit query/source/candidate/completeness metadata, provider-order canonical refs, zero query leakage, and a no-`grep` flow into an exact decimal room action |
| Member-name candidate discovery | Domain/application/CLI/presentation and active synthetic tests require one complete room-members provider read, exact case-sensitive substring filtering with no normalization or automatic selection, explicit query/source/candidate/completeness metadata, stable canonical refs, escaped external names, zero query leakage, and a two-command `members find` to exact `messages list --sender` flow with no full-message pre-dump or external processing |
| External text structure | Visible-projection unit/E2E tests plus scoped I/O trust metadata; printable meaning remains explicitly out of scope |
| Agent command certainty | Root/namespace text-help to executable-usage transcripts with no full-contract preloading, command probing, external help scraping, or undeclared recovery call; disabled names have a separate name-only help projection and exact disabled calls fail as `command_disabled` with zero handler/PAT/provider calls; typo-path guidance remains active-only and suggests only a nearest `read` candidate |
| Supported outcome completeness | Transcript assertion of zero external post-processing and declared provider/context coverage |
| Context relationship truth | Presentation-independent typed fixtures and negative inference tests for To, quote, names, proximity, and missing references |
| Presentation eligibility | Shared semantic answer key, canonical-reference/coverage/trust checks, determinism, and zero external post-processing |
| Presentation selection | Parallel-worktree comparison with pinned agent tasks, model/tool versions, repetitions, token accounting, latency, and raw per-candidate results |
| Presentation decision provenance | Retained raw runs, score summaries, audit findings, and benchmark-defect records that distinguish an experiment result from a later owner compatibility decision |
| Current success text | All-route and golden tests require the headerless task projection. Seven reviewed homogeneous collections require exactly one trust/schema prelude even when empty and one provider-order physical line per item with stable canonical positions and optional suffixes; `files list` fixes `message_ref` as canonical-or-`absent`. `messages list` additionally requires one room/trust/fixed-schema header with the provider bound named `source-limit`, a document-local actor dictionary, positional canonical message/time/body values without repeated labels, flat provider-order adjacency records, reachability fields, and separate relation-resolution/context/gap records. Tests preserve migration history without claiming a Competition 1 winner |
| Subtractive task projection | Catalog/result field checks and negative canaries allow only declared task facts, exact canonical references, task-relevant bounds/completeness/uncertainty, and external-text trust framing. Message actor aliases are allowed only as document-local compression with canonical dictionary entries; semantic raw-notation records, wire extras, derived thread metadata, and non-contract defaults fail |
| Bounded message selection | Domain/application truth tables, adapter-request guards, scoped-help/runtime tests, fixed clocks, and active synthetic agent scenarios require omitted or explicit `recent` to select the latest bounded window, explicit `changes` to select differential retrieval, exact sender OR inputs with catalog-declared repeatability and a 100-reference bound; optional inclusive whole-second offset-bearing RFC3339 `since`, exclusive `until`, or fixed-Tokyo exact/`today`/`yesterday` day; optional one-based `--start-index` and maximum `--count` 1..100; catalog-declared deprecated `--limit` alias equivalence plus stderr migration warning and conflict rejection; sender-and-period predicate then typed-send-time ranking with later-position tie-break then index/count selection then direct typed in-window reply expansion, with omitted context defaulting to `replies` and explicit `none` opting out; context allowed beyond requested count/period with explicit anchors; context without a primary selection naming the offending input and alternatives before I/O; provider-order gapped source sequences; source/effective-period/candidate/start/requested-count/items-per-page/next-start and anchor/context distinction; one documented `force` list request with no provider pagination; pre-I/O invalid-input and over-bound-source rejection; canonical-reference reuse; and zero rendered-text filtering or raw-notation inference |
| Byte-bounded message delivery | CLI-only 1..16,777,216-byte parser and request-leakage tests; same-call-count spies; longest newest-first anchor-prefix fixtures; atomic whole-message/direct-reply/reachable-relation closure checks; provider-order rendering; exact delivery keys including always-present `output-total-items` and count-bounded remaining `output-next-count`; absolute-final-line, continuation-rank/range, projected-relation-target, and original-fetch-attempt assertions; unchanged provider `complete` and selection `next-start-index`; UTF-8/exact-boundary/final-newline coverage; typed zero-stdout too-small failures; and omitted-option byte-for-byte goldens |
| Per-message body delivery | Strict pre-auth 0..65,535 ASCII decimal validation; request/call neutrality; raw UTF-8 longest complete-rune prefixes before visible escaping; per-record truncation and original-byte facts; Japanese/emoji, zero, below-rune, hostile control/backslash/bidi, contains-before-truncation, canonical relation/context, emitted relation-context, and unchanged completeness fixtures; combined escaped `--max-bytes` accounting; and omitted-option byte-for-byte goldens |
| Optional JST timestamps | Fixed `+09:00` without timezone lookup; four-digit local-year bounds checked before conversion including adjacent seconds and `int64` extremes; ambient `TZ` independence; list/show primary and all emitted typed context rows; no timestamps on quote candidates or reachability boundaries; Unix-second preservation; stripped CLI flag, no provider request/call changes, active help limit, combined budget redraw, and omitted-option byte-for-byte goldens |
| Unknown input guidance | Deterministic selected-command flag suggestions; leading-root-option suggestions before selection/authentication from only `--error-format`/`--help`/`--version`; `help --hlep` guidance to `cwk --help`; active-catalog canonical command-path suggestions restricted to a nearest `read` candidate without promoting farther reads; exact `messages search` bounded `--contains` guidance only while `messages list` is active; `help tasks` selecting only active `room-tasks`/`personal-tasks` namespaces (one exact namespace, two root help, none generic); `--around`/`--message`, `members list --query`, and `--format` task guidance with visibility/format rules; state-precedence, hostile-value, and doctor JSON preservation tests; alias-prefix collision validation; and negative tests proving no disabled-path suggestion, argv rewrite, target choice, handler execution, authentication, or I/O |
| Bounded message relation closure | Domain/application/CLI and active T1 tests require `--resolve-relations` 0..100 with public default five; breadth-first recursive traversal of every same-room reply edge; source reuse, duplicate/cycle suppression, exact room/message result binding, one attempt per unique missing target, limit/attempt/provenance/gap evidence, retained not-found/restricted outcomes, whole-task abort for other faults, and no To/quote/body/cross-room expansion |
| Exact-message direct replies and around | Domain/application/CLI and active semantic tests require exact-anchor-first execution; `--around 0..100` alone enables one same-binding bounded recent read and shares exactly one read with `--context replies`; explicit `none` remains valid and omission behavior is unchanged; around neighbors exist only when the exact anchor occurs in the returned source and are selected by original provider sequence, with typed in-window/outside-window/unknown anchor state, requested/actual side counts, positions, source count/limit/completeness/access; direct children still require a typed same-room edge to the anchor; overlap is deduplicated without removing the explicit edge; absent/restricted anchors produce no neighbors; zero request count still reads the source; around does not consume the outgoing-parent budget; invalid values fail before authentication; and there is no body/To/quote/name/time/ID inference or external post-processing |
| Message reachability | Domain truth tables and active T2 tests require an oldest boundary only for nonempty unrestricted recent sources, within/partial/outside period classification against that boundary, unknown for changes/empty/limited/unprovable sources, one list invocation, and no false empty-day or complete-history claim |
| Token efficiency | Pareto comparison among quality-eligible candidates followed by a selected-format non-regression budget |
| Public capability coverage | Exact bidirectional match between capability ledger and catalog `CapabilityID` values |
| Fixed Chatwork API coverage | Strict 32-operation snapshot plus bidirectional operation-to-public-capability validation |
| Fixed Chatwork resource bounds | Exact typed snapshot values plus transport, provider source-cardinality, aggregation, upload, and output boundary tests; local message selection cannot hide a response above declared coverage |
| Chatwork mutation confirmation | Exact operation-ID policy sets plus typed invoker/CLI zero-call tests |
| External schema compatibility | Vendored fixture, generator, and drift test |
| Secret or private-data exclusion | Repository policy, scanner, and synthetic fixtures; real temporary Git fixtures prove untracked Claude-local home paths, invalid Markdown links, and secret canaries stay outside the publishable scan, staged paths return to that scan and fail Claude policy plus ordinary checks, symlink/special-file rejection inside local paths, fail-closed index access, and preservation of other shape invariants |
| Reproducible generation | Regenerate and require a clean diff |
| Artifact integrity | Build, inspect, checksum, and install tests |
| Same-minor Go patch proposals | Offline updater tests cover numeric eligibility, hostile/malformed release data, bounded/network failures, and unchanged unrelated files; release lint fixes workflow jobs, steps, permissions, action revisions, inputs, candidate checks, and `go.mod`-only PR scope; negative fixtures reject extra run/action/env/with steps, permissions, runtime expressions, arbitrary revisions, and proposal-before-validation order; manual review confirms that pre-creation gates do not claim head-attached CI approval |
| Shared Homebrew tap publication | Release lint validates reviewed annotated-tag notes, exact-tag publication checkout, the workflow and Formula-job field allowlists, every Formula-job step start, and exact checksum/toolchain, render/audit/artifact upload, fresh-runner artifact validation, token, checkout, staging, and PR shapes; confines the App action and each secret to one reviewed workflow-wide occurrence; fixes the stable-only manual recovery input, quoted tag binding, read-only existing-Release verification, exact six-file/checksum handoff, audit-to-publish dependency, public tap repository, requested permissions, read-only audit-job source token, zero source-repository permissions in the token-bearing job, absence of ambient `env`/`defaults`, tagged-source checkout, or checked-out code execution there, exact audited Formula binding/path, non-symbolic destination paths, PR branch/title prefixes, pinned actions, and render/audit-before-write order; negative mutation fixtures prove generated-note substitution, unpinned/reclassified recovery, skipped metadata/checksum validation, alternate/extra fields or steps, external secret/action use, runtime injection, symbolic-path acceptance, same-runner code execution, representative drift, and ignored failures are rejected; the release owner manually confirms the external App installation maximum |
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

The Go patch updater reads only the fixed official release index
`https://go.dev/dl/?mode=json`, validates bounded JSON, and selects only a
newer stable patch in the major/minor line already selected by `go.mod`. Its
offline unit tests cover equal, older, newer, and different-minor versions;
malformed, duplicate, unknown-field, oversized, and trailing JSON; bad
responses, timeout, and network failure; and proof that unrelated files stay
unchanged. The scheduled workflow validates a candidate against one immutable
default-branch SHA under the exact candidate toolchain, then requires the
`full`, `security`, and `release` profiles before its separate PR-write job.
The release profile runs the workflow contract linter and negative fixtures,
which fix the job/step order, permissions, action revisions, inputs, runtime
environment, candidate checks, and `go.mod`-only staging; added steps,
untrusted run/action/env/with inputs, arbitrary SHA-shaped revisions, and
write-before-validation changes must fail.

The workflow's `GITHUB_TOKEN` can cause GitHub to hold pull-request CI for
approval. Candidate validation is evidence about the generated tree, not an
attached PR status check. The harness does not claim that this preflight
satisfies branch protection that requires a check on the PR head.

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
