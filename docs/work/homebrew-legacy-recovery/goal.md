# Work Goal: Recover a published stable Formula without changing release assets

- Status: Design
- Retention: temporary
- Retention reason: None
- Governing contract: `docs/00_theses.md`, `docs/02_architecture.md`, `docs/03_security_model.md`, `docs/04_harness.md`, `docs/05_public_repository.md`, `docs/06_release.md`
- Review/delete trigger: Promote accepted recovery rules and executable checks before completion; preserve any unique release observation only under an explicit evidence exception.
- Owner: Main session
- Target: Published v0.3.2 Homebrew rollout
- Related ADR: `docs/decisions/0004-shared-homebrew-tap.md`

## Outcome

Make a reviewed, reproducible Homebrew recovery possible for an already-published stable release whose tagged Formula has become un-auditable solely because it redundantly declares the version encoded in both release URLs. Preserve the original immutable release and the existing audit-before-tap-write boundary.

## Non-goals

- Replace assets, move tags, change published release metadata, bypass audit, widen GitHub App permissions, or use the procedure to make arbitrary Formula edits.
- Claim the full feedback release or Homebrew rollout complete before its own required review, merge, installation, and final gates.

## Acceptance criteria

- [ ] Review the necessary change to exact-revision recovery policy before implementing any alternate mechanism.
- [ ] The exact stable tag and all six published release files are verified against the complete five-entry checksum manifest.
- [ ] The starting Formula is rendered by the exact tag's own source. A deterministic, fail-closed transformation removes only one redundant version line if both URLs encode exactly the same release tag; any other change is rejected.
- [ ] A real strict Homebrew audit is run on the exact transformed bytes and their digest is compared with the proposed Formula. The shared tap change is limited to its Formula and independently reviewed.
- [ ] Required security, public, and release checks pass; the owner verifies the old `--format agent` migration before installation. No live user installation or external proposal is claimed until it occurs.
