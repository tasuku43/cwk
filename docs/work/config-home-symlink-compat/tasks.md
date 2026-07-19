# Work Tasks: Config-home symlink compatibility

## Understand

- [x] Read governing theses, product, architecture, security, and harness.
- [x] Trace the reported failure to the final base-directory `Lstat`.
- [x] Record current behavior, constraints, and thesis evidence.
- [x] Confirm `v0.1.1` as the target and release publication as a non-goal.

## Decide

- [x] Compare environment workaround, broad link support, and resolve-once
  compatibility.
- [x] Keep the `cwk` directory and preference file as strict non-link targets.
- [x] Confirm no command/catalog/capability/authentication contract changes.
- [x] Confirm no ADR or thesis revision is required.

## Implement

- [x] Add positive base-alias and negative broken-alias tests.
- [x] Resolve an existing configuration-home alias once to an absolute target.
- [x] Preserve missing-base creation and owned-target protections.
- [x] Update README and durable governing documents.

## Verify

- [x] Focused commandconfig tests pass. Evidence: `go test
  ./internal/infra/commandconfig` passed on Go 1.26.5.
- [x] `task check` passes. Evidence: full gate passed on Go 1.26.5, including
  race, security, release lint, public guard, and contract lint stages.
- [x] `task security` passes. Evidence: module verification, security guard, and
  `govulncheck` reported no called vulnerabilities.
- [x] `task public:check` passes. Evidence: public guard and contract lint passed.
- [x] Runtime behavior is covered on a temporary Darwin/Unix filesystem.
  Evidence: `TestProductionStoreResolvesSymlinkedXDGConfigHome` exercised the
  production resolver and a relative configuration-home alias on Darwin.
- [x] Repository diff and pre-existing README edits are understood. Evidence:
  the existing Homebrew tap-trust change remains intact and the new guidance is
  additive below it.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions are promoted out of this packet.
- [x] No temporary diagnostics or sensitive artifacts remain.
- [x] Release/tag creation remains explicit follow-up work.
