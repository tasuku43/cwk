# Work Goal: Recent message window by default

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: None

## Outcome

An agent or human can run `cwk messages list --room <room-ref>` and receive the
room's latest bounded provider window without remembering an extra flag. The
differential provider window remains available only through the explicit
`--window changes` input.

## Why now

The owner observed that the common task is understanding the current flow of a
room conversation. The existing omitted-window behavior instead requests only
provider differential changes, making the shortest and most discoverable
invocation poorly aligned with that outcome.

## Non-goals

- Removing `--window changes` or changing its provider semantics.
- Fetching complete room history, adding pagination, or making more than one
  provider request.
- Changing the 100-message source bound, typed relation semantics, sender and
  limit composition, flat presentation grammar, or canonical references.
- Changing authentication, effect, mutation policy, output formats, or other
  commands' defaults.

## Acceptance criteria

- [x] Omitting `--window` selects `recent`, sends the documented `force=1`
  query, and renders `window=recent`.
- [x] Exact human and agent help list `recent|changes`, identify `recent` as the
  default, and describe `changes` as the explicit differential alternative.
- [x] Explicit `--window recent` remains equivalent to omission.
- [x] Explicit `--window changes` sends no `force` query and renders
  `window=changes`.
- [x] `--sender`, `--limit`, and `--context` retain their existing bounded
  selection order and use the recent source window when `--window` is omitted.
- [x] The provider call count remains one; source limit, completeness,
  relation, trust, and canonical-reference contracts do not change.
- [x] README, theses, product, architecture, harness, Skill guidance, and the
  active agent-readiness scenario agree on the default.
- [x] Focused tests, relevant race tests, and `go vet` pass.
- [x] `task check` passes.
- [x] The change is committed intentionally in reviewed slices with no
  unrelated worktree change absorbed.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1 and 2
- Product contract: `docs/01_product_contract.md`, Filtering and task
  composition
- Architecture: CLI default selection over the existing typed request and
  provider `force` mapping
- Security invariant: one bounded read with unchanged authentication,
  destination, timeout, and source ceiling
- Existing ADR: None

## Completion definition

Stop when omission and both explicit values have executable end-to-end
evidence, the common latest-window task needs no redundant flag, durable
contracts agree, `task check` passes, and the reviewed change is committed. Do
not expand into history traversal or another message-selection redesign.
