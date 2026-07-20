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
- GitHub's generated-note preview for the direct `main` commit returned only a
  full-changelog comparison link. Release preparation therefore changed the
  workflow to publish reviewed annotated-tag notes and added positive/negative
  lint enforcement before any `v0.1.1` tag was created.
- The reviewed source commit is
  `1362038fb860f4ddc2e6b50719811dd396a68df4`; GitHub `main` CI run
  `29689471296` passed for that exact revision.
- The release owner confirmed immediately before tagging that the App
  installation is limited to `homebrew-tap` with Contents and Pull requests
  read/write.
- Annotated tag `v0.1.1` resolves to the reviewed commit. Release workflow run
  `29690203706` passed preflight and all five matrix builds, then failed before
  publication because `gh release create --notes-from-tag` could not see a
  local tag in the publish job.
- No GitHub Release existed after that failure. The five successful Actions
  artifacts were downloaded, reproduced locally byte-for-byte with Go 1.26.5,
  checksummed, and published once with the exact annotated-tag notes at
  <https://github.com/tasuku43/cwk/releases/tag/v0.1.1>.
- All six published assets were downloaded again; the exact filename set and
  every checksum passed. The rendered Formula also passed `ruby -c` and a real
  Homebrew strict audit.
- The post-tag workflow correction checks out the exact tag before future
  annotated-note publication and adds a stable-tag-only, read-only recovery
  dispatch that verifies an existing six-file Release before resuming the same
  Formula audit and App-scoped tap publisher.
- Recovery run `29699022181` exposed a preflight-only argument-order defect:
  `git rev-list -n 1 -- <tag>` supplied no revision because `--` starts the
  path list. It failed before the full gate and before any Release/tap write.
  The correction peels the already validated annotated tag with
  `git rev-parse --verify "${tag}^{commit}"`; a negative mutation test rejects
  replacing that immutable binding with `HEAD`.
- Corrected main CI run `29699185052` passed. Recovery run `29699364723` then
  passed exact tag/full-gate preflight, read-only six-asset/checksum validation,
  exact-revision Formula render/strict audit, fresh-runner App token creation,
  and Formula-only tap proposal.
- App-authored <https://github.com/tasuku43/homebrew-tap/pull/27> changed only
  `Formula/cwk.rb`; syntax and auto-merge checks in run `29699636925` passed and
  the PR merged on 2026-07-19. The merged Formula is byte-identical to the
  locally strict-audited Formula.
- A clean Homebrew rollout removed only the old Cellar-managed `cwk 0.1.0`,
  retained external configuration, and installed `cwk 0.1.1`. The installed
  binary reports `cwk 0.1.1 (1362038fb860f4ddc2e6b50719811dd396a68df4)`.
- Installed-binary `cwk doctor` passed. A separate temporary
  `XDG_CONFIG_HOME` symbolic link also passed `cwk doctor`; `cwk config` opened
  its TUI and exited with `q` without a write, directly replaying the original
  failed prerequisite.

## Relevant structure

- Fix: `internal/infra/commandconfig/store.go`
- Boundary tests: `internal/infra/commandconfig/store_test.go` and
  `resolver_unix_test.go`
- User guidance: `README.md`
- Fix and regression boundary: `internal/infra/commandconfig/store.go`,
  `store_test.go`, and `resolver_unix_test.go`
- Release workflow: `.github/workflows/release.yml`
- Package and Formula checks: `scripts/package-release.sh`,
  `scripts/render-formula.sh`, `scripts/audit-formula.sh`, and
  `scripts/lint-release.sh`
- Release-note enforcement: `.github/workflows/release.yml`,
  `scripts/lint-release-workflow.sh`, and `scripts/test-release-workflow.sh`

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
- Included release correction: direct commits use reviewed annotated-tag notes
  rather than incomplete generated notes derived from pull-request history.
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

- None blocking release completion.
- Non-blocking follow-up: pinned `actions/create-github-app-token` currently
  warns that `app-id` is deprecated in favor of `client-id`; migration must be
  reviewed with the pinned action contract and secret naming before changing
  the release boundary.

## Reviewed release notes

The exact `v0.1.1` annotated-tag and GitHub Release message is:

```text
cwk v0.1.1

変更:
- macOS/Linuxで、~/.configまたはXDG_CONFIG_HOMEがシンボリックリンクでも、実体が通常のディレクトリならコマンド選択設定を利用できるようにしました。
- Homebrew 6のFormula単位tap trustと、command_selection_unsafeの確認・修復手順をREADMEへ追加しました。

セキュリティ:
- cwkディレクトリとcommand-selection.json自体のシンボリックリンク拒否、およびUnixの700/600権限要件は維持します。

移行:
- 既存の有効な設定ファイルに変更は不要です。
```

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
- Public artifacts: five archives, `checksums.txt`, reviewed annotated-tag notes,
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
