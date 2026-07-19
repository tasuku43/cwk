# Work Plan: Release cwk v0.1.1

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Commit the complete understood working tree directly to `main` as the patch
source, including the maintainer's Homebrew 6 README refinement. Run every
required profile on that state, push the commit, and wait for exact-commit
GitHub CI. Preview generated notes, confirm the external Homebrew App maximum,
then create and push one annotated `v0.1.1` tag. Monitor immutable Release
publication, verify all assets, inspect the Formula-only tap pull request, and
perform a clean install after merge.

## Alternatives considered

### Document an `XDG_CONFIG_HOME` workaround only

Rejected because the adapter can deterministically support the common dotfiles
layout while retaining stricter validation on the actual owned targets.

### Publish a prerelease first

Rejected because the patch is covered by deterministic boundary tests and the
requested outcome includes the stable Formula path, which prereleases skip.

### Reuse or amend v0.1.0

Rejected because released tags and assets are immutable. Changed source bytes
require a new patch version.

## Design

### Public contract

`v0.1.1` is backward compatible for existing valid stores and expands one
local prerequisite: the configuration home may be a symbolic-link alias to a
real directory. Leaf-target restrictions, structured failures, outputs, and
exit codes remain unchanged. Release notes must state the root cause, retained
security boundary, Homebrew installation guidance, and absence of migration.

### Data and control flow

```text
reviewed clean main commit
  -> local full/security/release/public profiles
  -> exact-commit GitHub main CI
  -> generated-note preview + App maximum confirmation
  -> annotated v0.1.1 tag
  -> exact-revision archives + checksums + GitHub Release
  -> audited Formula-only shared-tap pull request
  -> tap merge
  -> clean Homebrew install
```

### Error and cancellation behavior

Stop before tagging if any local profile, remote CI, note review, or App
precondition fails. Once the Release exists, do not overwrite assets or move
the tag. A Formula-only rollout failure may be retried only when artifact and
Formula identity are unchanged.

### Security and public boundary

Inspect secret names only. Public guards and manual diff review cover source
content. The release workflow keeps source-repository and tap permissions
separate; the external App installation maximum remains a human-confirmed
pre-tag condition.

## Implementation slices

1. Record release target, included source, compatibility, and preconditions.
2. Run local required profiles and review the final diff.
3. Commit and push the exact source to `main`; wait for GitHub CI.
4. Preview notes and confirm App scope; create and push the annotated tag.
5. Verify Release assets, Formula PR, merge, and clean installation.

## Verification

- Local: `task check`, `task security`, `task release:check`,
  `task public:check`, and `git diff --check`.
- Remote pre-tag: successful GitHub CI for the exact commit.
- Release: five archives plus `checksums.txt`, checksum recomputation, exact tag
  and commit metadata, and generated notes.
- Shared tap: one `Formula/cwk.rb` diff with reviewed branch/title/base.
- Rollout: clean `brew install tasuku43/tap/cwk` after merge.

## Rollout and rollback

There is no rollback by tag mutation. Stop before publication when possible.
After publication, recover unchanged external Formula automation in place;
changed source or artifacts require a later patch release.

## Documentation promotion

The fix decision is already promoted to product, architecture, security,
harness, and README documents. Existing release and shared-tap policy remains
governed by `docs/05_public_repository.md`, `docs/06_release.md`, and ADR 0004.
