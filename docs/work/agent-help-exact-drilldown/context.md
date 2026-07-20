# Work Context: Safe exact-command agent-help drilldown

## Verified current behavior

- Installed `cwk v0.1.1` root agent help is 7,901 bytes.
- Installed `cwk v0.1.1` `messages` namespace agent help is 201,042 bytes and
  embeds every selected command's complete contract plus workflows.
- Installed `cwk v0.1.1` exact `messages list` agent help is 55,458 bytes.
- Human `messages --help` is a compact command list, but it advertises the
  whole namespace machine-readable contract as a secondary choice.
- Agent-help schema version 3 uses `view=index` only for root and `view=scope`
  for both namespace and exact-command selectors.

## Constraints

- `cli.Catalog` remains the only command source of truth.
- Root entries retain only path, namespace, summary, capability, outcome,
  effect, and role and remain within 512 encoded bytes per command.
- An exact known path must still reach its full contract in one invocation.
- A namespace response must remain useful as a bounded local index rather than
  merely fail or emit a warning before the same oversized payload.
- Existing uncommitted message-index work owns `--start-index` and `--count`;
  this work must preserve it and must not reintroduce message `--limit`.

## Product decision

- Schema version 4 has two index projections and one exact scope projection.
- Root index: all active exact commands and an exact-command request template.
- Namespace index: only matching compact entries, a namespace scope marker,
  and the same exact-command request template.
- Exact scope: one complete command contract, global I/O/error contracts, and
  workflows touching that command.
- No normal invocation returns multiple complete command contracts.

## Investigation-guidance decision

- `--help` is the default for namespace navigation and ordinary input/effect
  inspection.
- Root `--format agent` is for an outcome not yet mapped to an exact path.
- Exact-command `--format agent` is for output, failure, recovery,
  authentication, mutation, or workflow certainty.
- Message investigation documentation distinguishes the latest bounded source
  from complete history, sender-authored messages from messages addressed to a
  person, and direct one-hop context from a whole thread.

## Security and public boundary

- This is a read-only help projection and documentation change.
- No credential, provider text, remote destination, filesystem mutation,
  command effect, or opaque-reference acceptance changes.
- Smaller namespace output reduces accidental agent-context consumption; it
  does not claim semantic prompt-injection prevention.

## Completion evidence

- Schema-v4 current-worktree output sizes are 11,984 bytes for the root index,
  2,443 bytes for the `messages` namespace index, and 64,403 bytes for the
  exact `messages list` contract.
- The namespace response contains only `schema_version`, `view`, `program`,
  `scope`, `scope_request`, and compact command entries. It contains no global
  input, I/O/error, complete command, mutation, authentication, or workflow
  contract.
- `go test ./internal/cli`, `task check:fast`, and the final `task check` pass
  with Go 1.26.5 on 2026-07-20. The repository gates include architecture,
  contract, race/full tests, security, vulnerability, release-lint, public,
  and generated-diff checks.
