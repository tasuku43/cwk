# Proposed Competition Protocol

This protocol must be reviewed and changed to `Frozen` before scored candidate
measurements begin. Prototype commits made while the runner was undergoing
live conformance are rebased onto the exact frozen base and reverified before
they become eligible; no prototype measurement is scored.

- Status: Proposed
- Baseline: C0, `cwk-context-capsule/1`
- Challengers: P task projection, L normalized ledger, R relationship-first
  timeline, J typed semantic JSON
- Semantic inputs: [fixtures.md](fixtures.md)
- Concepts: [concepts.md](concepts.md)

## Pre-implementation freeze

Record these exact values before creating candidate commits:

- base commit after shared correctness repair;
- fixture and semantic-answer-key SHA-256 digests;
- runner and scoring commit;
- agent/model identifiers and immutable version or dated snapshot;
- system/task prompts and context supplied to each run;
- tool versions and invocation budget;
- temperature, seed support, timeout, and concurrency;
- authoritative API token-usage fields and pinned fallback tokenizer/version;
- operating system, architecture, and Go version.

No candidate implementation starts while one of these is undecided. A later
change invalidates prior measurements unless the report demonstrates that it
cannot affect the comparison.

## Isolated worktrees

Create every worktree from the same frozen base:

| Candidate | Branch | Suggested worktree |
|---|---|---|
| C0 | `codex/prescomp1-c0` | `../cwk-prescomp1-c0` |
| P | `codex/prescomp1-p` | `../cwk-prescomp1-p` |
| L | `codex/prescomp1-l` | `../cwk-prescomp1-l` |
| R | `codex/prescomp1-r` | `../cwk-prescomp1-r` |
| J | `codex/prescomp1-j` | `../cwk-prescomp1-j` |

Before adding worktrees, require a clean integration worktree, record
`git worktree list --porcelain`, and verify that none of the target paths or
branches exists. Candidate commits may change presentation and candidate-local
tests only. Changes to fixtures, semantic types, application selection,
provider mapping, catalog facts, or scoring make the run ineligible.

C0 must have no diff from the frozen base. Challenger diffs are restricted to
the reviewed CLI renderer package, candidate-specific tests, and candidate
goldens. `internal/domain`, `internal/app`, `internal/infra`, catalog and
command wiring, error rendering, fixtures, answer keys, prompts, protocol,
evaluator, `go.mod`, and `go.sum` are forbidden paths. Validate every
`<BASE_SHA>..HEAD` path mechanically before a run.

## Benchmark unit and agent tasks

The scored unit is a realistic natural-language user situation, not a direct
field-extraction question. A fresh agent receives the fixed system prompt, one
user situation, and the offline simulator invocation. It chooses and executes
real public `cwk` argv paths against fixture-backed state. Public help is
rendered by the production catalog; task results use the candidate renderer.
The scorer deterministically replays every recorded argv, verifies stdout and
stderr hashes, checks command discovery and exact-reference flow, and then
scores the final exact JSON outcome.

The initial pilot situations cover attention rooms, recent-thread
relationships, safe message selection without mutation, created file and
parent verification, explicit zero after mark-read, structured failure
recovery, a 100-room selection, and hostile prompt-like message data.

Run one candidate-local scenario through the pinned Codex runner as:

```sh
go run ./tools/presentationeval run \
  --candidate <c0|p|l|r|j> \
  --scenario <id> \
  --codex <absolute-path-to-codex-0.145.0-alpha.18> \
  --model <pinned-model-id> \
  --out <runs.jsonl>
```

Run the complete frozen 12-run candidate schedule as:

```sh
go run ./tools/presentationeval run-suite \
  --candidate <c0|p|l|r|j> \
  --codex <absolute-path-to-codex-0.145.0-alpha.18> \
  --model gpt-5.6-terra \
  --out <new-runs.jsonl>
```

The runner builds the candidate-local evaluator as a disposable executable
named `cwk`, supplies only the fixed scenario and candidate through the agent
shell policy, and invokes public `cwk` argv. Codex uses `workspace-write` with
approval `never`; agent-shell environment inheritance and sandbox network are
disabled, as are installed apps, plugins, browser/web tools, and multi-agent
features. Unknown JSONL events, item types, non-`cwk` commands, and shell
operators fail closed. Tests use a fake Codex process and no network.

The paired no-tool measurement is also independently runnable:

```sh
go run ./tools/presentationeval token-probe \
  --candidate <c0|p|l|r|j> \
  --scenario <id> \
  --codex <absolute-path-to-codex-0.145.0-alpha.18> \
  --model <pinned-model-id> \
  --out <probe.json>
```

The machine-readable draft is `benchmark-protocol.json`, validated against
`benchmark-protocol.schema.json`. Run submissions conform to `run.schema.json`.

## Exact outcome tasks

Each run receives only the public root/scoped help allowed by the scenario and
one candidate's direct output. It must return a small exact JSON answer for:

1. select the exact room-discovery and bounded-message commands;
2. identify task-requested room, account, message, task, and file facts;
3. distinguish To, resolved reply, unresolved reply, quote, and absence;
4. state window kind, bound, completeness, and missing context;
5. select the exact canonical reference for a declared next command;
6. select an exact recovery action from a structured failure.

