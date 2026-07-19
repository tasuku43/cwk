# Work Tasks: 日本の利用者を既定とするローカライズ

## Understand

- [x] Read governing theses, product, architecture, security, harness, authentication, external API, and agent-readiness documents.
- [x] Observe current English help, catalog, error, TUI, and public-document surfaces.
- [x] Record scope, compatibility, and trust constraints.

## Decide

- [x] Select Japanese as the single default human language.
- [x] Keep machine identifiers, output schema tokens, references, fixtures, and historical evidence unchanged.
- [x] Record the pre-1.0 text compatibility impact.

## Implement

- [x] Update durable product and contributor policy.
- [x] Localize human help and TUI.
- [x] Localize public fault messages and recovery reasons.
- [x] Localize catalog descriptive prose used by help and agent contracts.
- [x] Localize active public documentation and GitHub templates.
- [x] Add localization regression checks.

## Verify

- [x] Focused tests pass. Evidence: `go test ./internal/cli`, application packages, and localization/repoguard tests.
- [x] `task check` passes. Evidence: 2026-07-19 full profile with Go 1.26.5; includes race, security, release, and public checks.
- [x] `task security` passes. Evidence: full profile reported `repoguard (security): OK`, no called vulnerabilities, and no gosec findings.
- [x] `task public:check` passes. Evidence: full profile reported `repoguard (public): OK` and `contractlint: OK`.
- [x] Runtime help and error examples are Japanese. Evidence: root, `rooms` namespace, `messages list` exact help, and `unknown_command` were observed.
- [x] Repository status and generated diff are understood. Evidence: no generated contract or module changes; localization source, tests, docs, and harness changes only.

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions are promoted.
- [x] Follow-up work is explicit: a future additional locale requires a separate reviewed locale-selection and fallback contract; it does not block the Japanese default.
