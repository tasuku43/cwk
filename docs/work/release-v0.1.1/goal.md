# Work Goal: Release cwk v0.1.1

- Status: Complete
- Owner: Release maintainer
- Target: v0.1.1 on 2026-07-19
- Related ADRs: [ADR 0004](../../decisions/0004-shared-homebrew-tap.md)

## Outcome

`v0.1.1` is published from one reviewed `main` commit as an immutable GitHub
Release containing the complete five-platform archive set and checksums. The
patch accepts a symlinked macOS/Linux configuration home used by common
dotfiles layouts, publishes corrected Homebrew 6 tap-trust installation
guidance, and rolls the checksum-pinned Formula through the shared tap.

## Why now

A clean `v0.1.0` Homebrew installation failed before normal use when
`~/.config` was a symbolic link. The local fix and its strict owned-target
tests are complete, and Homebrew 6 installation guidance also needs to match
the current tap-trust workflow.

## Non-goals

- Allow `${XDG_CONFIG_HOME:-$HOME/.config}/cwk` or
  `command-selection.json` itself to be a symbolic link.
- Change command-selection content, authentication, provider behavior, command
  catalog, output schema, or mutation policy.
- Rewrite `v0.1.0` or replace any published artifact.
- Add Linux Homebrew support, signing, notarization, SBOMs, or provenance.

## Acceptance criteria

- [x] One reviewed commit on `main` contains the compatibility fix,
  installation guidance, enforcement, and both active work packets.
- [x] All required local profiles and GitHub `main` CI pass for that exact
  source state.
- [x] Annotated tag `v0.1.1` points to that commit and produces one create-only
  GitHub Release with five archives and `checksums.txt`.
- [x] Published checksums match every archive and release metadata identifies
  the reviewed commit.
- [x] The stable workflow proposes only `Formula/cwk.rb` to
  `tasuku43/homebrew-tap` through the reviewed App boundary.
- [x] After the Formula pull request merges, a clean
  `brew install tasuku43/tap/cwk` succeeds.
- [x] No credential value, private identifier, personal data, or mutable
  private asset URL is published.

## Governing documents

- Thesis: [Project theses](../../00_theses.md)
- Product contract: [Product contract](../../01_product_contract.md)
- Security model: [Security model](../../03_security_model.md)
- Public boundary: [Public repository](../../05_public_repository.md)
- Release contract: [Release model](../../06_release.md)
- Existing ADR: [ADR 0004](../../decisions/0004-shared-homebrew-tap.md)

## Completion definition

The goal is complete when the immutable Release exists at the reviewed commit,
all published assets verify, the Formula-only tap pull request has merged, a
clean Homebrew installation succeeds, and no required recovery remains.
