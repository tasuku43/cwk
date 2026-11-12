# Plan: Chatwork User-Agent

## Chosen approach

Have the lazy CLI composition closure pass its current `CLI.Version` string to
`chatworkapi.NewFromEnvironment`. Infrastructure validates `dev` or the
repository's release-compatible SemVer form with a 128-byte ceiling and
constructs a safe `cwk/<version>` string, falling back to `cwk/dev` on any
invalid value. Store only that validated header value on the infrastructure
client and set it in `Client.Execute` alongside the existing `Accept` header.

Keep test-only client construction on `cwk/dev` by default. Exercise a release
prerelease through every response fixture and verify the PAT header and empty
Authorization header at the same HTTP boundary. Add pure negative tests for
controls, Unicode, slash, excessive length, malformed versions and SemVer
leading-zero cases. Add a CLI factory test that captures the scalar lazily.

## Alternatives considered

- Putting build version in a domain or application type: rejected because it
  is transport metadata and would cross layers unnecessarily.
- Relying on the Go transport's default user agent: rejected because it does
  not identify `cwk` or the release.
- Passing the raw CLI version directly as a header: rejected because injected
  or malformed values must not become HTTP header structure.

## Risks and verification

- SemVer validation can drift from release policy; tests mirror the repository
  parser's release subset, and both reject leading `v`, build metadata, and
  numeric leading zeroes.
- Missing a task route would omit the header; the existing all-task transport
  fixture corpus verifies one shared request path, including authentication's
  `/me` check and upload.
- Broad package tests can fail because unrelated shared-worktree changes are
  underway; report focused test results separately.
