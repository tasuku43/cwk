# Work Goal: Measure message-resource agent readiness

- Status: Design handoff
- Retention: temporary
- Retention reason: Terra needs the verified harness gap and proposed test-only route before completing the real agent transcript.
- Governing contract: `docs/00_theses.md`, `docs/02_architecture.md`, `docs/03_security_model.md`, `docs/04_harness.md`, and `docs/09_agent_readiness_validation.md`
- Review/delete trigger: Delete after a production-CLI fixture runner records and audits a qualifying synthetic Codex run, or after the owner records a reviewed reason the run cannot be performed.
- Owner: Terra / readiness implementation
- Target: Current message-resource readiness re-review
- Related ADRs: None

## Outcome

Record a synthetic, credential-free agent run in which the agent discovers the message-resource and show commands from public CLI help, executes them through the production CLI parser/application/presentation route, passes canonical file/task refs unchanged, and returns the fixture answer without external post-processing or writes.

## Why now

The current fixture test pins a three-command expected sequence and directly invokes the application service with a fake port. It does not run the CLI process or an agent. Calling that sequence a transcript would overstate the evidence and leave the repository's actual-agent readiness requirement unverified.

## Non-goals

- Do not use a real Chatwork account, credential, network request, or private history.
- Do not count a hand-written argv sequence or a manually replayed service test as an agent measurement.
- Do not restore the historical presentation simulator unchanged; it bypasses production CLI task execution.

## Acceptance criteria

- [ ] A fresh Codex run receives a natural-language prompt and synthetic fixture only; the prompt does not disclose the command sequence or canonical resource refs.
- [ ] Public `cwk` help, argument parsing, routing, authentication gate, application service, catalog, and renderer run through a test-only fixture composition path.
- [ ] The agent finds the relevant command paths, uses the exact canonical file/task refs from the resource result unchanged in successful show commands, and answers the semantic answer key.
- [ ] Runner evidence records command discovery and execution, per-call stdout/stderr hashes, provider-port task journal, external tool/process count, write count, agent/model/version, usage, and run limits.
- [ ] The run is bounded, network-disabled, secret-free, and permits only the fixture `cwk` command; external post-processing and writes are observed as zero.
- [ ] `docs/04_harness.md` and `docs/09_agent_readiness_validation.md` distinguish contract tests from measured agent evidence, and report the actual run honestly.

## Governing documents

- Thesis: `docs/00_theses.md`, supported outcomes own their semantics and claims are executable
- Product contract section: `docs/01_product_contract.md`, public command and reference behavior
- Architecture or security invariant: `docs/02_architecture.md` and `docs/03_security_model.md`, CLI composition and bounded external calls
- Existing ADR: None

## Completion definition

The criteria have a sanitized raw runner trace and deterministic audit, focused tests pass, docs cite only observed evidence, and this temporary packet is removed.
