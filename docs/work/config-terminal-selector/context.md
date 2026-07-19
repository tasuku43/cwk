# Work Context: Single terminal command selector

## Current behavior

- The complete catalog currently exposes `config show` as a read utility and
  `config edit` as a fixed-target write below a `config` namespace.
- `config edit` renders all choices once, then accepts line-oriented numbers,
  `all`, `none`, `save`, and `cancel`.
- `doctor` and `version` are currently configurable and therefore appear in
  persisted allowlists created by the released selector.
- The old `config show` output begins with
  `config purpose=attention-only security-boundary=false source=...`.
- The file adapter already owns strict bounded JSON, XDG-on-macOS/Linux,
  AppData-on-Windows, symlink/special-file refusal, and platform-specific safe
  replacement. Those mechanisms do not depend on the selector grammar.
- Active-view validation and routing already reject disabled commands before
  lazy PAT resolution or provider I/O.

## Relevant structure

- Entry point: `cmd/cwk/main.go` supplies process streams and signal context.
- Domain rule: `internal/domain/commandselection` validates bounded canonical
  exact command paths.
- Application use case: `internal/app/configcmd` owns the fixed tool-local
  mutation and persistence port.
- Infrastructure boundary: `internal/infra/commandconfig` owns the preference
  file. A terminal-mode adapter is required for portable raw-mode entry,
  restoration, terminal detection, and dimensions.
- CLI/catalog: `internal/cli/config.go`, `config_catalog.go`, `catalog.go`, and
  `cli.go` own interaction, presentation, active-view derivation, help, and
  routing.
- Existing tests: `config_command_test.go`, `catalog_active_view_test.go`, help
  contract tests, storage tests, and the command-attention readiness scenario.

## Fixed decisions

- Public topology becomes one exact `config` write command. Bare invocation
  opens the selector; `config --help` and `help config --format agent` expose
  its contract.
- `doctor` becomes the exact read-only reconciliation task. Its
  command-selection check reports source, state, enabled and disabled counts,
  and a deterministic fingerprint of the canonical enabled selection.
  Uncertain config mutation outcomes point to `doctor`; `help` is only command
  discovery and is never presented as state reconciliation.
- The TUI uses a bounded viewport and a catalog-order selection. It enters an
  alternate screen, hides the cursor, restores both plus terminal mode before
  crossing the save boundary, and emits only one final saved/unchanged record.
- Advertised keys are Up, Down, Space, Enter, and `q`. Ctrl-C produces the
  typed canceled result; input closure is an unchanged exit. Escape is accepted
  as a safe non-saving cancel but is not advertised in the compact footer.
- `q` and terminal EOF are deliberate unchanged exits. Ctrl-C/context
  cancellation retain the typed canceled result. Neither writes.
- Redirected or otherwise non-terminal stdin/stdout fails with the stable
  unavailable `interactive_terminal_required` fault; the selector does not
  silently fall back to the retired
  line grammar.
- `help`, `doctor`, `version`, and `config` are explicit always-on leaves. This
  is catalog metadata, not inference from role or effect.
- Failure to load or validate the active profile cannot block routing or exact
  help for those four leaves. It also cannot create a false normal Chatwork
  view: `doctor` exposes the load state, while `config` retains only the
  already-reviewed repair behavior appropriate to that failure class.
- Only `doctor` and `version` found in an older saved allowlist are accepted as
  legacy selectable entries. They are ignored for active-view construction and
  removed on the next successful Enter save. `help`, `config`, or any other
  always-on path in the allowlist remains invalid; unknown canonical paths
  retain the existing stale-path behavior.
- The screen does not repeat purpose/security/source metadata. Exact help and
  durable contracts retain the attention-only/non-authority statement.
- Every selectable command row carries an explicit text effect badge from its
  catalog declaration: `read`, `create`, or `write`. ANSI colors are secondary
  affordances: cyan for read, yellow for create, and magenta for write. Meaning
  never depends on color, and red is deliberately unused because `write` does
  not by itself mean destructive.
