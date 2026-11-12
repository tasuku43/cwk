# Plan: Same-minor Go patch update proposals

## Chosen approach

1. Add a standard-library-only `tools/gopatch` program. Its production entry point reads the exact `go` directive, fetches only `https://go.dev/dl/?mode=json` with a fixed timeout and maximum body size, strictly parses the official stable-release response, and selects the numerically greatest patch sharing the directive's major/minor. It rewrites only the exact `go` directive.
2. Keep candidate application separately callable for the validation job. It accepts only a strict numeric version that is newer within the unchanged major/minor and updates the one directive; it has no source override or arbitrary file paths in its workflow-facing interface.
3. Add Go unit tests using an injected in-memory HTTP server for equal, older, newer, other-minor, malformed/duplicate/trailing/unknown-field, oversized, bad-status, timeout/network-failure, malformed-current-version, and no-unrelated-file-change cases.
4. Add `.github/workflows/go-patch.yml` with a weekly schedule and manual dispatch. Read-only jobs pin the workflow commit SHA, detect/reconstruct the candidate, install the exact candidate toolchain, and run `task check`, `task security`, and `task release:check`. Only after all gates pass does a narrowly scoped write job verify that candidate and base SHA again, stage only `go.mod`, and create or update one stable PR branch.
5. Reuse the existing pinned `peter-evans/create-pull-request` action with `GITHUB_TOKEN` limited to Contents and Pull requests write. Do not auto-merge. The existing Homebrew App scope remains unchanged. Because GitHub does not automatically run normal PR workflows from this token without approval, explicitly validate the exact generated candidate before PR creation and state the head-check limitation in durable docs.
6. Add an exact workflow lint plus negative mutation tests to the release/security checks, covering trigger, pinned actions, permissions, job dependencies, immutable SHA binding, exact candidate, staged path, no PR checkout/execution, and no auto-merge.
7. Remove toolchain-version-specific wording from `THIRD_PARTY_NOTICES` and derive/check notice files against the exact selected GOROOT. Bump `go.mod` and exact fake-toolchain fixtures to `1.26.10`; keep the notice content independently verified.

## Alternatives considered

- Extend the Homebrew GitHub App to `cwk`: rejected because ADR 0004 intentionally restricts its installation to the external tap repository.
- Use a personal token or broaden the App to trigger CI automatically: rejected because it adds a new credential and authorization boundary.
- Dispatch CI on the bot PR ref with `actions: write`: rejected because GitHub loads workflow code from the selected ref, which would make branch-controlled workflow content executable.
- `pull_request_target`: rejected because it runs with the base repository's privileged context while receiving untrusted PR data.
- Depend only on actionlint or grep: insufficient to assert the end-to-end safe order and PR file scope; retain focused contract tests and negative fixtures.

## Risks and mitigations

- The Go release list is remote and may be malformed or drift. Bound bytes/time, require strict JSON/schema, reject duplicate keys and duplicate versions, validate every version and file record, and fail closed without changing files.
- A candidate may be published but unavailable to setup-go. The exact-toolchain setup or required gates fail before PR write.
- `go.mod` may be changed concurrently after workflow selection. Bind all jobs to the same immutable SHA and make the write job check its base identity and generated one-file diff.
- `GITHUB_TOKEN` PR CI may require manual approval. Run all required gates in a read-only validation job on the exact generated candidate before writing the PR, and document that this does not substitute for an attached required check if branch protection requires one.
- Go patch notices may change. The release gate compares exact candidate GOROOT license/patent bytes with the checked-in notice; drift blocks the automated PR until a reviewed notice update is made.

## Verification

- `go test ./tools/gopatch`
- `scripts/test-go-patch-workflow.sh`
- `task check`
- `task security`
- `task release:check`
- Inspect the resulting tree and prove the only current Go floor transition is `1.26.9` to `1.26.10`; generated update PRs remain `go.mod`-only.
