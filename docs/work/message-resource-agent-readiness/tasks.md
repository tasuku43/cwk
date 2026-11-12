# Tasks: Message-resource agent readiness

The older scored-runner tasks below are historical, not a mandate to build a larger harness. The user chose a simpler exploratory evaluation; its evidence and limits appear at the end of this file and in `context.md`.

## Completed in this handoff

- [x] Rename the fixture field to `ExpectedCommandArgv`; remove the fixed command list from the natural-language prompt.
- [x] Keep the answer key semantic-only and rename the service-level test as an expected-sequence test.
- [x] Correct `docs/04_harness.md` and `docs/09_agent_readiness_validation.md` so they do not call fixed-sequence evidence an agent transcript or claim measured zero external processing.
- [x] Run `go test ./tools/presentationeval` successfully.
- [x] Inspect the historical runner, current CLI dependency injection, installed Codex/cwk versions, and report the gap to the parent.

## Current status

- [x] Add the tag-scoped CLI composition seam and synthetic fixture child executable.
- [x] Disable the unsafe draft run and prevent an unverified audit from declaring a pass; run focused race tests and obtain independent reviews of both guards.
- [ ] Design and review host-secret/network isolation, executable identity, independently observed writes, and semantic answer assessment; implement and validate them before enabling a scored run.
- [ ] Freeze the target tree and all budgets before scored invocation; preserve intermediate traces separately from final evidence.
- [ ] Record command discovery, exact argv/ref flow, read and write journal, external-tool count, usage, and deterministic answer audit from the actual run.
- [ ] After a qualifying scored run, update docs with its measured result; until then, state that no formal scored run has been conducted and identify the unresolved prerequisites.
- [ ] Run the focused suites and required fresh Luna reviews, then delete this temporary packet when the readiness evidence is complete.


## Simplified exploratory evaluation

- [x] Freeze a source copy and compare listed file hashes before and after one fresh-agent run.
- [x] Record the natural-language task, observed CLI/provider calls, manual answer assessment, and journal limitations in `context.md` and the synthetic JSONL.
- [x] Review this evidence and the changed readiness wording with a fresh Luna reviewer; distinguish this exploratory run from comparison and final validation.
