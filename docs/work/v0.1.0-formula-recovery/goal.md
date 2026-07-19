# Work Goal: Recover v0.1.0 Homebrew Formula publication

- Status: In progress
- Owner: Release maintainer
- Target: v0.1.0 post-publication recovery
- Related ADRs: [ADR 0004](../../decisions/0004-shared-homebrew-tap.md)

## Outcome

The already-published `v0.1.0` archives remain immutable, while the exact
checksum-pinned Formula is strictly audited and proposed as the sole change in
the shared Homebrew tap.

## Acceptance criteria

- [ ] Audit staging normalizes an owner-only rendered Formula to `0644`.
- [ ] A regression test fails without that normalization.
- [ ] Required repository gates pass on the fix.
- [ ] The published `v0.1.0` checksums render the recovered Formula.
- [ ] Strict Homebrew audit passes before a Formula-only tap pull request.
- [ ] Neither the published tag nor Release assets are rewritten.
