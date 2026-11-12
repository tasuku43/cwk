# Work Goal: Same-minor Go patch update proposals

- Status: Active
- Retention: temporary
- Retention reason: None
- Governing contract: `AGENTS.md` claims-to-check, security, automation, and completion rules; `docs/03_security_model.md` supply-chain boundary; `docs/04_harness.md` automated-update gate policy; `docs/06_release.md` exact toolchain and notices
- Review/delete trigger: Delete after durable policy is promoted and required gates pass
- Owner: Codex
- Target: Current work session
- Related ADRs: ADR 0001 (Go toolchain policy)

## Outcome

Once a week, the repository checks the official Go stable-release list for a newer patch in the exact minor already selected by `go.mod`. If one exists, it opens or reuses a reviewable pull request whose only change is the `go` directive, after the candidate tree passes the required checks under that exact Go toolchain.

## Why now

The repository's minimum Go version is `1.26.9`, while a newer stable same-minor patch is available. The project already states that automated updates must pass the same gates as human changes, but it has no bounded mechanism for proposing exact Go patch updates.

## Non-goals

- Automatically merge or approve an update.
- Advance the Go major or minor version, or accept prerelease/unstable versions.
- Update Go dependencies, GitHub Actions, release notices, or any file other than `go.mod` in the automated PR.
- Extend the Homebrew GitHub App beyond its reviewed `homebrew-tap` repository scope.
- Claim that the workflow-created PR head-attached CI checks are automatically approved by GitHub.

## Acceptance criteria

- [ ] The updater uses a fixed official HTTPS source, a finite timeout and body limit, and strict untrusted JSON validation.
- [ ] Only a numerically newer stable patch with the exact current major/minor is eligible; equal, older, other-minor, prerelease, malformed, oversized, and network-failure cases do not alter files.
- [ ] The updater changes only the `go` directive in `go.mod`; tests prove unrelated files remain unchanged.
- [ ] A weekly pinned-action workflow reconstructs one candidate against the exact default-branch SHA and runs full, security, and release gates with read-only repository permissions before any write-capable job.
- [ ] The write-capable job revalidates the target SHA, candidate version, and sole `go.mod` diff, creates or reuses one PR, stages only `go.mod`, and does not auto-merge or execute PR-head code.
- [ ] Workflow contract tests reject broader permissions, unpinned actions, alternate data sources, changed files, skipped candidate validation, and write-before-validation ordering.
- [ ] Required repository gates pass and durable security/harness/release documentation is promoted.

## Governing documents

- Thesis: `docs/00_theses.md` claims-to-check and automated update evidence
- Product contract section: Not applicable; no user-facing CLI capability
- Architecture or security invariant: `docs/03_security_model.md` supply-chain boundary and `docs/04_harness.md` automated updates
- Existing ADR: ADR 0001

## Completion definition

The work is complete when acceptance criteria have evidence, durable decisions are promoted to numbered documentation or an ADR, required profiles pass, temporary diagnostics or sensitive artifacts are removed, and this temporary packet is removed from the final tree.
