# Work Context: Persistent command selection

## Verified pre-change facts

- `DefaultCatalog()` is the complete source for dispatch, human help, agent
  help, recovery actions, reference workflows, contract lint, and API coverage.
- `CLI.RunContext` validates and matches that catalog before lazily resolving
  `CWK_API_TOKEN`; an active catalog view can therefore reject hidden commands
  with zero authentication or provider calls.
- Full `Catalog.Validate` requires every produced reference kind to have a
  consumer. A selected view needs a narrower validator: consumers still need
  reachable producers, but a visible terminal producer may have no consumer.
- Catalog validation also resolves every declared recovery command and requires
  uncertain mutation outcomes to reconcile through a read-only command.
- Exact `config` cannot coexist with `config show` or `config edit` because
  command paths may not collide with namespace prefixes. Namespaces are help
  selectors, not implicit executable defaults.
- Fixed `tool_local` singleton mutations are already a supported catalog and
  `execution.Invoker` contract.
- The retired OAuth implementation used strict bounded JSON, XDG-on-macOS,
  Windows AppData, restricted Unix modes, symlink checks, and platform-specific
  replacement.
  ADR 0003 explicitly says the current binary ignores its `cwk/config.json`.

## Fixed decisions

- The complete catalog stays intact. Runtime derives one active view from it;
  no second command registry is introduced.
- `CommandSpec` declares whether a leaf is configurable. `help`, `config show`,
  and `config edit` are always-on. `doctor`, `version`, and every Chatwork leaf
  are configurable.
- The persisted schema is an exact-path allowlist of configurable leaves:

  ```json
  {"schema_version":1,"enabled_commands":["rooms list","messages list"]}
  ```

- Missing file means all current configurable commands enabled. A present file
  is authoritative, so later catalog additions remain off. Canonical unknown
  paths are retained as stale upgrade evidence and ignored by the active view.
- Hidden invocation uses the ordinary `unknown_command` fault. The filter is
  not designed to conceal product existence as a security property.
- `config show` is the read-only reconciliation task for `config edit`.
- The selector is line-oriented and dependency-free. Numbers are document-local
  aliases; only exact catalog paths are persisted. Prompts/transcript use stdout
  so structured stderr errors remain parseable. It accepts any injected stream;
  a TTY is not treated as evidence of human authority.
- The file is `${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json`
  on macOS/Linux and `%AppData%\\cwk\\command-selection.json` on Windows.
- Normal commands, root help, and `config show` fail closed on malformed,
  unsafe, or unavailable state. Config-scoped help remains reachable.
  `config edit` may recover malformed serialized content by starting from the
  documented all-enabled baseline, but unsafe objects/modes and unavailable
  paths require local repair followed by `config show`. No write occurs before
  the explicit `save` token.

## Constraints

- Active help must not leak disabled paths through scoped errors, next actions,
  workflows, namespace counts, or trailing-help normalization.
- The selector cannot auto-enable dependencies. It reports the exact missing
  producer or recovery edge and waits for another selection.
- Configuration is non-secret preference state and must never contain the PAT,
  provider data, personal data, or arbitrary copied command metadata.
- Enabling a command does not bypass PAT authentication, canonical references,
  confirmations, effects, or provider permissions.
- Existing test constructors must remain isolated from the developer's real
  XDG state; only production composition installs the file-backed service.

## Risks and mitigations

- **Self-lockout:** all configuration tasks and help are catalog-declared
  always-on; when state cannot be loaded, config-scoped help remains available
  without presenting a false normal root view.
- **Dead agent workflow:** save-time view validation requires visible recovery
  closure and reachable producers for every visible consumed reference.
- **Silent fail-open:** malformed, unsafe, or unreadable state is a typed
  failure. Only malformed serialized content enters the explicit repair screen,
  which still requires `save`.
- **Interrupted overwrite:** validate and write a same-directory restricted
  temporary file and recheck the target. Unix uses rename plus sync of the
  opened directory root. Windows requests replace-existing without claiming
  portable atomicity or durability. After replacement starts, raw failures are
  uncertain and route to `config show`.
- **Blocked Ctrl-C:** line reads race their result against `ctx.Done()`.
- **Late Ctrl-C:** a confirmed nil save result is emitted as success rather than
  reclassified as a retryable cancellation.
- **Misplaced trust:** UI and docs state `security-boundary=false`; local actors
  able to edit the setting can re-enable commands.

## External facts

None. This capability is local and does not alter the Chatwork API snapshot.

## Thesis evidence

- The user identified contact-request workflows as common irrelevant surface.
- Earlier root-help work removed duplicated leaves from the first human view;
  this request repeats the same need at the user-specific command-graph level.
- The governing consequence is therefore durable: a complete public catalog
  and a smaller active attention view can coexist only when both routing and
  every discovery projection consume the same derived view.

## Completion evidence

- The complete catalog contains three always-on control leaves and 35
  configurable leaves in the implemented snapshot. Disabling the three
  contact-request leaves leaves a 32-command active configurable view.
- Missing required-reference dependencies report the blocked command, exact
  input, reference kind, and every currently runnable exact producer candidate;
  hidden recovery edges report their exact command path.
- Malformed serialized content yields `command_selection_invalid` and can enter
  the explicit selector. Unsafe object/mode state yields
  `command_selection_unsafe`; unavailable paths retain
  `command_selection_unavailable`. The latter two do not enter the selector.
- Root help fails with the same typed selection fault instead of presenting a
  false empty catalog. `config --help` and config-scoped agent help remain
  available for diagnosis.
- Full `task check`, focused race tests, Windows amd64 cross-compilation, and
  three independent post-fix reviews completed successfully.