The answer scorer compares values, not presentation-specific explanations.
Unsupported extra prose does not repair a wrong or missing value.

## No-post-processing rule

The task fails if the transcript uses or asks for `jq`, `grep`, `awk`, `sed`,
Python, a custom parser, a pipe, raw notation interpretation, source
inspection, a manual identifier join not declared by the format, an
exploratory provider call, or an undocumented follow-up command. Reading a
declared canonical field directly is allowed.

## Repetitions and run order

- One primary repetition per candidate/situation pair.
- One additional repetition for each of the four predeclared high-variance
  situations.
- Eight scored agent situations: 8 primary runs per candidate, plus 4
  high-variance confirmation runs per candidate, for 12 runs per candidate
  and exactly 60 scheduled scored runs across the fixed five candidates.
- Run one paired no-tool token probe per candidate/situation, then reuse that
  deterministic measurement for the additional high-variance repetition.
- The scored schedule consumes at most 140 external model invocations: 60
  workflow runs plus 80 paired-probe calls. The complete competition,
  including conformance attempts, has a hard cap of 160 invocations.
- Alternate candidate order with a Latin-square schedule using fixed seed
  `20260718`.
- Do not run two candidates in the same agent context.
- Pin `gpt-5.6-terra` with reasoning effort `medium`. The Codex surface does
  not expose temperature or seed controls for this model; run order and every
  observable setting remain fixed.
- Preserve failed, timed-out, and malformed runs. Do not replace them silently.

## Eligibility gates

A candidate is rejected before efficiency comparison unless it achieves:

- 100% deterministic-byte, trust-framing, and canonical-reference checks;
- 100% on every critical identity, relationship, bound, completeness,
  mutation-outcome, and recovery answer across all repetitions;
- at least 97% micro accuracy over all scored fields;
- at least 95% accuracy in every non-critical scenario family;
- zero external reconstruction/tool violations;
- no fabricated relationship or silently strengthened completeness claim;
- a paired 95% confidence-interval lower bound no worse than 1 percentage point
  below C0 for overall answer accuracy.

These are floors, not weights. Token savings cannot compensate for failure.

## Promotion gates

An eligible challenger may replace C0 only when it satisfies one of:

1. median total task tokens decrease by at least 15%, and the paired 95%
   confidence-interval lower bound for reduction is at least 10%; or
2. overall exact-answer accuracy improves by at least 3 percentage points while
   median total task tokens increase by no more than 5%.

It must also have no increase in median agent tool steps, no production
dependency addition, and candidate render latency p95 no more than 20% above
C0. Serialized bytes and maintenance cost are reported even when they do not
decide the gate.

If multiple challengers pass, select from the Pareto frontier rather than
combining scores into an unreviewed weighted total. An inconclusive result
retains C0.

Paired intervals use task-and-repetition pairs, scenario stratification,
10,000 bootstrap resamples, and seed `20260718`.

## Hybrid rule

A task-family hybrid is eligible only when every component independently
passes the hard gates on its applicable fixtures and routing is a static
catalog-task projection. Data-dependent heuristics, output-size switches, and
agent-selected detail flags are separate candidates and require their own
frozen hypothesis and tests.

## Measurements

Record per run:

- candidate, exact commit, fixture, task, repetition, and agent/model version;
- rendered input bytes and candidate-render latency;
- prompt, cached, completion, and total task tokens from the authoritative
  source, plus fallback tokenizer count when used;
- answer JSON, exact scored fields, failures, and critical-answer status;
- agent tool calls, discovery calls, and external-processing violations;
- wall time, timeout, and malformed-output state.

End-to-end total tokens remain a workflow-cost measurement, but they are not
the presentation-efficiency denominator: a minimal Codex probe has a large
fixed context that can hide renderer differences. Each scored run therefore
also records a model-authoritative paired no-tool probe. Candidate and control
requests use identical model, prompt, configuration, and tool policy; the only
difference is candidate rendered output versus a blank/control slot. The
primary presentation-token input is
`candidate_input_tokens - control_input_tokens`. Bytes and hashes remain
deterministic transport measurements and are never treated as token counts.

Report paired deltas against C0, medians, p95 latency, confidence intervals,
failure counts, and every ineligibility reason. Retain raw runs; do not report
only aggregate winners.

## Artifact layout

```text
evidence/
  manifest.json
  fixtures.sha256
  prompts/
  rendered/<candidate>/<fixture>.txt
  runs/<candidate>.jsonl
  scores/<candidate>.json
  summary.json
  report.md
```

Artifacts use strict schemas, stable key ordering where bytes are compared,
and repository-relative paths. Secrets, live data, local absolute paths, and
provider traffic are forbidden. Large or model-derived raw evidence should be
retained according to the reviewed repository/publication decision rather than
committed automatically.

## Integration and cleanup

Review exact commits and raw results before integrating one candidate. Rebase
or cherry-pick only the accepted presentation changes after resolving the
public schema/version decision. Run the required gates on the integration
branch. Preserve a report for every candidate, then remove only clean,
explicitly named experimental worktrees. Never force-remove a dirty worktree or
delete an unmerged branch containing the sole copy of evidence.
