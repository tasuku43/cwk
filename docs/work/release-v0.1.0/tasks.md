# Work Tasks: Release cwk v0.1.0

## Prepare

- [x] Select `v0.1.0` as the first public stable version.
- [x] Confirm no existing remote tag or GitHub Release uses the version.
- [x] Confirm both required Actions secret names exist without reading values.
- [x] Record version rationale, public effects, recovery, and non-goals.
- [x] Confirm the maintainer's reviewed shortened README commit `5f6c03c` is
      already on `main` and `origin/main` as the remaining release work's parent.
- [ ] Confirm the App installation is limited to `homebrew-tap` with Contents
      read/write and Pull requests read/write.
- [ ] Obtain the release owner's explicit public-boundary approval for the
      exact source state, including confidentiality, ownership, trademark,
      history, and first-release readiness.

## Verify before tag

- [x] `task check` passes on the final source tree. Evidence: exact Go 1.26.5
      full profile passed on 2026-07-19 and was rerun after recording this
      final evidence.
- [x] `task security` passes on the final source tree. Evidence: the explicit
      exact Go 1.26.5 profile reported zero called vulnerabilities on
      2026-07-19 and was rerun after recording final evidence.
- [x] `task release:check` passes on the final source tree. Evidence: the exact
      Go 1.26.5 release profile completed with `lint-release: OK` on 2026-07-19
      and was rerun after recording final evidence.
- [x] `task public:check` passes on the final source tree. Evidence: the
      explicit public profile completed with repoguard and contractlint OK on
      2026-07-19 and was rerun after recording final evidence.
- [x] `git diff --check` passes and every changed path is understood. Evidence:
      the pending 32-path diff contains the shared-tap boundary, Concept A
      config result, executable enforcement, durable decisions, and three work
      packets; together with parent commit `5f6c03c` it forms the reviewed
      33-path release source delta. Three independent read-only reviews found
      no remaining P0-P2 issue.
- [x] Pre-publication installation evidence is recorded. Evidence: README has
      no non-Homebrew package-manager installation instruction, so no separate
      clean install applies before publication; the final release profile
      built, extracted, inspected, and checksum-verified all five archives and
      executed the host-native version contract.
- [ ] The exact committed `main` revision passes GitHub CI. Evidence:
- [ ] GitHub's generated release notes for the exact remote commit were
      previewed and agree with the included changes, compatibility, security,
      and migration impact recorded in `context.md`. Evidence:

## Publish

- [ ] Commit the release workflow, Concept A config result, enforcement,
      documentation, all active work packets, and this packet on top of the
      already-pushed README commit.
- [ ] Push the reviewed commit to `origin/main`.
- [ ] Create annotated tag `v0.1.0` at that exact commit.
- [ ] Push only that tag and monitor the Release workflow.

## Verify publication and rollout

- [ ] Download the five published archives and `checksums.txt`, prove their
      one-to-one filename set, and recompute every published checksum.
- [ ] Release metadata identifies `v0.1.0` and the reviewed commit.
- [ ] The shared-tap pull request changes only `Formula/cwk.rb`.
- [ ] The Formula pull request merges through the tap's reviewed automation.
- [ ] A clean `brew install tasuku43/tap/cwk` succeeds after merge.
- [ ] Homebrew availability is announced only after the clean install.

## Hand off

- [ ] Record final URLs and bounded evidence without credential values.
- [ ] Mark the goal complete only after the post-merge install succeeds.
- [ ] Leave any non-blocking follow-up explicit.
