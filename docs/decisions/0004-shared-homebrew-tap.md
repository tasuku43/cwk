# ADR 0004: Publish the stable Formula through the shared Homebrew tap

- Status: Accepted
- Date: 2026-07-19
- Deciders: Chatwork CLI maintainers
- Scope: Release, supply chain, public boundary, and GitHub Actions credentials
- Supersedes: None
- Superseded by: None

## Context

`cwk` already builds and verifies an immutable GitHub Release and renders a
checksum-pinned Homebrew Formula from the exact tagged revision. The existing
workflow proposes that Formula back to the `cwk` source repository, while the
maintainer's CLI tools are distributed from the public
`tasuku43/homebrew-tap` repository. The established `vivi` release path uses a
GitHub App installation token scoped to that tap and opens a Formula-only pull
request after publishing a stable release.

The `cwk` packaging contract is stronger and materially different from
`vivi`'s artifact builder. Replacing it or copying `vivi`'s Formula generator
would weaken reviewed archive, legal-notice, prerelease, and reproducibility
contracts without helping the shared-tap outcome.

## Decision drivers

- Give users one shared `tasuku43/tap` installation namespace for maintained
  CLI tools.
- Preserve `cwk`'s exact-revision build, reproducibility, immutable-release,
  checksum, legal-notice, Formula-rendering, and strict-audit contracts.
- Confine cross-repository write authority to `tasuku43/homebrew-tap`.
- Keep Formula publication reviewable and Formula-only rather than writing
  directly to the tap's default branch.
- Keep prereleases from replacing stable Homebrew metadata.

## Considered options

### Keep the Formula pull request in the source repository

This retains the current workflow but does not publish through the shared tap.
Users would need another tap or a manual copy, and the maintained tools would
not share one installation path.

### Open a scoped pull request in the shared tap

This reuses the already audited Formula, grants a short-lived installation
token access only to the tap, and leaves the exact change visible before it is
merged. It also matches the proven `vivi` operating model without copying its
different packaging implementation.

### Push the Formula directly to the shared tap's default branch

This removes one review step but gives the release job a less reviewable
mutation and bypasses the tap's pull-request checks and automation contract.

## Decision

For stable tags only, render and strictly audit `Formula/cwk.rb` from the exact
release revision in a job with no App credential, and upload only that Formula
as a workflow artifact. A fresh dependent runner checks out no source
repository and executes no checked-out tagged source or Formula content; it
validates that artifact as data, then uses a GitHub App installation token restricted to
`tasuku43/homebrew-tap` to propose that one file against the tap's `main`
branch. The publish job uses repository secrets `HOMEBREW_APP_ID` and
`HOMEBREW_APP_KEY`; secret provisioning remains an operator action and no
credential value is committed. Token creation fixes that repository and
explicitly requests only Contents write and Pull requests write as
write-capable permissions. The pull-request branch starts with the tap's
accepted `chore/homebrew-formula-v` prefix and includes `cwk` to avoid
same-version branch collisions between tools. Prereleases publish GitHub
Release artifacts but do not request a Formula change.

The existing `cwk` package builder, five-target matrix, archive names,
checksums, Formula template, strict audit, and create-only GitHub Release path
remain authoritative. Homebrew support remains the reviewed macOS arm64 and
amd64 contract; Linux Homebrew support and provenance attestations are
separate decisions.

## Consequences

### Positive

- Users can install stable `cwk` releases from the same tap as the maintainer's
  other CLI tools.
- The source repository's workflow token needs no cross-repository write
  permission.
- Checked-out tagged source/tool execution and the App token never share a
  runner.
- The tap receives a deterministic, checksum-pinned, pre-audited Formula-only
  change.
- Existing prerelease and immutable-artifact behavior does not change.

### Negative

- A stable release now depends on two separately administered GitHub
  repositories and a GitHub App installation.
- GitHub Release publication can succeed before the asynchronous tap pull
  request is merged; release completion does not claim that the Formula is
  already available.
- The tap README must be updated separately because its automated Formula
  pull requests intentionally admit only Formula files.

### Risks and mitigations

- Missing or mis-scoped App credentials fail the Formula job without exposing
  a secret; the operator configures the two repository secrets before tagging
  and may rerun the failed job after correcting them.
- A compromised token is bounded to the tap repository and the job stages only
  `Formula/cwk.rb`; the pull-request action receives the token explicitly and
  checkout does not persist it.
- Concurrent tools can release the same version; the branch keeps the tap's
  automation prefix and adds a `cwk` suffix.
- The external tap workflow can drift; `cwk` audits before proposing, pins the
  required branch/title/file conventions locally, and still treats merge as a
  separate review boundary.

## Mechanical enforcement

- `scripts/lint-release-workflow.sh` enumerates every Formula-job step start and
  validates exact checksum/toolchain, render/audit/artifact upload,
  fresh-runner artifact validation, token, tap checkout, staging, and
  pull-request shapes. It confines the App action and each secret to one
  reviewed workflow-wide occurrence and fixes the audit-to-publish dependency,
  shared-tap repository, requested permissions, read-only Formula-job source
  token, zero source-repository permissions in the token-bearing job,
  credential-free checkouts, reviewed workflow and Formula-job fields without
  ambient `env` or `defaults`, absence of tagged-source checkout or checked-out
  code execution in that runner, audited Formula binding, non-symbolic Formula
  destination paths, Formula-only staging, accepted branch/title conventions,
  and
  render/audit-before-tap ordering.
- `scripts/test-release-workflow.sh` applies representative scope, permission,
  token use, duplicate-secret, ambient runtime, audited-source, symbolic path,
  base, title, wildcard, ignored-failure, and persisted-credential mutations
  and proves that the workflow lint rejects them.
- The release workflow pins the App-token and pull-request actions to reviewed
  commit SHAs.
- `task release:check` renders the real Formula from complete release
  checksums, validates its URLs and digests, runs `ruby -c`, and exercises the
  strict-audit ownership and cleanup boundary with fake Homebrew. The stable
  macOS Formula job performs the real Homebrew strict audit before minting the
  App token.
- `task check` includes release, security, and public-boundary profiles.
- The release owner manually confirms the App installation is limited to
  `homebrew-tap` with Contents read/write and Pull requests read/write.

## Compatibility and migration

No CLI command, output, archive, checksum, version, or GitHub Release contract
changes. The Formula update destination moves from the source repository to
the shared tap before the first stable release. Existing prerelease behavior
is unchanged. Rollback is a reviewed workflow change; published GitHub Release
artifacts are never overwritten.

## Security and public-boundary impact

The new external asset is one branch and pull request in
`tasuku43/homebrew-tap`. The new credentials are a GitHub App ID and private
key stored as GitHub Actions secrets and exchanged for a short-lived token
restricted to that repository. The token must not enter source checkout
configuration, Formula content, artifacts, logs, or repository files. No new
runtime dependency, CLI credential, provider destination, or user-data flow is
introduced.

## Validation

- `task release:check`
- `task check`
- `task public:check`
- First stable release: inspect the Formula-only tap pull request and perform a
  clean `brew install tasuku43/tap/cwk` after merge.

## Reconsideration signals

Reconsider when the shared tap changes its branch/title or review contract,
the App can no longer be restricted to one repository, Linux Homebrew becomes
a supported outcome, or signed provenance becomes a release requirement.
