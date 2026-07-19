# Work Context: Publish stable Formula updates through the shared tap

## Observed baseline

- `.github/workflows/release.yml` already validates a leading-`v` SemVer tag,
  binds every build to one exact revision, runs the full gate, builds five
  canonical archives, publishes create-only GitHub Release assets, and skips
  Formula work for prereleases.
- The current Formula job renders and strictly audits from the tagged revision,
  then checks out `cwk` `main` and proposes `Formula/cwk.rb` there.
- `scripts/render-formula.sh`, `Formula/cwk.rb.template`, and
  `scripts/audit-formula.sh` already implement the reviewed macOS Formula
  contract and must remain the generator and audit boundary.
- The worktree was clean and `.harness/project.json` reported `profile: ready`
  before implementation.

## Relevant structure

- Entry point: `.github/workflows/release.yml`
- Release artifact boundary: `scripts/package-release.sh`
- Formula generation and audit: `scripts/render-formula.sh`,
  `scripts/audit-formula.sh`
- Existing tests and harness checks: `scripts/lint-release.sh`,
  `task release:check`, `task check`

## Constraints

- GitHub Release assets are create-only and must not be overwritten.
- Formula generation remains bound to the exact tag revision and downloaded
  release checksums.
- Stable tags update Formula metadata; prereleases do not.
- Workflow actions remain pinned, tokens are least-privilege, and checkout
  does not persist credentials.
- Checked-out tagged source/tool execution and the shared-tap App token never
  share a runner; the audited Formula crosses jobs only as a one-file artifact
  treated as data.
- Workflow and Formula-job field allowlists reject ambient `env` or `defaults`
  injection before the token action, and staging rejects symbolic Formula
  destination paths or an existing non-regular target.
- The automated tap pull request changes only one Formula file; the tap README
  is a separate external change.
- User-facing repository prose is Japanese.

## External facts

- `tasuku43/vivi` release workflow, checked 2026-07-19:
  <https://github.com/tasuku43/vivi/blob/main/.github/workflows/release.yml>.
  Stable releases create a GitHub App token from `HOMEBREW_APP_ID` and
  `HOMEBREW_APP_KEY`, restricted to `tasuku43/homebrew-tap`, then open a
  Formula-only pull request.
- `tasuku43/homebrew-tap` workflow, checked 2026-07-19:
  <https://github.com/tasuku43/homebrew-tap/blob/main/.github/workflows/test.yml>.
  The automation admits the App bot, `main` base, a branch beginning
  `chore/homebrew-formula-v`, a title beginning
  `Update Homebrew formula for v`, and Formula-only changes.
- The `vivi` v0.0.15 release and tap PR 24 provide an observed successful
  release-to-tap path. Their artifact generator is not reusable for `cwk`.

## External follow-ups

- [ ] The operator will configure the two Actions secrets and confirm the App
      installation permissions before the first stable tag.
- [ ] First-release clean installation evidence cannot exist until the Formula
      pull request is merged; record it in that release's work packet.
- [ ] Update the shared tap README in a separate reviewed change after
      `Formula/cwk.rb` is available; Formula automation intentionally rejects
      README changes.

## Thesis evidence

- The release destination is a durable supply-chain choice but does not alter
  the task-oriented CLI thesis.
- Copying another tool's weaker artifact builder would route around the
  existing release thesis; retaining `cwk`'s verified builder avoids that
  exception.

## Reproduction or observation

```sh
go run ./tools/projectmeta --field profile
git status --short --branch
task release:check
```

The initial profile was `ready`, the branch was clean `main`, and the existing
release profile passed with the exact Go toolchain selected by `go.mod`.

## Security and public-boundary notes

- Assets and side effects: a branch and pull request containing only
  `Formula/cwk.rb` in the public shared tap.
- Credentials: GitHub App ID/private key in Actions secrets; a short-lived
  installation token restricted to the tap. No value enters source, and the
  token-bearing runner checks out no tagged source and executes no checked-out
  source or Formula content.
- New destination: `github.com/tasuku43/homebrew-tap` during release
  automation only.
- Dependencies: `actions/create-github-app-token` is pinned to
  `bcd2ba49218906704ab6c1aa796996da409d3eb1`, verified as the v3.2.0 release
  commit on 2026-07-19. Its pinned `action.yml` uses the bundled Node 24 action,
  defines repository and per-permission inputs, and the selected workflow asks
  only for Contents write and Pull requests write. The license at that exact
  revision is MIT. Evidence: [release commit](https://github.com/actions/create-github-app-token/commit/bcd2ba49218906704ab6c1aa796996da409d3eb1),
  [action definition](https://github.com/actions/create-github-app-token/blob/bcd2ba49218906704ab6c1aa796996da409d3eb1/action.yml),
  and [license](https://github.com/actions/create-github-app-token/blob/bcd2ba49218906704ab6c1aa796996da409d3eb1/LICENSE).
  `peter-evans/create-pull-request` is pinned to
  `5f6978faf089d4d20b00c7766989d076bb2fc7f1`. Its pinned Node 24 action
  definition confirms that `token` performs the PR mutation, `branch-token`
  defaults to that same token, `path` selects the checked-out tap, and
  `add-paths` limits committed pathspecs when explicitly set. Its exact-revision
  license is MIT. Evidence: [commit](https://github.com/peter-evans/create-pull-request/commit/5f6978faf089d4d20b00c7766989d076bb2fc7f1),
  [action definition](https://github.com/peter-evans/create-pull-request/blob/5f6978faf089d4d20b00c7766989d076bb2fc7f1/action.yml),
  and [license](https://github.com/peter-evans/create-pull-request/blob/5f6978faf089d4d20b00c7766989d076bb2fc7f1/LICENSE).
  Both actions bundle their runtime and add no release-time package install.
- Publication: Formula URLs remain public GitHub Release URLs and checksums;
  the external tap merge remains separately reviewable.

## Glossary

- **source repository**: `tasuku43/cwk`.
- **shared tap**: GitHub repository `tasuku43/homebrew-tap`, installed through
  the Homebrew name `tasuku43/tap`.
- **Formula PR**: the stable-release pull request changing only
  `Formula/cwk.rb` in the shared tap.
