# Work Goal: Bound displayed message bodies

- Status: In progress
- Retention: temporary
- Retention reason: None
- Governing contract: `docs/00_theses.md`, `docs/01_product_contract.md`, `docs/02_architecture.md`, `docs/03_security_model.md`, and `docs/04_harness.md`
- Review/delete trigger: Remove after the catalog, implementation, tests, and durable contracts agree
- Owner: Codex
- Target: Current `messages list` capability work
- Related ADRs: None

## Outcome

`messages list` can cap each rendered message body independently by raw UTF-8 bytes while preserving complete typed relations, canonical identities, source ordering, selection metadata, and safe terminal projection.

## Why now

The existing `--max-bytes` option bounds total escaped stdout by withholding complete primary anchor units. Some users also need to keep each visible body short while retaining all otherwise selected records and context.

## Non-goals

- Changing typed body values, filtering, reply/quote interpretation, relation resolution, provider requests, or authentication.
- Changing output when `--max-body-bytes` is omitted.
- Treating body truncation as source incompleteness or byte-delivery incompleteness.

## Acceptance criteria

- [ ] `--max-body-bytes` accepts strict decimal values from 0 through 65,535 and rejects invalid values before authentication or provider I/O.
- [ ] Truncation uses the longest UTF-8-valid raw-byte prefix, then terminal escaping; affected records identify truncation and original raw byte length.
- [ ] Selection, relation/context semantics, canonical references, and source/output completeness remain unchanged; `--max-bytes` measures the final escaped projection after body limits.
- [ ] Scoped help, catalog metadata, durable docs, hostile-output tests, and semantic fixtures explain the behavior.
- [ ] Focused package tests pass; completion status follows the repository gate managed by the task owner.

## Governing documents

- Thesis: `docs/00_theses.md`, bounded message selection and task projection
- Product contract section: `docs/01_product_contract.md`, messages list and byte-bounded delivery
- Architecture or security invariant: CLI owns body display limits; presentation cannot alter typed semantic results
- Existing ADR: None

## Completion definition

The work is complete when acceptance criteria have evidence, durable decisions are promoted to numbered documentation, focused and repository gates are reported, and this temporary packet is removed from the final tree.
