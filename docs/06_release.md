# Release Model

The base template defines byte-for-byte reproducible archives within a pinned pure-Go build contract and a public, reproducible-enough overall release path without private package infrastructure. A derived project must review supported platforms, artifact signing, provenance, package managers, and compatibility promises before its first release.

## Version contract

Release tags use Semantic Versioning with a leading `v`:

```text
vMAJOR.MINOR.PATCH
vMAJOR.MINOR.PATCH-PRERELEASE
```

Examples are `v1.2.3` and `v1.2.3-rc.1`. Stable tags may update stable package-manager metadata. Prerelease tags publish prerelease artifacts but do not replace the stable Homebrew Formula.

`go run ./tools/releaseversion <tag>` is the single validator used by local packaging and the release workflow. It implements SemVer 2.0 identifier rules, including rejection of leading zeroes in numeric prerelease identifiers. The repository release policy excludes SemVer build metadata even though the SemVer grammar permits it: tags with equal precedence must not identify different immutable artifact sets. Use a new patch or prerelease version instead.

The binary reports the embedded version and commit as:

```text
cwk <version> (<commit>)
```

Release checks verify that the displayed values match the tag and source revision.

`task build` is deliberately unversioned and retains the compiled `dev` default. It does not interpolate `git describe`, a tag name, or another repository-controlled string into a shell command. Only `scripts/package-release.sh` injects version and revision metadata, after the shared release-tag validator and full lowercase commit-SHA validation succeed.

## Default platform matrix

The template builds with `CGO_ENABLED=0` for:

| Operating system | Architecture | Archive |
|---|---|---|
| Linux | `amd64` | `.tar.gz` |
| Linux | `arm64` | `.tar.gz` |
| macOS | `amd64` | `.tar.gz` |
| macOS | `arm64` | `.tar.gz` |
| Windows | `amd64` | `.zip` |

Archive names and bytes are deterministic for identical source, tag, full revision, target, and exact Go toolchain:

```text
<binary>_<tag>_<os>_<arch>.tar.gz
<binary>_<tag>_windows_amd64.zip
```

The tag retains its leading `v`, and architectures keep Go-native names such as `amd64` and `arm64`.

A derived project may change this matrix only after updating supported-platform documentation, packaging checks, installation instructions, and package-manager metadata together.

## Published release contents

Each tag publishes:

- one archive for every supported platform tuple;
- `checksums.txt` containing cryptographic checksums for all archives;
- reviewed release notes from the annotated tag describing user-visible
  changes, compatibility, security, and migration impact;
- prerelease metadata when the tag contains a prerelease suffix.

Archives contain exactly the intended binary, the project `LICENSE`, and the checked-in `THIRD_PARTY_NOTICES` artifact. `scripts/package-release.sh` builds and verifies artifacts; `task release:check` validates the packaging contract, verifies the notice text against the exact pinned Go toolchain license and patent-grant sources plus the reviewed dependency-module licenses, and fails closed if the modules linked by any supported target differ from that reviewed notice manifest.

The packaging command is create-only. It stages and inspects an archive in a temporary directory on the output filesystem, then publishes it with an atomic no-overwrite hard link. It refuses to overwrite an archive that already exists or appears during the build.

Archive creation uses the Go standard library rather than host-specific `tar`, `gzip`, or `zip` creation flags. Every archive contains exactly three regular entries in bytewise basename order: `LICENSE` and `THIRD_PARTY_NOTICES` with mode `0644`, then the executable with mode `0755`. Every entry has the same fixed UTC modification time; tar entries have empty user and group names and numeric user and group IDs of zero. Gzip and ZIP headers use the canonical time and contain no build-host identity. The packaging and release gates reopen each completed archive through the Go archive readers and verify the exact ordered names, header modes, canonical metadata, and reviewed contents rather than trusting caller arguments or extraction behavior. The packaging boundary forces module mode; fixes `GOAMD64` or `GOARM64` at the portable baseline; sets `GOFIPS140=off`; ignores ambient Go workspace, toolchain, experiment, and flag configuration; and disables implicit Go VCS stamping because the reviewed full revision is already embedded explicitly. Repository release inputs—including production and tool source, `LICENSE`, `THIRD_PARTY_NOTICES`, packaging and Formula policy, workflow configuration, project metadata, and the Codex harness—must be regular files and are content-fingerprinted through the final release check; dependency modules are verified around each archive pass; and local filesystem module replacements are rejected because their source would sit outside the public release input boundary. The exact Go version in `go.mod`, `-trimpath`, the source bytes, tag, revision, target, and verified module graph are part of the reproducibility input. This contract does not promise equal bytes across different Go versions or establish who performed a build.

