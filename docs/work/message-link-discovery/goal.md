# Work Goal: Discover message references from a link

- Status: Implementation reviewed; readiness evidence pending
- Retention: temporary
- Retention reason: None
- Governing contract: docs/00_theses.md through docs/04_harness.md
- Review/delete trigger: Promote the reviewed contract, verify implementation, then remove packet
- Owner: Maintainer
- Target: Feedback item 6
- Related ADRs: None

## Outcome

An agent can turn a user-supplied Chatwork message link into the canonical room and message references needed by existing read commands, without supplying the link directly to an act command or requiring external parsing.

## Non-goals

- Fetch a URL or infer that the target exists or is accessible.
- Accept arbitrary hosts, unverified URL variants, room-only links, or references normalized from malformed numeric input.
- Change the inputs accepted by existing act commands.

## Acceptance criteria

- [ ] A catalog-registered configurable read-only discover command returns the exact validated room and message references from the supported link form, with no authentication or network call.
- [ ] Its output explicitly distinguishes local extraction from verification of existence or access.
- [ ] Malformed, hostile, oversized and unrecognized links fail before any effect, with Japanese recovery; no untrusted URL is reflected into terminal structure or recovery commands.
- [ ] Agent help leads to link discovery when the resolver is active and to messages show when both leaves are active, without requiring a parser. If show is disabled, explain that a person must enable it through config.
- [ ] Document contract, source evidence, command-selection behavior, reference flow, security boundary and tests in durable contracts.
- [ ] Focused tests, new independent Luna reviews and task check pass.