- `golang.org/x/term v0.36.0` and its pinned `golang.org/x/sys v0.37.0`
  dependency are the selected terminal primitives. Both modules declare Go
  1.24 and use BSD-3-Clause, which is compatible with this repository's Go
  1.26.5 contract. `x/term` supplies terminal detection, raw mode, restoration,
  and sizing. Direct, platform-scoped `x/sys` imports supply context-responsive
  descriptor waiting/reading on Unix plus synchronous Windows console reads,
  exact reader-thread cancellation, and VT-output mode management. Only the
  infrastructure terminal adapter imports either module, so the CLI third-party
  allowlist remains empty and no CLI-import ADR exception is required.

## Constraints

- Command paths cannot collide with word-boundary namespaces, so both old
  config leaves must be removed before exact `config` can enter the catalog.
- A confirmed save cannot be overwritten by later context cancellation.
- Enter first validates the selected active view. A validation failure remains
  in the selector and performs no restore or save. A valid selection then
  restores alternate-screen, cursor, and terminal state before persistence;
  restoration failure proves zero save attempts. Only after successful restore
  may one save cross the mutation boundary.
- External text does not enter this screen; catalog summaries are trusted
  static contract text. ANSI control remains CLI-owned structure.
- Width calculation and truncation operate on visible cells rather than ANSI
  byte length. A narrow terminal must preserve cursor, checkbox, full text
  effect badge, and command path before truncating optional summary text. The
  same state and dimensions produce byte-identical output.
- A valid selection fingerprint is `sha256:` plus lowercase SHA-256 hex over
  the version tag `cwk-command-selection/v1` followed by each enabled exact
  path in catalog order using length-prefixed UTF-8 parts. This distinguishes
  path boundaries and ordering while ignoring JSON whitespace. When state is
  unavailable and no canonical selection exists, doctor emits the literal
  `fingerprint=unavailable` instead of fabricating one.
- A fingerprint identifies only the canonical selection. Reconciliation of an
  uncertain save also requires the candidate's expected `source=saved`; the
  all-enabled missing-profile default can legitimately have the same
  fingerprint and must remain distinguishable as `source=default`.
- The uncertain fault's dynamic values follow one catalog-declared message
  grammar in text and JSON. This is an explicit public fallback because the
  current shared fault envelope has no typed command-specific detail object.
- An effect badge without the complete command path is not enough to identify
  a preference target. A frame that cannot display the current identity blocks
  movement, toggle, and save rather than permitting a hidden mutation. A key
  that first observes a resize-only frame becoming usable only redraws that
  identity and is consumed. When validation produces a notice, its complete
  escaped text and the exact identity must share one frame before mutation keys
  become active again; diagnostics are never truncated into an actionable
  frame.
- The terminal adapter may control only the injected stdin/stdout descriptors;
  it must not open another terminal or infer human authorization from TTY state.
- A canceled terminal read must leave no background reader able to consume a
  later invocation's input. Infrastructure therefore owns context-responsive
  descriptor waiting and reading rather than wrapping a blocking `io.Reader`
  call in an abandoned goroutine.
- Tests use synthetic key streams and fake terminal control; they never modify
  the developer's real XDG profile.

## External facts

- `golang.org/x/term v0.36.0` is the selected Go terminal package for portable
  terminal detection, raw-mode state/restore, and size helpers. Its license is
  BSD-3-Clause, its module declares Go 1.24, and it introduces no runtime
  network destination or subprocess.
- `golang.org/x/sys v0.37.0` is already the exact transitive requirement of
  `x/term v0.36.0` and is pinned directly because the adapter imports its
  platform packages. `x/sys/unix` provides `Poll` and `Read` for bounded,
  context-responsive terminal reads; `x/sys/windows` provides synchronous
  `ReadFile`, thread-handle duplication, `CancelSynchronousIo`, and console-mode
  operations for equivalent Windows input and VT output behavior. The reader
  stays on one locked OS thread, and cancellation always joins it before
  returning. The module is BSD-3-Clause, declares Go 1.24, and adds no network
  destination or subprocess.
