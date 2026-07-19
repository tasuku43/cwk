# Work Tasks: Publish stable Formula updates through the shared tap

## Understand

- [x] Read governing documents 00 through 09.
- [x] Observe the current source-repository Formula PR behavior.
- [x] Verify the current `vivi` and shared-tap workflows and one successful
      release path.
- [x] Confirm `profile: ready` and a clean starting worktree.

## Decide

- [x] Preserve `cwk` packaging instead of copying `vivi`'s builder.
- [x] Select a Formula-only shared-tap PR with a repository-scoped GitHub App.
- [x] Record stable-only behavior, external target, credential scope, and
      non-goals in ADR 0004.
- [x] Leave Linux Homebrew and provenance as separate decisions.

## Implement

- [x] Update the stable Formula jobs to target `tasuku43/homebrew-tap` through
      an audited artifact and fresh token-bearing runner.
- [x] Add the App-token boundary without committing credentials.
- [x] Stage only the exact audited `Formula/cwk.rb`, after rejecting symbolic
      destination paths and an existing non-regular target.
- [x] Update release lint for target, scope, job separation, ordering, and PR
      conventions, including workflow/job field allowlists.
- [x] Update security, harness, public, release, and installation docs.

## Verify

- [x] Focused workflow/release lint passes. Evidence: on 2026-07-19,
      `bash -n`, ShellCheck, actionlint, `lint-release-workflow.sh`, all negative
      workflow mutations, repoguard unit tests, and security scope passed.
- [x] `task release:check` passes. Evidence: exact Go 1.26.5 run completed with
      `lint-release: OK` on 2026-07-19.
- [x] `task check` passes. Evidence: exact Go 1.26.5 full gate passed on
      2026-07-19, including hygiene, architecture, contract, test, security,
      vulnerability, release, and public profiles.
- [x] `task public:check` passes. Evidence: explicit 2026-07-19 run completed
      with `repoguard (public): OK` and `contractlint: OK`.
- [x] `git diff --check` passes. Evidence: no output on 2026-07-19.
- [x] Final status contains only understood changes. Evidence: nine modified
      workflow, documentation, release-lint, and repoguard files plus ADR 0004,
      this work packet, and two new release-workflow lint scripts; the worktree
      was clean before the change.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Required Actions secrets and App permissions are explicit.
- [x] First-release live Formula PR/install validation is an explicit follow-up.
- [x] Separate shared-tap README update is an explicit follow-up.
- [x] No tag, release, tap branch, or pull request was created by this change.
