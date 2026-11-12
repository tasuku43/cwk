# Context: Same-minor Go patch update proposals

## Verified facts

- At work start, `go.mod` declares `go 1.26.9`, the local Go command is `go1.26.9 darwin/arm64`, and the release notice embeds `go1.26.9` in its Go-runtime heading.
- `scripts/lint-release.sh` hardcodes the same Go notice heading, and `scripts/testdata/fake-go-gate-environment.sh` defaults to `go1.26.9`.
- `scripts/check.sh` reads the module `go` directive and allows the PATH-selected toolchain when it is numerically greater than or equal to that version. The release profile separately requires the exact toolchain from `go.mod`.
- During local candidate-toolchain validation, a preexisting `GOROOT` pointing at Go `1.26.7` overrode the extracted Go `1.26.10` tree and caused standard-library compiler-version mismatches. The canonical gate now unsets `GOROOT` and its synthetic environment test supplies an invalid `GOROOT` canary to prove it is removed before invoking Go.
- `task check` and `task security` are the normal implementation/dependency gates. `task release:check` also validates the exact Go toolchain and reproducibility notices.
- ADR 0004 and `docs/03_security_model.md` restrict the existing Homebrew GitHub App to `tasuku43/homebrew-tap`; adding `cwk` would expand the reviewed authorization boundary.
- GitHub documents that PR workflows created or updated with `GITHUB_TOKEN` require workflow approval; `workflow_dispatch` is an exception, but it executes workflow code at the selected ref. Dispatching the PR ref could therefore execute branch-controlled workflow code and is not used as the safety workaround.
- The official Go download endpoint `https://go.dev/dl/?mode=json` returns a JSON list of stable Go releases. As of 2026-11-07, it includes stable `go1.26.10` and `go1.27.3`; the selected current minor is `1.26`, so only `1.26.10` is eligible. The `go1.26.10.darwin-arm64.tar.gz` archive used for local verification matched the official published SHA-256 before extraction.

## Constraints

- Production workflow code and its proposed candidate are checked out from the same immutable default-branch SHA.
- Candidate metadata is external untrusted input, even though it arrives over HTTPS from the official Go host.
- The candidate-validation job has `contents: read` only and must run before a separate PR-writing job.
- The PR-writing job may use only the default-branch workflow and validated generated file state; it must not check out or execute PR-head code.
- GitHub actions are pinned to immutable revisions. No PAT or expanded GitHub App installation is introduced.
- The automatic PR contains only a one-line `go.mod` directive change, with manual review and merge.
- Exact release-notice checking must remain tied to the selected Go toolchain. To keep later patch PRs one-file-only, the checked-in notice label should be version-neutral while notice contents are still compared byte-for-byte with that exact toolchain's license and patent files.

## Unknowns

- GitHub repository branch-protection configuration may require head-attached CI contexts. A read-only pre-PR validation job proves the generated candidate but does not create those contexts on the PR head; GITHUB_TOKEN-triggered PR runs may remain approval-required.
- Whether future Go patch releases change license or patent text. Release validation intentionally fails closed if the checked-in notices cease to match.

## Evidence sources

- Go download JSON/API description: [official Go download endpoint](https://go.dev/dl/?mode=json) and [Go download package documentation](https://pkg.go.dev/golang.org/x/website/internal/dl).
- GitHub token-trigger behavior: [GitHub Actions `GITHUB_TOKEN` documentation](https://docs.github.com/en/actions/concepts/security/github_token) and [workflow dispatch documentation](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow).
