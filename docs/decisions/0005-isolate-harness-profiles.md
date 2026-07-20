# ADR 0005: Isolate routine and specialized harness profiles

- Status: Accepted
- Date: 2026-07-20
- Deciders: Chatwork CLI maintainers
- Scope: Local verification, CI, release preflight, and Codex automation
- Supersedes: The `task check` profile-composition consequence in [ADR 0004](0004-shared-homebrew-tap.md)
- Superseded by: None

## Context

The canonical `full` profile had accumulated the complete security, public,
and release profiles. A warm local measurement took about 90 seconds, of which
the release profile consumed about 75 seconds by building five targets twice
with independent caches. The completion policy already requires specialized
profiles only when their boundary changes, so nesting every profile inside
`full` made the documented profile separation ineffective.

The tracked Codex Stop hook also ran the fast profile after every agent turn.
That added roughly nine seconds even with a warm cache and failed immediately
when the PATH-selected Go binary was older than the repository's exact local
toolchain requirement. CI remains the completion authority; a mandatory
per-turn hook is neither reliable evidence nor a substitute for an explicit
gate run.

## Decision

- `full` is the ordinary implementation gate: `fast`, vet, race, tidy-diff,
  and Git whitespace checks.
- `security`, `release`, and `public` remain complete standalone profiles and
  are required according to the existing completion rules.
- Pull-request CI runs `full` and the security/public boundary profiles as
  parallel jobs. It does not run the expensive release reproducibility matrix
  for every ordinary source change.
- Release preflight invokes all four required profiles explicitly before any
  artifact publication.
- The repository does not install a tracked automatic Codex Stop hook. Local
  automation is opt-in and must delegate to a named `scripts/check.sh` profile.
- CI dependency caches may be enabled. The release reproducibility check keeps
  its own two isolated build caches, so ambient CI cache hits cannot satisfy
  or weaken the byte-reproducibility comparison.

## Consequences

- `task check` gives a substantially shorter ordinary completion loop without
  removing race, architecture, catalog, localization, or behavior coverage.
- Security and public checks remain enforced in ordinary CI, while release
  checks run only for packaging/release changes and release preflight.
- Maintainers must name every specialized profile they ran; `task check` no
  longer implies them.
- Release verification remains intentionally heavy. Future optimization must
  preserve independent exact-revision builds and artifact inspection.

