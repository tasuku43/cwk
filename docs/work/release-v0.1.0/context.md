# Work Context: Release cwk v0.1.0

## Current behavior

- Local `main` and `origin/main` initially pointed to commit `8d15734`; the
  shared-tap release implementation was an understood uncommitted change.
- During release preparation, the maintainer committed and pushed the reviewed
  shortened README separately as `5f6c03c`. The remaining release/config
  commit builds on that exact `main` parent; no history is rewritten.
- On 2026-07-19, `gh release list --repo tasuku43/cwk` and
  `git ls-remote --tags origin` returned no existing releases or tags.
- `gh secret list --repo tasuku43/cwk --app actions` confirmed the names
  `HOMEBREW_APP_ID` and `HOMEBREW_APP_KEY`. No secret value was read or copied.
- The release workflow accepts exact leading-`v` SemVer. A version without a
  prerelease suffix runs the shared-tap Formula path.

## Relevant structure

- Release workflow: `.github/workflows/release.yml`
- Package and Formula checks: `scripts/lint-release.sh`,
  `scripts/lint-release-workflow.sh`, `scripts/package-release.sh`,
  `scripts/render-formula.sh`, and `scripts/audit-formula.sh`
- Release policy: `docs/05_public_repository.md` and `docs/06_release.md`
- Shared-tap decision: `docs/decisions/0004-shared-homebrew-tap.md`

## Constraints

- The tag must identify the exact clean commit that passed review and gates.
- GitHub Release publication is create-only; a bad immutable artifact requires
  a new version rather than overwrite or tag reuse.
- Stable publication can create the GitHub Release before a later Formula job
  fails, so App installation scope must be confirmed before tag push.
- The Formula update changes exactly `Formula/cwk.rb`; tap README changes are a
  separate review.
- Homebrew availability is not claimed until the tap pull request merges and a
  clean installation succeeds.
- Generated release notes must be previewed against the exact remote commit
  before tag push; the workflow's generated notes are public release content.

## Release contents and compatibility

- Included changes: the current task-oriented Chatwork CLI catalog and its
  authentication, reference, effect, bounded-call, structured outcome, Japanese
  presentation, command-selection, packaging, and shared-tap contracts. The
  final pre-release command-selection presentation replaces its machine-shaped
  confirmed-save record with the short natural-Japanese Concept A result and
  corresponding scoped agent fields. The maintainer's shortened README is also
  part of the public source state under review.
- Compatibility: this is the first published version and establishes the
  baseline. SemVer major zero permits deliberate compatibility changes in later
  releases, which must still be documented rather than silently rewritten.
- Security: no coordinated embargoed security fix is disclosed by this
  release. The shared-tap isolation and negative workflow tests strengthen the
  release supply-chain boundary.
- Migration and deprecation: none, because no earlier public version, tag, or
  Release exists. The superseded config success projection was never released.
- Pre-publication installation: README contains no non-Homebrew package-manager
  install path. The release profile builds, extracts, inspects, and executes the
  exact supported archive matrix before publication. Homebrew remains a
  separately recorded post-merge rollout check.

## External facts

- The maintainer reported on 2026-07-19 that the required secrets were set.
- The existing shared App and tap automation are used by `vivi`; ADR 0004 and
  the shared-tap release work packet record the observed branch, title, author,
  and Formula-only conventions.
- Repository secret metadata cannot prove the App's external repository scope
  or maximum installation permissions. The release owner must confirm that
  boundary without copying secret values into this packet.

## Unknowns

- [ ] Confirm immediately before tagging that the GitHub App installation is
      limited to `homebrew-tap` with Contents read/write and Pull requests
      read/write.
- [ ] The resulting GitHub Release URL, Formula pull request, and clean install
      result do not exist until the tag workflow and post-merge rollout run.
- [ ] Preview and review GitHub's generated release notes after the final
      commit is pushed and before the tag is created.

## Thesis evidence

- `v0.1.0` establishes the first public compatibility baseline while SemVer
  major zero keeps the product's early-evolution status explicit.
- A prerelease would verify artifacts but intentionally skip the Homebrew path;
  it would not validate the requested shared-tap outcome.

## Reproduction or observation

```sh
gh release list --repo tasuku43/cwk --limit 20
git ls-remote --tags origin
gh secret list --repo tasuku43/cwk --app actions
task check
task security
task release:check
task public:check
```

## Security and public-boundary notes

- External effects: a `main` push, annotated tag, immutable GitHub Release,
  short-lived tap branch, and Formula-only pull request.
- Credentials: GitHub CLI authentication for maintainer pushes and the two
  Actions secrets for the tap App. Values stay outside repository content.
- Public artifacts: five platform archives, `checksums.txt`, generated release
  notes, one checksum-pinned Formula, and workflow logs.
- Recovery: rerun a failed post-release Formula job after correcting external
  App state; never overwrite Release assets or reuse `v0.1.0` for new bytes.

## Glossary

- **reviewed commit**: the clean `main` commit that passed the local gates and
  GitHub CI before tag creation.
- **stable tag**: a valid release tag without a prerelease suffix; `v0.1.0` is
  stable for workflow routing even though its SemVer major is zero.
