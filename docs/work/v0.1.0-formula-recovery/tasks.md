# Work Tasks: Recover v0.1.0 Homebrew Formula publication

## Implement

- [x] Confirm render and syntax passed before the audit failure.
- [x] Confirm no Formula PR step ran.
- [x] Normalize only the isolated tap Formula copy to `0644`.
- [x] Add a `0600` source-to-`0644` audit-boundary regression test.
- [x] Update harness and release contracts.

## Verify and publish

- [x] Focused audit tests and shell analysis pass.
- [x] `task release:check` and `task check` pass.
- [x] Published `v0.1.0` checksums are downloaded and verified.
- [x] The recovered Formula passes syntax and real strict audit.
- [ ] A shared-tap PR changes only `Formula/cwk.rb`.
- [ ] Record the PR URL and post-merge clean install result.
