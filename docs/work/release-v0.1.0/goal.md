# Work Goal: Release cwk v0.1.0

- Status: In progress
- Owner: Release maintainer
- Target: v0.1.0 on 2026-07-19
- Related ADRs: [ADR 0004](../../decisions/0004-shared-homebrew-tap.md)

## Outcome

The first public `cwk` version is published from one reviewed `main` commit as
an immutable GitHub Release containing the complete five-platform archive set
and checksums. The stable Formula is proposed as a one-file pull request to the
shared `tasuku43/homebrew-tap`, then verified by a clean install after merge.

## Why now

The CLI, public repository boundary, reproducible package contract, and shared
Homebrew tap workflow are implemented. The snapshot immediately before this
release packet passed the repository gates; the final source tree and committed
revision still require the evidence tracked below. GitHub Actions now contains
the two required App secret names, and no existing remote tag or GitHub Release
claims the initial version.

## Non-goals

- Claim compatibility guarantees associated with a `v1.0.0` release.
- Add Linux Homebrew support, signing, notarization, or provenance.
- Rewrite or replace a published artifact after release.
- Make further CLI, catalog, output, authentication, or provider changes after
  accepting the pre-release human-readable `config` result refinement.

## Acceptance criteria

- [ ] One reviewed commit on `main` contains the shared-tap workflow and this
      release packet.
- [ ] All required local and GitHub `main` gates pass for that source state.
- [ ] Annotated tag `v0.1.0` points to that exact commit and produces one
      create-only GitHub Release with five archives and `checksums.txt`.
- [ ] The stable workflow proposes only `Formula/cwk.rb` to
      `tasuku43/homebrew-tap` with the reviewed App boundary.
- [ ] After the Formula pull request merges, a clean
      `brew install tasuku43/tap/cwk` succeeds before Homebrew availability is
      announced.
- [ ] No credential value, private identifier, or mutable private asset URL is
      published.

## Governing documents

- Thesis: [Project theses](../../00_theses.md)
- Product contract: [Product contract](../../01_product_contract.md)
- Security invariant: [Shared Homebrew tap publication](../../03_security_model.md#shared-homebrew-tap-publication)
- Release contract: [Release model](../../06_release.md)
- Existing ADR: [ADR 0004](../../decisions/0004-shared-homebrew-tap.md)

## Completion definition

The goal is complete when the tag and immutable Release exist at the reviewed
commit, the Formula-only tap pull request has merged, clean Homebrew
installation evidence is recorded, and no required recovery work remains.
