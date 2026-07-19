# Work Context: Release cwk v0.1.1

## Current behavior

- Local `main` and `origin/main` initially point to `962e87d`.
- Remote tag `v0.1.0` and its GitHub Release exist; no local/remote `v0.1.1`
  tag or GitHub Release exists as of 2026-07-19.
- GitHub CLI authentication for `tasuku43` is active with repository and
  workflow scopes.
- Actions secret metadata exposes both required names, `HOMEBREW_APP_ID` and
  `HOMEBREW_APP_KEY`; no value was read or copied.
- The exact pre-release source diff contains the configuration-home alias fix,
  tests, product/architecture/security/harness propagation, its work packet,
  and README Homebrew 6 tap-trust plus troubleshooting guidance.

## Relevant structure

- Fix: `internal/infra/commandconfig/store.go`
- Boundary tests: `internal/infra/commandconfig/store_test.go` and
  `resolver_unix_test.go`
- User guidance: `README.md`
- Fix packet: `docs/work/config-home-symlink-compat/`
- Release workflow: `.github/workflows/release.yml`
- Package and Formula checks: `scripts/package-release.sh`,
  `scripts/render-formula.sh`, `scripts/audit-formula.sh`, and
  `scripts/lint-release.sh`

## Constraints

- The tag must identify the exact clean commit reviewed locally and by GitHub
  `main` CI.
- Release publication and archive creation are create-only; artifact changes
  require a new version.
- The App installation must be limited to `homebrew-tap` with Contents and Pull
  requests read/write before the stable tag is pushed.
- The Formula pull request changes only `Formula/cwk.rb`.
- Homebrew availability is not complete until the tap change merges and a
  clean install succeeds.

## Release contents and compatibility

- Included fix: resolve an existing macOS/Linux configuration-home symbolic
  link once to its real absolute directory before joining `cwk`-owned paths.
- Retained protection: the `cwk` directory and preference file remain strict
  non-symbolic targets with Unix `0700`/`0600` modes.
- Included documentation: Homebrew 6 Formula-specific tap trust and actionable
  `command_selection_unsafe` inspection/repair guidance.
- Compatibility: normal directories and existing preference bytes are
  unchanged; affected dotfiles environments change from false unsafe failure
  to normal default/saved-state behavior.
- Security: no credential, authentication, permission, provider, confirmation,
  or network boundary changes. Broken aliases and owned-target links still fail
  closed.
- Migration/deprecation: none.

## External facts

- `gh secret list --app actions --repo tasuku43/cwk` confirms only the required
  secret names used by the workflow; values remain unavailable.
- GitHub's user-installation API rejects the CLI OAuth token for installation
  metadata, so the release owner must verify the App's external maximum in
  GitHub settings.

## Unknowns

- [ ] Confirm immediately before tagging that the GitHub App installation is
  limited to `homebrew-tap` with Contents read/write and Pull requests
  read/write.
- [ ] Preview generated release notes against the exact pushed commit.
- [ ] Record the resulting Release URL, workflow run, Formula pull request,
  merge, and clean installation.

## Reproduction or observation

```sh
gh auth status
git ls-remote --heads --tags origin
gh release view v0.1.1 --repo tasuku43/cwk
gh secret list --app actions --repo tasuku43/cwk
task check
task security
task release:check
task public:check
```

## Security and public-boundary notes

- External effects: one `main` push, annotated tag, immutable GitHub Release,
  short-lived tap branch, and Formula-only pull request.
- Credentials: maintainer GitHub CLI authentication and two Actions secrets;
  no value enters repository content or evidence.
- Public artifacts: five archives, `checksums.txt`, generated release notes,
  checksum-pinned Formula, and workflow logs.
- Confidentiality review: changed examples contain only local generic paths and
  stable public repository identifiers; public/security guards remain required.
- Recovery: stop before tag on any failed precondition. After Release creation,
  never move/reuse the tag or replace assets; repair only unchanged Formula
  rollout state or publish a new patch for changed bytes.

## Glossary

- **reviewed commit**: the clean `main` revision that passed local required
  profiles and GitHub CI before tag creation.
- **stable tag**: a valid release tag without a prerelease suffix; `v0.1.1`
  takes the stable Homebrew workflow path.
