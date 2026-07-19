# Work Plan: Release cwk v0.1.0

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Build on the maintainer's reviewed, already-pushed shortened README commit
`5f6c03c`. Commit the shared-tap implementation, Concept A `config` result,
their enforcement and work packets, and this release packet directly to
`main`. Run every required profile, wait for the exact resulting `main`
commit's GitHub CI, then create and push one annotated `v0.1.0` tag. Monitor
the tag workflow without rewriting assets. Inspect the resulting Formula-only
tap pull request and verify Homebrew after merge.

## Alternatives considered

### Publish v0.1.0-rc.1 first

Rejected for this rollout because prerelease routing intentionally skips the
Formula jobs and would not test the shared-tap credential and pull-request
boundary. It remains appropriate for a future artifact-only release rehearsal.

### Publish v0.0.1

Rejected because `v0.1.0` better communicates the first usable public feature
baseline while still reserving `v1.0.0` for a mature compatibility promise.

## Design

### Public contract

This is the first published version, so it establishes rather than migrates the
public binary, command, output, archive, checksum, and Homebrew contracts. The
release commit includes the accepted pre-publication `config` presentation and
scoped agent field refinement; the tag itself changes no command behavior
beyond selecting version metadata at package time.

Before tag push, preview GitHub's generated notes against the exact remote
commit and confirm that their included changes, compatibility, security, and
migration framing agrees with this packet. The release owner separately
approves the public boundary because automated scanning cannot decide
confidentiality context, ownership, trademark, or human readiness.

### Data and control flow

```text
reviewed clean main commit
  -> local full/security/release/public gates
  -> GitHub main CI
  -> annotated v0.1.0 tag
  -> exact-revision release workflow
  -> immutable archives + checksums + GitHub Release
  -> audited Formula-only shared-tap pull request
  -> tap merge
  -> clean Homebrew install verification
```

### Error and cancellation behavior

Stop before tagging if a local profile or `main` CI fails. After a GitHub
Release exists, never move or reuse the tag and never replace assets. A later
tap failure is recovered by correcting the external App/tap condition and
rerunning the failed job or proposing the identical audited Formula through a
reviewed path.

### Security and public boundary

Inspect only secret names, never values. Confirm the App installation maximum
before the stable tag. The source workflow token cannot write the tap; only the
short-lived installation token on the isolated publish runner can propose the
exact Formula path.

## Implementation slices

1. Record first-release target, prerequisites, recovery, and evidence fields.
2. Run local release profiles against the final source tree.
3. Commit and push the exact source state to `main`.
4. Wait for GitHub `main` CI, then create and push annotated `v0.1.0`.
5. Inspect Release artifacts and Formula PR; verify post-merge installation.

## Verification

- Required local profiles: `task check`, `task security`,
  `task release:check`, and `task public:check`.
- Remote pre-tag check: successful GitHub CI for the exact commit.
- Release check: five archives plus `checksums.txt`, correct tag and commit,
  and no replaced existing Release.
- Release-notes check: generated notes previewed for the exact remote commit
  before tag creation.
- Shared-tap check: one `Formula/cwk.rb` diff with accepted author, base,
  branch, and title.
- Rollout check: clean `brew install tasuku43/tap/cwk` after merge.

## Rollout and rollback

There is no version rollback by mutation. Stop before tag push when possible;
after publication, fix external Formula automation in place only when artifact
identity is unchanged, otherwise publish a new patch version. Withdrawal or a
security incident follows the release and security policies.

## Documentation promotion

The durable release, shared-tap, credential, and recovery decisions already
live in `docs/03_security_model.md`, `docs/05_public_repository.md`,
`docs/06_release.md`, and ADR 0004. This packet owns only `v0.1.0` evidence.