## Executable release profile

`task release:check` performs two independent complete local matrix package passes, not a host-only approximation. It compares each corresponding digest, then reuses the primary five archives for every remaining check. The profile:

1. requires the exact Go toolchain selected by `go.mod`;
2. independently builds Linux `amd64` and `arm64`, macOS `amd64` and `arm64`, and Windows `amd64` twice with `CGO_ENABLED=0` and separate Go build caches;
3. fingerprints release inputs before and after each pass and after the remaining release checks, reports source drift separately, and proves byte-for-byte reproducibility by comparing every corresponding archive digest across the two output directories;
4. verifies the exact archive set and the three expected members, canonical order, header modes, canonical metadata, project license, and notice contents in every primary archive;
5. extracts every primary archive and checks the executable's Go module, `GOOS`, and `GOARCH` build metadata;
6. creates `checksums.txt`, proves it has a one-to-one correspondence with all five primary archives, and recomputes every digest;
7. positively renders a stable Homebrew Formula from the real macOS archive checksums and verifies its URLs, digests, version, class, and placeholder removal;
8. runs `ruby -c` against the rendered Formula;
9. exercises the isolated-tap ownership test for Formula audit cleanup; and
10. validates the stable workflow's exact `tasuku43/homebrew-tap` destination,
    scoped GitHub App secret inputs and requested permissions, read-only audit
    job source token, zero source-repository permissions in the token-bearing
    job, non-persisted checkouts, audited artifact handoff
    to a fresh publish runner with no tagged-source checkout or checked-out
    code execution, reviewed workflow and Formula-job fields without ambient
    `env` or `defaults`, non-symbolic Formula destination paths, Formula-only
    staging, accepted pull-request conventions, workflow-wide single
    action/secret occurrences, the complete Formula-job step lists, and
    render/audit-before-cross-repository-write
    ordering; and
11. runs negative workflow mutations that must reject broader repositories or
    permissions, another PR token/path/base, wildcard staging, persisted tap
    credentials, an alternate/extra step, missing audit-job dependency,
    tagged-source checkout or checked-out code execution in the token-bearing
    runner, workflow- or job-level runtime injection, symbolic Formula
    destinations, an App action or secret outside its reviewed step, an extra
    token consumer, changed audited Formula binding, ignored audit/PR failure,
    or an incompatible title.

The profile requires `tar`, `unzip`, either `sha256sum` or `shasum`, ShellCheck `0.9.0` or newer, and Ruby. Archive creation and canonical header verification themselves have no host `zip` dependency. ShellCheck covers every publishable `.sh` file rather than a hand-maintained subset. It is a system prerequisite with an explicit compatibility floor, not an exact repository pin: the floor accepts the `0.9.0` analyzer supplied by the documented Linux runner and newer compatible analyzers such as `0.11.x`. A missing or older ShellCheck, or a missing Ruby executable, is a failed release check rather than a skipped check. A developer without these tools must use the documented CI release gate and treat its result as required evidence before tagging.

The workflow runs this canonical release profile once inside the Ubuntu preflight's full gate. The later macOS Formula job is deliberately narrower: it renders the checksum-pinned Formula, runs `ruby -c`, performs the real Homebrew strict audit, and uploads only that Formula. It does not repeat `check.sh release`, because that would rebuild the complete five-target verification matrix on a different host and would incorrectly make Formula publication depend on Linux preflight tools such as ShellCheck being installed on the macOS runner. The audit job consumes only artifacts produced after the preflight and build jobs succeed and receives no App credential. A dependent fresh Ubuntu runner downloads and validates the Formula as data before minting the App token; it checks out no tagged source and executes no checked-out source or Formula content. Static release lint fixes both job boundaries and proves that failures cannot be ignored.

## Release workflow

The release workflow follows this order:

1. Validate tag syntax and resolve its exact source revision.
2. Run source, security, release, and public-boundary gates required by policy.
3. Build the complete pure-Go platform matrix from that revision.
4. Verify archive names, canonical order and header modes, reviewed contents, executable behavior, version, and commit.
5. Generate and verify `checksums.txt`.
6. Publish one GitHub Release from the reviewed tag, using the annotated tag
   message unchanged as its reviewed notes.
