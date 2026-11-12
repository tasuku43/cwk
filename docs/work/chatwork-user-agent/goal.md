# Work Goal: Identify cwk requests with a bounded User-Agent

- Status: In review
- Retention: temporary
- Retention reason: Implementation review packet for the User-Agent contract
- Governing contract: `docs/03_security_model.md` network boundary and `docs/08_external_api_contracts.md` Chatwork API contract
- Review/delete trigger: Delete after the Terra review and promotion of any durable follow-up; Git retains the packet history
- Owner: Codex
- Target: #19 User-Agent
- Related ADRs: None

## Outcome

Every Chatwork API request identifies the running `cwk` version through one
bounded, secret-free `User-Agent` header. Development and release builds use
`cwk/dev` and `cwk/<version>` respectively. Invalid version input cannot alter
request headers or appear in the transmitted value.

## Why now

The executable already injects its release version into `cli.Version`, while
the infrastructure adapter owns every authenticated request. Carrying that
version into the shared adapter request path gives the provider a stable client
identifier without changing command behavior or transport policy.

## Non-goals

- Changing request hosts, routes, methods, redirects, PAT headers, timeouts, or attempt policy
- Adding a dependency, public CLI flag, environment variable, or application/domain version contract
- Sending commit identifiers, credentials, account details, or raw invalid version input

## Acceptance criteria

- [x] The lazy CLI adapter factory passes `CLI.Version` as a scalar into infrastructure.
- [x] Infrastructure accepts only `dev` or bounded repository-compatible SemVer without a leading `v` or build metadata, and falls back silently to `cwk/dev` for invalid input.
- [x] The shared authenticated execution path sets exactly `User-Agent: cwk/<version>` for every task request, including `/me`, reads, mutations, and uploads.
- [x] Negative tests cover controls, newlines, Unicode, slash, oversize, malformed SemVer, leading-zero components and prerelease identifiers; prerelease SemVer is accepted.
- [x] HTTP fixtures verify exact User-Agent and PAT binding across all task routes.
- [x] Focused infrastructure and CLI factory tests pass; no third-party dependency or layer inversion is introduced.
- [x] Security/API contracts, harness enforcement, and the add-capability workflow guidance state the version and test requirements.

## Governing documents

- Thesis: `docs/00_theses.md`
- Product contract section: `docs/01_product_contract.md` (no public outcome change)
- Architecture or security invariant: `docs/02_architecture.md` and `docs/03_security_model.md`
- External API contract: `docs/08_external_api_contracts.md`
- Existing ADR: None

## Completion definition

Complete when code and focused tests demonstrate the request behavior, the
security and API documents state the bounded header contract, review has no
remaining issue, and this temporary packet is removed from the final tree.
