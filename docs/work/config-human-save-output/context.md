# Work Context: Human-readable config save result

## Baseline observation

- The selector itself is a terminal-native human interface, but confirmed save
  success is one machine-shaped line with English keys and a full SHA-256 value.
- `config` accepts no non-interactive grammar and supports only text output.
- The normal fingerprint duplicates reconciliation data already owned by the
  uncertain fault and read-only `doctor`.
- Existing tests require the machine-shaped line and legacy cleanup key.

## Implemented observation

- A synthetic real-terminal Enter save with an isolated config home emitted
  `33件を表示し、0件を非表示にしました（0件変更）。` after terminal
  restoration, with no fingerprint on confirmed success.
- The following read-only `doctor` run retained the saved source, counts, and
  deterministic fingerprint.
- While this change was in progress, the maintainer concurrently replaced the
  detailed README with a shorter product-oriented version. That rewrite is
  preserved; the required `config` guidance is added as a concise section in
  its new style rather than restoring the old README.

## Relevant structure

- Success renderer: `internal/cli/config.go`
- Public output declaration: `internal/cli/config_catalog.go`
- Golden and lifecycle tests: `internal/cli/config_command_test.go`
- Reconciliation: `internal/cli/config_diagnostic.go` and
  `internal/cli/config_diagnostic_test.go`

## Constraints

- Confirmed success must remain stdout with exit zero and one complete write.
- A late cancellation cannot overwrite confirmed success.
- Cleanup evidence remains visible when stale or legacy entries were removed.
- Uncertain outcomes still carry the candidate fingerprint and require
  `doctor` to report `source=saved` plus that exact value.
- Japanese is the only user-facing prose language; stable catalog field names
  remain locale-neutral ASCII.

## Interface evidence

The maintainer compared three transcript concepts:

- A: a short natural summary with cleanup only when nonzero;
- B: an aligned label/value list; and
- C: a complete audit block including the fingerprint.

Concept A was selected explicitly. Its normal transcript is:

```text
コマンド表示を保存しました。
12件を表示し、21件を非表示にしました（22件変更）。
```

When cleanup occurs, it adds:

```text
古い設定を2件整理しました。
```

## Thesis evidence

- Human usability was degraded by exposing recovery internals on an ordinary
  successful interactive path.
- This is presentation evidence, not a change to command-selection semantics or
  the thesis that `doctor` owns uncertain-write reconciliation.

## Security and public-boundary notes

- Assets, target, effect, saved bytes, and external destinations do not change.
- The removed success fingerprint contains no secret, but omitting it reduces
  irrelevant diagnostic detail on a human path.
- The fingerprint remains available exactly where it changes safe recovery:
  the uncertain-save error and read-only `doctor`.

## Glossary

- **visible**: a configurable Chatwork task retained in the active help and
  routing view; it does not imply provider authorization.
- **hidden**: a configurable Chatwork task omitted from that local view; it is
  not a security boundary.
- **cleaned**: the total stale or legacy settings removed by the save.