7. For a stable tag, render and strictly audit the checksum-pinned Homebrew
   Formula without an App credential, then upload that one-file workflow
   artifact.
8. On a fresh runner with no tagged-source checkout or checked-out code
   execution, validate the Formula artifact as data and open a Formula update
   pull request in `tasuku43/homebrew-tap`.

If GitHub Release publication succeeded but the Formula stages did not, a
maintainer may dispatch the same `Release` workflow with the existing stable
tag. That recovery path checks out the tag only in the read-only preflight,
runs the canonical full gate, downloads the already-published six-file asset
set, requires a non-draft/non-prerelease Release with the exact tag, verifies
the exact filenames and all five checksums, and uploads only `checksums.txt` as
Formula input. It never creates, uploads, replaces, or deletes Release assets.
The normal exact-revision Formula audit and fresh App-token runner then resume.

The workflow uses a public GitHub Release path. It must not embed private asset URLs, personal access tokens, authorization headers, or organization-specific package infrastructure in Formula content.

Publication is create-only. If a GitHub Release already exists for the tag, the workflow fails without uploading or replacing any asset. It never uses `--clobber`. Correct a failed or incorrect release with a new version and an explicit incident or withdrawal decision; do not silently rewrite published evidence.

Release notes are likewise selected before publication. The workflow requires
an annotated tag and passes `--notes-from-tag`; it rejects generated-note
substitution because a direct reviewed commit may otherwise collapse to only a
comparison link. The release owner reviews the exact annotation before tag
push. The publication job checks out that exact tag without persisted
credentials before asking GitHub CLI to read its local annotation. A lightweight tag or an annotation that omits included changes,
compatibility, security, or migration impact is not release-ready.

Workflow checkouts disable persisted Git credentials. The Formula pull-request
action receives only a short-lived GitHub App installation token restricted to
`tasuku43/homebrew-tap`. The audit job's source-repository `GITHUB_TOKEN`
remains Contents-read-only, while the token-bearing publish job has no
source-repository `GITHUB_TOKEN` permissions. The App token comes from
GitHub Actions secrets `HOMEBREW_APP_ID` and `HOMEBREW_APP_KEY`; their values
never enter source, artifacts, Formula content, or release notes.

Every matrix build checkout and Formula generation step is bound to
`needs.preflight.outputs.revision`, not an implicit event checkout or a moving
branch. The workflow renders and strictly audits the Formula from that exact
release revision and its project metadata into runner-owned temporary storage.
After audit succeeds, the macOS job uploads only the audited Formula. Its
dependent fresh runner validates that artifact's exact identity without
executing it, then mints the tap-scoped token, checks out the shared tap's
current `main`, copies the Formula to `Formula/cwk.rb`, and opens the reviewable
pull request. Changes to source identity, templates, or generation scripts
after the tag therefore cannot race the artifacts and checksums used as
generation input.

The manual recovery input is accepted only as a stable SemVer tag, is passed to
shell through a quoted environment value, and resolves to the same immutable
peeled commit with `git rev-parse --verify` in preflight. Recovery is deliberately limited to the Homebrew
rollout: it cannot be used to republish GitHub Release assets or to update a
prerelease Formula.

The GitHub App must be installed only on `homebrew-tap` with Contents
read/write plus Pull requests read/write there, and the release owner confirms
that external state before a stable tag. Token creation fixes owner
`tasuku43` and repository `homebrew-tap`, then requests exactly Contents write
and Pull requests write as its write-capable permissions; the audit job grants
the source-repository workflow token only Contents read and the token-bearing
job grants it no permissions. Secret
provisioning and external App permission review are release-owner prerequisites
rather than repository content. [ADR 0004](decisions/0004-shared-homebrew-tap.md)
records this boundary.

## Homebrew contract

Stable releases support macOS `arm64` and `amd64` through a generated Formula. The Formula:

- selects the archive matching the user's CPU;
- uses the public release URL;
- pins the exact checksum;
- installs the binary without cloning source or requiring a Go toolchain;
- installs the project license and third-party notices under the package documentation prefix;
- contains no unreplaced template value or private authentication behavior.

`scripts/render-formula.sh` renders the project Formula from the reviewed
template. Formula changes are proposed to the shared public
`tasuku43/homebrew-tap` repository through a pull request so the generated diff
and release references are visible before merge. The branch begins
`chore/homebrew-formula-v`, includes the `cwk` binary suffix to avoid
same-version collisions with another tool, and the title begins
`Update Homebrew formula for v` as required by the tap automation. The staged
path is exactly `Formula/cwk.rb`; tap README changes use a separate reviewed
change because automated Formula pull requests remain Formula-only.

