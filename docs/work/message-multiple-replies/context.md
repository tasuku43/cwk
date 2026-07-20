# Work Context: Preserve multiple explicit message replies

## Current behavior

- `internal/infra/chatworkapi/notation.go` returns exactly one `*Relation` and explicitly treats a second `[rp]` tag as unknown.
- `chatwork.Message` stores one `Reply *Relation`; domain validation, application selection/closure, and presentation therefore assume one reply edge.
- Valid multiple-reply input retains the raw body but drops recipients and replies, causing `relation-state=unknown` and incrementing `unknown-relation-sets`.
- Single replies resolve and render correctly.

## Relevant structure

- Entry point: `messages list`
- Domain rule: `chatwork.Message`, message validation, relation-resolution validation
- Application use case: `ResolveMessageRelations`, message selection, recursive relation closure
- Infrastructure boundary: Chatwork notation parser and response mapping
- CLI catalog or presentation: message output fields and `internal/cli/capsule`
- Existing tests and harness checks: notation, semantic fixtures, relation resolution, context selection, current projection goldens, agent help

## Constraints

- Provider body remains untrusted data; only complete reviewed `[rp]` syntax creates an edge.
- Any malformed recognized tag keeps the all-or-nothing unknown relation-set rule.
- Reply order follows provider body order and duplicate/cycle handling remains bounded.
- A resolved relation must be backed by the displayed/source/fetched typed message set.
- Single-reply text is compatible; multiple replies use the existing compact list syntax.

## External facts

- Source: Chatwork message notation guide already pinned by the product contract. No new remote content is required for this change.
- User-supplied black-box evidence: multiple complete `[rp]` tags are emitted by Chatwork and can refer to distinct messages inside one returned window.

## Unknowns

- [x] Ordered slice versus unordered set: use an ordered slice so provider notation order remains observable and deterministic.
- [x] New output label versus list-valued `reply`: retain the existing label and compact scalar/list grammar, matching `to` and `quote`.

## Thesis evidence

- Repeated design decision or point of agent confusion: a single relation slot forced valid provider facts into semantic unknown.
- User outcome or friction observed in the minimal slice: reply-chain agents must parse raw `[rp]` body text at multi-reply nodes.
- Code workaround or exception being considered: retaining only the first reply would silently discard a proved edge.
- Current thesis that resolves it, or proposed thesis revision: operational closure and semantics-before-presentation require a typed one-to-many reply model.
- Downstream impact: domain, parser, application relation algorithms, projection, agent contract, docs, fixtures, and harness claims.

## Reproduction or observation

```text
[rp aid=1002 to=4101-8802][rp aid=1003 to=4101-8803] ...
```

Current result: no typed reply/To fields and `relation-state=unknown`.
Required result when both targets are in-window: `reply=[#41,#42] to=[a2,a3]`.

## Security and public-boundary notes

- Assets and side effects involved: read-only Chatwork messages; no new side effect.
- Credentials or confidential data involved: unchanged PAT boundary; synthetic fixtures only.
- New dependencies, destinations, files, processes, or generated content: none.
- Pagination, timeout, retry, idempotency, and cancellation facts: unchanged.
- Publication and licensing concerns: no copied private data; fixtures remain synthetic.

## Glossary

- Multiple reply: one message body containing two or more complete explicit `[rp]` tags.
- Unknown relation set: the whole reviewed notation set could not be proved because at least one recognized form was malformed or contradictory.
