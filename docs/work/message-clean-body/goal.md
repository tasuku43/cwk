# Work Goal: Comparable semantic message bodies

- Status: In progress
- Retention: temporary
- Retention reason: None
- Governing contract: docs/00_theses.md and docs/09_agent_readiness_validation.md
- Review/delete trigger: Promote durable conclusions and delete after the comparison and implementation decision
- Owner: Maintainer
- Target: Feedback item 7
- Related ADRs: None

## Outcome

Prepare a typed message-body representation that separates only recognized addressing and quote metadata from text, while preserving quoted text, code, order, and the exact source body. Use it for a controlled comparison before deciding on any public presentation change.

## Non-goals

- Guess undocumented system notification kinds or object references.
- Change public output, selection, or mutation behavior without the required comparison and compatibility decision.

## Acceptance criteria

- [ ] Semantic fixtures prove ordinary text, addressing, replies, quoted text and code are preserved or separated without inventing meaning.
- [ ] Unknown or malformed notation is represented as unknown without partial cleaned claims; original body remains unchanged.
- [ ] Compare candidates against the same fixtures and actual agent measurements under the governing protocol before selecting public output.
- [ ] Required reviews and repository gates pass.
