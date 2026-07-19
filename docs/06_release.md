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
- generated release notes or reviewed notes describing user-visible changes;
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
8. runs `ruby -c` against the rendered Formula; and
9. exercises the isolated-tap ownership test for Formula audit cleanup.

The profile requires `tar`, `unzip`, either `sha256sum` or `shasum`, ShellCheck `0.9.0` or newer, and Ruby. Archive creation and canonical header verification themselves have no host `zip` dependency. ShellCheck covers every publishable `.sh` file rather than a hand-maintained subset. It is a system prerequisite with an explicit compatibility floor, not an exact repository pin: the floor accepts the `0.9.0` analyzer supplied by the documented Linux runner and newer compatible analyzers such as `0.11.x`. A missing or older ShellCheck, or a missing Ruby executable, is a failed release check rather than a skipped check. A developer without these tools must use the documented CI release gate and treat its result as required evidence before tagging.

The workflow runs this canonical release profile once inside the Ubuntu preflight's full gate. The later macOS Formula job is deliberately narrower: it renders the checksum-pinned Formula, runs `ruby -c`, and performs the real Homebrew strict audit. It does not repeat `check.sh release`, because that would rebuild the complete five-target verification matrix on a different host and would incorrectly make Formula publication depend on Linux preflight tools such as ShellCheck being installed on the macOS runner. The Formula job consumes only artifacts produced after the preflight and build jobs succeed.

## Release workflow

The release workflow follows this order:

1. Validate tag syntax and resolve its exact source revision.
2. Run source, security, release, and public-boundary gates required by policy.
3. Build the complete pure-Go platform matrix from that revision.
4. Verify archive names, canonical order and header modes, reviewed contents, executable behavior, version, and commit.
5. Generate and verify `checksums.txt`.
6. Publish one GitHub Release from the reviewed tag.
7. For a stable tag, render the checksum-pinned Homebrew Formula and open a Formula update pull request.

The workflow uses a public GitHub Release path. It must not embed private asset URLs, personal access tokens, authorization headers, or organization-specific package infrastructure in Formula content.

Publication is create-only. If a GitHub Release already exists for the tag, the workflow fails without uploading or replacing any asset. It never uses `--clobber`. Correct a failed or incorrect release with a new version and an explicit incident or withdrawal decision; do not silently rewrite published evidence.

Workflow checkouts disable persisted Git credentials. The Formula pull-request action receives only its explicitly scoped workflow token, while source checkout does not leave that token in Git configuration.

Every matrix build checkout and Formula generation step is bound to `needs.preflight.outputs.revision`, not an implicit event checkout or the moving `main` branch. The workflow renders and strictly audits the Formula from that exact release revision and its project metadata into runner-owned temporary storage. Only after audit succeeds does it check out current `main`, copy the already-audited Formula into `Formula/`, and open the reviewable pull request. Changes to identity, templates, or generation scripts on `main` therefore cannot race the tagged artifacts and checksums used as generation input.

## Homebrew contract

Stable releases support macOS `arm64` and `amd64` through a generated Formula. The Formula:

- selects the archive matching the user's CPU;
- uses the public release URL;
- pins the exact checksum;
- installs the binary without cloning source or requiring a Go toolchain;
- installs the project license and third-party notices under the package documentation prefix;
- contains no unreplaced template value or private authentication behavior.

`scripts/render-formula.sh` renders the project Formula from the reviewed template. Formula changes are proposed through a pull request so the generated diff and release references are visible before merge.

`scripts/audit-formula.sh` creates a collision-resistant temporary tap name from `mktemp`, verifies that name is not already installed, and records ownership only after `brew tap-new` succeeds. Its exit trap removes only that owned tap. It never pre-emptively untaps a fixed name, so an existing user tap is outside its cleanup authority. The release profile tests this property with a fake Homebrew boundary on both audit success and audit failure.

Prereleases do not update stable Formula metadata.

## Release preparation

Create a work packet for a release and record:

- target version and rationale;
- included changes and compatibility impact;
- security fixes and disclosure coordination;
- migration or deprecation notes;
- required profiles and their results;
- clean-environment installation evidence;
- artifact and checksum verification;
- public-boundary review.

Before tagging, run:

```sh
task check
task security
task release:check
task public:check
```

Then review the exact commit that will receive the tag. A clean local run does not authorize tagging a different revision.

## Failure and recovery

- If preflight or any matrix build fails, publish nothing.
- If the artifact set is incomplete, do not create a partial stable release.
- If a checksum or Formula reference is wrong, correct it through a reviewed replacement release or metadata change; do not silently mutate evidence.
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