`scripts/audit-formula.sh` creates a collision-resistant temporary tap name from `mktemp`, verifies that name is not already installed, and records ownership only after `brew tap-new` succeeds. Its exit trap removes only that owned tap. It never pre-emptively untaps a fixed name, so an existing user tap is outside its cleanup authority. After copying the rendered Formula into the owned tap, it fixes that copy to non-executable mode `0644` before strict audit; the source Formula remains unchanged. The release profile tests ownership cleanup on audit success and failure and starts from a `0600` fixture to prove runner umask cannot violate Homebrew's readability contract.

Prereleases do not update stable Formula metadata.

The workflow ends after proposing the Formula. For a stable release, the
release owner separately waits for the tap pull request to merge, performs a
clean `brew install tasuku43/tap/cwk`, and records that post-merge rollout
evidence before announcing Homebrew availability. Pre-tag clean-install
evidence covers installation paths available before publication; it cannot
prove a Formula that does not exist in the tap until after GitHub Release
publication and PR merge.

## Release preparation

Create a work packet for a release and record:

- target version and rationale;
- included changes and compatibility impact;
- security fixes and disclosure coordination;
- migration or deprecation notes;
- the exact annotated-tag release notes that summarize those decisions;
- required profiles and their results;
- pre-publication clean-environment evidence for installation instructions
  that do not depend on the post-publication Formula;
- for a stable release, post-merge clean Homebrew installation evidence before
  declaring the shared-tap rollout complete;
- artifact and checksum verification;
- public-boundary review.

Before tagging, run:

```sh
task check
task security
task release:check
task public:check
```

Before a stable tag, also confirm in GitHub settings that
`HOMEBREW_APP_ID`/`HOMEBREW_APP_KEY` are configured and that the App
installation is restricted to `homebrew-tap` with Contents read/write and Pull
requests read/write. Confirm that token creation requests only Contents write
and Pull requests write. Record only that the review occurred, never either
secret value. Then review the exact commit that will receive the tag. A clean
local run does not authorize tagging a different revision.

## Failure and recovery

- If preflight or any matrix build fails, publish nothing.
- If the artifact set is incomplete, do not create a partial stable release.
- If a checksum or Formula reference is wrong, correct it through a reviewed replacement release or metadata change; do not silently mutate evidence.
- If the shared-tap job fails after GitHub Release publication, do not replace
  the release assets. Correct the App/tap condition and rerun the failed job, or
  dispatch `gh workflow run release.yml -f tag=<stable-tag>` to revalidate the
  immutable published inputs and resume the same App-scoped Formula path. A
  manually proposed Formula is acceptable only when it is byte-identical to
  the audited result and receives the same tap review. Change the release
  version when artifact identity must change.
- If the tag already has a GitHub Release, stop. Do not overwrite its assets or rerun publication as an update operation.
- If sensitive content reaches a release, stop distribution, revoke affected credentials, preserve incident evidence, and follow the security response process.
- Do not reuse a stable version for different source or artifacts.

## Signing and provenance

The base template does not claim code signing, notarization, or externally verifiable build provenance. Checksums detect accidental corruption after a trusted release is selected; they do not establish who produced it.

A derived project that needs stronger guarantees must document and test:

- signing identity and key protection;
- verification instructions;
- provenance format and builder trust;
- rotation and revocation;
- platform-specific installation consequences.

Absence of these controls must remain visible in the security model and release notes rather than being implied away.

## Derived-project release decisions

Before the first project-specific tag, decide:

1. Which platforms are supported and tested?
2. When does compatibility become stable?
3. Are prereleases allowed, and who may create them?
4. Which package managers are maintained?
5. Are signing, notarization, SBOMs, or attestations required?
6. How are vulnerable or broken releases withdrawn?
7. Which workflow permissions and environments protect publication?
8. Who performs the manual public-boundary review?

Record durable trade-offs in an ADR and update `task release:check` so the chosen contract is executable.

For this project, ADR 0004 selects stable macOS Formula publication through
the shared `tasuku43/homebrew-tap` pull-request path and a repository-scoped
GitHub App. Prerelease Formula updates, Linux Homebrew support, signing,
notarization, SBOMs, and attestations remain outside that decision.
