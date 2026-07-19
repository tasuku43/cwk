# Work Tasks: Release cwk v0.1.1

## Prepare

- [x] Select `v0.1.1` as a stable patch release.
- [x] Confirm no local/remote tag or GitHub Release uses `v0.1.1`.
- [x] Confirm both required Actions secret names exist without reading values.
- [x] Record included changes, compatibility, security impact, recovery, and
  non-goals.
- [x] Reject the incomplete generated-note preview and enforce reviewed
  annotated-tag notes with a negative workflow mutation test.
- [x] Confirm the complete working-tree scope includes the configuration-home
  compatibility fix and the existing Homebrew 6 README change.
- [x] Confirm the App installation is limited to `homebrew-tap` with Contents
  read/write and Pull requests read/write.
- [x] Obtain the release owner's instruction to publish the current source to
  `main` as `v0.1.1`.

## Verify before tag

- [x] `task check` passes on the final source tree. Evidence: the exact Go
  1.26.5 full profile passed on 2026-07-19.
- [x] `task security` passes on the final source tree. Evidence: module
  verification and repoguard passed; `govulncheck` found no called
  vulnerabilities on 2026-07-19.
- [x] `task release:check` passes on the final source tree. Evidence: two
  complete five-target package passes, archive/checksum/Formula verification,
  strict audit fixtures, and workflow lint completed with `lint-release: OK`
  on 2026-07-19.
- [x] `task public:check` passes on the final source tree. Evidence: repoguard
  public and contractlint passed on 2026-07-19.
- [x] `git diff --check` passes and every changed path is understood. Evidence:
  the reviewed scope is the store fix, two test files, README installation
  guidance, governing documents, release-note workflow enforcement, and the
  fix/release work packets.
- [x] The exact committed `main` revision passes GitHub CI. Evidence: run
  `29689471296` passed for
  `1362038fb860f4ddc2e6b50719811dd396a68df4`.
- [x] Annotated-tag release notes for the exact remote commit are reviewed.

## Publish

- [x] Commit the fix, tests, documentation, and work packets to `main`.
- [x] Push the reviewed commit to `origin/main`.
- [x] Create annotated tag `v0.1.1` at that exact commit.
- [x] Push only that tag and monitor the Release workflow. Evidence: run
  `29690203706` passed preflight and five builds; its publish checkout defect
  was recovered without moving the tag or replacing any published asset.

## Verify publication and rollout

- [x] Download the five archives and `checksums.txt`, verify the exact filename
  set, and recompute every checksum.
- [x] Release metadata identifies `v0.1.1` and the reviewed commit.
- [ ] Dispatch the stable-tag-only recovery workflow and confirm it revalidates
  the immutable Release before crossing the App boundary.
- [ ] The shared-tap pull request changes only `Formula/cwk.rb`.
- [ ] The Formula pull request merges through reviewed automation.
- [ ] A clean `brew install tasuku43/tap/cwk` succeeds after merge.

## Hand off

- [ ] Record final URLs and bounded evidence without credential values.
- [ ] Mark the goal complete only after the post-merge clean install succeeds.
- [ ] Leave any non-blocking follow-up explicit.
