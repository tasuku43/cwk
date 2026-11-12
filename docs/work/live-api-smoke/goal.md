# Work Goal: Opt-in live Chatwork read checks

- Status: Active
- Retention: temporary
- Retention reason: None
- Governing contract: `docs/00_theses.md`, `docs/03_security_model.md`, `docs/04_harness.md`, `docs/07_authentication.md`, `docs/08_external_api_contracts.md`
- Review/delete trigger: Delete after durable policy and tests are promoted and required gates pass
- Owner: Main session
- Target: Feedback low-priority live API verification
- Related ADRs: None

## Outcome

Maintainers can opt in a dedicated Chatwork test account to a scheduled, read-only production CLI smoke check without exposing provider results or credentials, independently of ordinary offline gates.

## Non-goals

- Claim comprehensive provider compatibility from a few reads, collect real provider fixtures, or create/update/delete provider data.
- Read a developer's token or run a live API call during ordinary repository checks.

## Acceptance criteria

- [ ] A default-branch-only scheduled workflow runs only with an explicit repository opt-in; missing configuration fails visibly when enabled.
- [ ] A test-only isolated scope invokes selected production CLI reads with finite bounds and no mutation; no provider stdout/stderr, tokens, account details or room content reach logs or artifacts.
- [ ] Tests exercise opt-in, missing token, command ordering, failure propagation, scope isolation, and output suppression with synthetic executables, without live calls or credentials.
- [ ] Japanese documentation explains dedicated test-account setup and the limits of this smoke check. Durable harness and security documents distinguish it from ordinary offline gates.
- [ ] Fresh Luna review and applicable `task check`, `task security`, and `task public:check` pass; live execution remains unclaimed until an account is configured and an actual scheduled run succeeds.
