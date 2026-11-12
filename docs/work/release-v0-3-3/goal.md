# Work Goal: Deliver an interim stable release

- Status: Active
- Retention: temporary
- Retention reason: None
- Governing contract: `docs/05_public_repository.md`, `docs/06_release.md`
- Review/delete trigger: Remove after the release, Formula review/merge, clean Homebrew installation, and durable evidence are complete.
- Owner: Main session
- Target: v0.3.3 at commit `65c84d8c482d5f65cba5a8fcc05303a74159cac0`
- Related ADRs: `docs/decisions/0004-shared-homebrew-tap.md`

## Outcome

Deliver already-merged v0.3.x improvements through the normal stable-release and Formula workflow while preserving the unfinished main feedback work for a later release.

## Acceptance criteria

- [ ] Exact release revision and annotated notes are reviewed; the stable tag does not already exist.
- [ ] All four canonical gates pass on that revision; required App installation scope and permissions are independently confirmed.
- [ ] Pre-publication archive installation and retired-help migration behavior are verified with isolated synthetic configuration.
- [ ] A stable release is published create-only; its five archives and checksum manifest are complete and valid.
- [ ] Its audited Formula PR is reviewed and merged, followed by a clean Homebrew install.
- [ ] The larger feedback task is not represented as completed by this interim release.
