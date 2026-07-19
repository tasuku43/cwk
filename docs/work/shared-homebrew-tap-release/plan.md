# Work Plan: Publish stable Formula updates through the shared tap

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Keep the existing exact-revision release and Formula render/audit path. After
audit, upload only the Formula as a workflow artifact. On a fresh runner that
checks out no tagged source and executes no checked-out source or Formula
content, validate that artifact as data, mint a repository-scoped GitHub App token,
check out the shared tap without persisting credentials, copy only the audited
Formula, and use the existing pinned pull-request action with that token and
tap path.

## Alternatives considered

### Copy the `vivi` release builder and updater

Rejected because its artifact names, archive members, platform matrix,
version policy, and provenance decisions differ from `cwk`'s reviewed
contracts.

### Push directly to tap `main`

Rejected because it bypasses the tap pull-request boundary and makes the
cross-repository mutation less reviewable.

### Keep the Formula in the source repository

Rejected because it does not provide the requested common installation tap.

## Design

### Public contract

No CLI contract changes. Stable GitHub Releases become installable through
`brew install tasuku43/tap/cwk` after the Formula PR merges. Prereleases remain
GitHub-Release-only.

### Layer changes

- Domain, application, infrastructure, CLI and catalog: none.
- Release workflow: change the post-audit Formula handoff, destination, and
  token boundary.
- Harness: assert the shared-tap repository, scope, file, ordering, and PR
  conventions.
- Documentation: record the security/release decision and installation path.

### Data and control flow

```text
stable tag + exact revision
  -> full gate and canonical release archives
  -> create-only GitHub Release + checksums
  -> render and strict-audit Formula/cwk.rb from tagged source (no App secret)
  -> one-file workflow artifact
  -> fresh runner: validate artifact as data (no tagged-source checkout or execution)
  -> tap-scoped GitHub App token
  -> credential-free checkout of tasuku43/homebrew-tap/main on the fresh runner
  -> stage only Formula/cwk.rb
  -> pull request through the scoped token
```

### Error and cancellation behavior

Any render, syntax, audit, artifact, token, checkout, staging, or pull-request
failure fails its job and prevents downstream Formula publication. No workflow
step overwrites GitHub Release assets or pushes directly to tap `main`.
Correcting App secret configuration permits a failed-job rerun; changing
published artifacts requires a new version.

### Security and public boundary

The audit job's source-repository `GITHUB_TOKEN` remains Contents-read-only;
the token-bearing Formula publish job has no source-repository permissions, and
the separate GitHub Release publish job keeps its existing write boundary. The
audit job has no App credential. The derived installation token exists only on
a fresh runner with no tagged-source checkout or
checked-out code execution and is supplied only to the tap
checkout/pull-request boundary. The staged file is exact and audited; no
wildcard stages another tap file. Workflow and Formula-job field allowlists
reject ambient runtime injection, and staging fails if the tap's Formula
directory or target is symbolic or an existing target is not regular.

## Implementation slices

1. Record ADR, work packet, and external facts.
2. Change the workflow destination and token boundary.
3. Update release lint to make the new claims executable.
4. Update durable security/release/public documentation and README.
5. Run release, full, and public gates and inspect the final diff.

## Verification

- Workflow syntax and action policy: `task release:check`.
- Formula render, checksums, Ruby syntax, workflow mutation checks, and fake
  Homebrew audit-boundary exercise: `task release:check`. The real Homebrew
  strict audit remains in the macOS Formula job before token creation.
- Repository and public boundary: `task check`, `task public:check`.
- Live App/tap mutation: deliberately deferred to the first stable release;
  no tag or external write is authorized in this change.

## Rollout and rollback

Configure the App and both Actions secrets before the first stable tag. The
first stable release verifies the opened tap PR and clean Homebrew install.
Rollback changes the workflow through review; it never removes or overwrites
published artifacts. A bad Formula is corrected by a reviewed tap change or a
new release according to whether artifact identity changed.

## Documentation promotion

- ADR 0004 owns the shared-tap and credential decision.
- `docs/03_security_model.md` owns the token and mutation boundary.
- `docs/04_harness.md` owns mechanical enforcement.
- `docs/05_public_repository.md` and `docs/06_release.md` own publication and
  operating procedure.
- `README.md` owns the Japanese installation path.