- Go 1.24 is compatible with this repository's Go 1.26.5 toolchain contract.
  Darwin, Linux, and Windows builds must compile the concrete platform adapter
  rather than relying only on host-platform tests.

## Resolved implementation constraints

- [x] Use a narrow infrastructure-only `x/term` plus platform-scoped `x/sys`
  adapter across Unix and Windows; do not add a general TUI framework or CLI
  third-party allowlist.
- [x] Test terminal lifecycle through a fake controller and synthetic key
  stream, supplementing it with a bounded PTY observation and cross-builds.

## Thesis evidence

- Root help was already shortened because repeated leaf listings increased
  selection cost. The numbered config grammar repeats the same friction inside
  a local preference workflow.
- The owner explicitly selected cursor/Space/Enter interaction and removed the
  show/edit distinction and repeated metadata header.
- The durable consequence is that the complete catalog and active view remain
  semantic contracts, while the one human selector may be stateful and
  terminal-native.

## Security and public-boundary notes

- Assets: non-secret exact-path allowlist and terminal mode/screen state.
- Credentials/provider data: none; PAT behavior is unchanged.
- Side effect: one existing local profile replacement, only after Enter.
- Dependencies: pinned `golang.org/x/term v0.36.0` and
  `golang.org/x/sys v0.37.0`, both BSD-3-Clause, infrastructure-only, Go 1.24
  minimum, and Go 1.26.5 compatible. The latter was already `x/term`'s exact
  transitive requirement and becomes direct only for the platform terminal
  operations above.
- Cancellation: validation and terminal/session cleanup precede mutation;
  post-action unknown outcomes reconcile through read-only `doctor`, never
  `help` or a removed config leaf.

## Verification evidence

- A user report reproduced in a real PTY: ASCII `0x20` toggled immediately,
  while U+3000 full-width space produced no frame because the byte parser
  discarded it. The corrected parser accepts the three UTF-8 bytes across
  arbitrary read boundaries and treats unrelated UTF-8 as ignored input.
- A second visual review found that variable badge lengths shifted the exact
  command path by up to two columns. The corrected renderer pads `[read]` and
  `[write]` to `[create]` width before the single path separator.
- Focused tests pass for `internal/app/configcmd`, `internal/infra/terminalui`,
  `internal/cli`, and `tools/archivepack`; focused `go vet` passes for the same
  implementation surfaces.
- Race tests pass for the terminal adapter and CLI, including blocked-read
  cancellation and the resize/validation state transitions.
- The command and terminal adapter cross-build for Darwin arm64, Linux amd64,
  and Windows amd64. Windows terminal tests compile and Windows CLI/terminal
  vet passes.
- A bounded PTY replay observed the colored effect rows, moved the cursor,
  toggled one command, saved it, and verified terminal restoration. Separate
  PTY replays covered unchanged `q` and Ctrl-C cancellation.
- `task release:check` and `task public:check` pass. The release archive gate
  reopens the generated tar/zip and verifies canonical entry order, metadata,
  modes, bytes, and symlink absence for `LICENSE` (0644),
  `THIRD_PARTY_NOTICES` (0644), and the canonical executable (0755).
- Release checks compare the linked production-module union for all five
  release targets with the reviewed dependency manifest. The notice contains
  the exact project license, Go 1.26.5 LICENSE/PATENTS, and the pinned x/term
  and x/sys license texts; the Homebrew formula installs both legal documents.
- Three independent final audits cover implementation correctness,
  catalog/presentation contracts, and release/legal packaging. Their reported
  P1/P2 sets are empty after the reviewed fixes.
- `git diff --check` passes.

## Final gate evidence

The managed sandbox could not complete the combined gate because its test suite
opens loopback listeners through `httptest.NewServer` and its vulnerability
step reads the official Go vulnerability database. After explicit approval, an
isolated clean clone of commit `069cf20` completed `task check`: fast contracts,
all package tests, race tests, module verification, repository security checks,
`govulncheck`, release lint, public guard, and contract lint all passed. The
shared workspace's unrelated concurrent message-window changes were excluded
from that clone and remain untouched.
