# Work Goal: Publish stable Formula updates through the shared tap

- Status: Complete
- Owner: Chatwork CLI maintainers
- Target: Before the first stable release
- Related ADRs: [ADR 0004](../../decisions/0004-shared-homebrew-tap.md)

## Outcome

A stable `cwk` tag publishes the existing verified GitHub Release artifacts
and proposes the exact audited `Formula/cwk.rb` to
`tasuku43/homebrew-tap`, so users can install `cwk` through
`tasuku43/tap` after that pull request is merged.

## Why now

The maintainer requested the same shared-tap operating model already proven by
`vivi`. The current `cwk` workflow proposes its Formula to the source
repository instead of the shared tap.

## Non-goals

- Provision or rotate GitHub App credentials.
- Push a tag, publish a release, merge a tap pull request, or edit the tap
  repository in this change.
- Replace `cwk` packaging with GoReleaser or `vivi`'s builder.
- Add Linux Homebrew support, signing, notarization, SBOMs, or provenance
  attestations.
- Change CLI commands, output, archives, checksums, or prerelease behavior.

## Acceptance criteria

- [x] A stable tag renders and audits the Formula from the exact release
      revision in an unprivileged job before any shared-tap checkout or
      mutation; a fresh token-bearing runner checks out no tagged source and
      executes no checked-out source or Formula content.
- [x] The workflow creates a token from `HOMEBREW_APP_ID` and
      `HOMEBREW_APP_KEY`, restricted to `tasuku43/homebrew-tap`.
- [x] Only `Formula/cwk.rb` is staged in a pull request against tap `main`,
      using conventions accepted by the tap automation.
- [x] Prereleases never request a Formula update.
- [x] Source and tap checkouts do not persist workflow credentials.
- [x] Release lint fails when the destination, scope, ordering, file, branch,
      title, or credential boundary drifts.
- [x] Japanese installation documentation names both supported Homebrew
      invocation forms and the stable-only rule.
- [x] `task release:check`, `task check`, and `task public:check` pass.

## Governing documents

- Thesis: Axiom 8, executable claims
- Product contract section: Compatibility boundary
- Architecture or security invariant: Supply-chain boundary and least
  privilege in `docs/03_security_model.md`
- Existing ADR: None; ADR 0004 records this derived release decision

## Completion definition

This workflow-change packet is complete when the workflow, release lint,
durable release and security documents, and installation guide agree and all
required gates pass. Secret/App configuration, the first stable tag, tap PR
merge, and post-merge clean Homebrew installation remain explicitly separate
operator follow-ups; release-specific policy decisions remain governed by the
first-release packet.
