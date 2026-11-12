# Tasks: Same-minor Go patch update proposals

- [x] Record the initial version, existing exact-toolchain checks, and current App/token boundaries in this packet.
- [x] Implement strict fixed-source Go stable patch selection and a go.mod-only rewrite.
- [x] Add offline synthetic tests for comparison, invalid API data/network, bounds, and unchanged unrelated files.
- [x] Bump the checked-in Go minimum to `1.26.10` and update exact fake-toolchain fixtures.
- [x] Make release notice verification compatible with future one-file Go patch PRs while retaining byte-exact license and patent checks against the exact candidate toolchain.
- [x] Add weekly/manual proposal workflow with pinned actions, read-only validation jobs, exact SHA/candidate binding, and isolated PR write permissions.
- [x] Add workflow contract tests and negative mutations for permissions, untrusted code, ordering, and PR file scope.
- [x] Provide a Japanese contributor-facing explanation of same-minor updates and the GITHUB_TOKEN approval/check limitation through the active documentation owner.
- [ ] Run required gates and relay evidence/design wording to the documentation owner.
- [ ] Delete this temporary work packet after durable documentation is promoted and the parent review completes.
