# Work Goal: Human-readable config save result

- Status: In progress
- Owner: Release maintainer
- Target: v0.1.0
- Related ADRs: [ADR 0003](../../decisions/0003-chatwork-pat-only.md)

## Outcome

After a person saves the interactive `cwk config` selector, the terminal
reports the result in short natural Japanese: how many commands will be shown,
how many hidden, and how many display choices changed. Old-setting cleanup is
mentioned only when it occurred. Normal success no longer prints an internal
key/value record or reconciliation fingerprint.

## Why now

A real terminal run produced
`config を保存しました enabled=12 disabled=21 changed=22 ... fingerprint=...`.
The maintainer identified that output as inappropriate for a deliberately
human-operated command and selected Concept A before the first public release.

## Non-goals

- Change selector keys, rows, colors, viewport, persistence, or terminal
  lifecycle.
- Change the saved profile schema or active command view.
- Remove the fingerprint from `doctor` or an uncertain-save fault.
- Add a machine-readable or non-interactive `config` mode.

## Acceptance criteria

- [x] The final result suffix emitted after terminal restoration is exactly two
      natural-Japanese lines and reports visible, hidden, and changed counts.
- [x] Cleanup adds one natural-Japanese line only when its count is nonzero.
- [x] Normal success contains no `enabled=`, `disabled=`, stale/legacy keys,
      or SHA-256 fingerprint.
- [x] `doctor` and uncertain-save reconciliation retain their exact source and
      fingerprint contract.
- [x] Catalog fields, human/agent help, durable docs, tests, and release packet
      agree before `v0.1.0`.
- [x] `task check` passes.

## Governing documents

- Thesis: [Axiom 2](../../00_theses.md#axiom-2-an-agent-reaches-an-executable-task-without-guessing)
- Product contract: [Product contract](../../01_product_contract.md)
- Architecture: [Architecture](../../02_architecture.md)
- Security: [Security model](../../03_security_model.md)
- Harness: [Harness](../../04_harness.md)

## Completion definition

The work is complete when the selected output appears in a real synthetic
terminal run, every reconciliation invariant remains tested, all required
gates pass, and the change is included in the exact `v0.1.0` release commit.
