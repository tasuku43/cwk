# Presentation Competition Evaluation Audit

- Audit date: 2026-07-19
- Scope: frozen competition 1 evidence only
- Evidence policy: no model reruns, no replacement of failed runs, and no
  changes to the frozen protocol, fixtures, answer keys, evaluator, or
  candidate implementations
- Decision status: no eligible challenger was selected by the frozen protocol

## Executive conclusion

The frozen competition did not produce an eligible challenger. The original
strict scorer rejected C0 and all four challengers. Its results must remain the
record of what the frozen evaluator reported; the audit calculations below do
not convert any candidate into a gate winner.

The evidence also exposed an objective defect in the thread answer oracle and
an ambiguity between the recovery prompt, answer key, and command budget. Those
defects prevent a legitimate presentation-quality comparison from being
reconstructed by changing aggregate statistics alone. Changing their semantic
contract would fall under the invalidation rule in `protocol.md` and requires a
separately reviewed evaluation, not a retroactive winner.

The token evidence is still useful as descriptive product input. After
deduplicating the intentionally reused token probes, the simple projection
formats are materially smaller than C0 on this corpus. That observation does
not overcome the failed quality gates.

## Frozen corpus and original strict scorer result

The scored corpus contains ten workflow runs per candidate: one run for each
of eight situations and a second run for `rooms.large-attention` and
`thread.relationships`. All 50 amended runs passed usage validation and
deterministic transcript replay. The original scorer produced the following
unchanged results:

| Candidate | Exact answers | Critical pass | Workflow pass | Usage pass | Correct leaves | E2E token sum | Reported probe median | Original eligible |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| C0 | 7/10 | 5/10 | 6/10 | 10/10 | 48/73 | 777,442 | 580 | no |
| J | 7/10 | 6/10 | 7/10 | 10/10 | 48/73 | 855,075 | 612 | no |
| L | 8/10 | 6/10 | 6/10 | 10/10 | 60/73 | 933,183 | 455 | no |
| P | 7/10 | 7/10 | 9/10 | 10/10 | 48/73 | 770,897 | 410 | no |
| R | 7/10 | 6/10 | 6/10 | 10/10 | 48/73 | 829,709 | 448 | no |

The score summaries rejected every candidate because not every run had an
exact semantic answer and not every run passed the critical semantic and
workflow gates. C0's rejection is not evidence that a rejected challenger may
replace it; the frozen promotion rule first requires an eligible challenger.

The reported probe median above is also preserved as original scorer output.
It is not the corrected descriptive median because the scorer counted the same
probe again for each confirmation repetition.

## Objective thread-oracle defect

The typed fixture for message `9003` contains two simultaneous explicit
relationships:

- a To relation to account `7001`; and
- a resolved reply relation to message `9001`.

The situation asks the agent to summarize explicit To, reply, and quote
relationships. However, the frozen answer key contains only the reply entry
for message `9003`. It therefore contradicts both the typed fixture and the
natural-language task.

C0, J, P, and R returned both explicit relationships in both repetitions. L
returned both in repetition 1 and omitted the To relation in repetition 2.
The strict scorer marked the semantically complete answers wrong and marked
L's omission exact. Because leaf scoring compares arrays positionally, the
extra true relationship also shifted later entries and reduced each affected
run from 26/26 to 14/26 leaves. This is an oracle defect, not evidence that the
agents fabricated a relationship.

The raw answers and original scores remain preserved. This audit does not
rewrite the oracle or rescore those answers as if a corrected oracle had been
frozen in advance.

## Recovery ambiguity and chronology

The recovery situation has three contracts that do not align cleanly:

1. The prompt asks for the exact read-only recovery command.
2. The structured failure and answer key use the catalog path
   `messages list` as `next_command`.
3. Executing that path in the fixture also requires `--room 4101` and
   `--window recent`, while the frozen command budget is four commands from a
   cold agent context.

Every candidate recovered the bounded window and returned the full executable
command `cwk messages list --room 4101 --window recent`. The strict key instead
expected the shorter path label `messages list`, so all five exact-answer
checks failed on that field.

The transcripts also show the same cold-discovery chronology:

1. an initial guessed read command failed;
2. root and/or scoped help discovered `messages show`;
3. `messages show` produced the intentional not-found failure;
4. scoped help or an attempted list command exposed the required window input;
5. `messages list --room 4101 --window recent` succeeded.

C0, J, L, and P used seven commands; R used eight. The scorer therefore
rejected all five workflows against the four-command budget. This chronology
does not prove that seven commands is the correct product target. It proves
that the frozen prompt, recovery value, allowed cold discovery, and budget did
not define one unambiguous four-command evaluation path.

## Protocol and statistics defects

### Token denominator contradiction

The promotion section gates on the median of total task tokens. The
measurements section later states that end-to-end total tokens are not the
presentation-efficiency denominator and defines the primary input as:

```text
candidate_input_tokens - control_input_tokens
```

The scorer cannot resolve that contradiction. It records the sum, not the
median, of end-to-end tokens and separately reports a probe-delta median.

### Reused probes were counted twice

The protocol runs one token probe per candidate and situation, then reuses it
for the two confirmation repetitions. There are eight independent probe IDs
per candidate, not ten. The score summary takes the median over all ten run
rows, thereby giving `rooms.large-attention` and `thread.relationships` double
weight without an additional measurement.

### Promotion statistics were not implemented

The scorer does not calculate the documented 97% micro-accuracy floor, the
per-family floor, paired accuracy intervals, paired token-reduction intervals,
median command-step comparison, candidate-wide latency p95 comparison, or
production-dependency comparison. Its `eligible` value instead requires every
run to be exact and every combined critical/workflow check to pass. The
fixtures happen to mark nearly every expected top-level value as critical, but
that does not make the missing aggregate gates executable.

The protocol also does not freeze a bootstrap random-number algorithm, a
quantile convention, or whether candidate latency p95 is pooled or checked per
operation. Selecting one of those interpretations after observing results
would be a post-hoc analysis choice.

### Evidence-provenance corrections

The frozen protocol records system-prompt SHA-256
`89b74ae96e29476d7507ee6781b990794faafdb849a03c48d5076de370bbb08b`.
That is the digest of the exact prompt followed by a line feed, as produced by
hashing `jq -r` output. The prompt bytes actually supplied by the runner hash
to `3125053741496f3ee8bfb29aba8cd4eaf71e47abdfb28d61cfdd57a68ee39751`,
which is also the value recorded by all five static manifests. This is a
checksum-encoding documentation defect; the prompt itself did not differ
between candidates.

The original static render manifests record candidate labels, schema, runtime,
and raw measurements but not exact candidate commits. Every one of the 11
static output hashes for each candidate appears in that candidate's scored
transcripts, so the bytes are cross-checked. The committed evidence manifest
now binds each copied metrics digest to the reviewed candidate commit, while
retaining the original provenance limitation rather than rewriting it.

## Corrected descriptive resource calculations

The following audit-only calculation uses the protocol's primary paired probe
input and removes duplicate probe IDs. It does not amend or replace the frozen
score:

- one probe delta for each of eight situations;
- lower empirical median, matching the evaluator convention
  `sorted[floor((n-1) * q)]`;
- relative reduction `1 - challenger_median / C0_median`;
- an audit-only paired situation-cluster bootstrap interval with 10,000
  resamples and seed `20260718`;
- lower median command count over the ten workflow runs; and
- pooled p95 over 220 static latency observations: 11 rendered operations by
  20 iterations.

| Candidate | Unique-probe median | Descriptive reduction vs C0 | Descriptive 95% interval | Lower median E2E tokens | Lower median tool steps | Pooled latency p95 | Latency ratio vs C0 |
|---|---:|---:|---:|---:|---:|---:|---:|
| C0 | 555 | baseline | n/a | 67,747 | 4 | 160,709 ns | 1.000 |
| J | 484 | 12.79% | -5.52% to 36.73% | 63,894 | 3 | 114,459 ns | 0.712 |
| L | 440 | 20.72% | 8.16% to 50.29% | 62,298 | 4 | 75,792 ns | 0.472 |
| P | 328 | 40.90% | 29.31% to 71.43% | 66,239 | 3 | 158,292 ns | 0.985 |
| R | 394 | 29.01% | 22.76% to 66.33% | 63,758 | 4 | 127,333 ns | 0.792 |

All challenger candidate diffs leave `go.mod` and `go.sum` unchanged. Under
the pooled descriptive interpretation, every challenger is below the numeric
1.2 latency ratio and none increases the lower median tool count. These facts
do not mean that P, R, or any other candidate passed the frozen gates: every
challenger failed the frozen eligibility evaluation, and latency provenance
is incomplete.

The end-to-end token medians are reported only as workflow-cost observations.
They are not used as the presentation denominator because discovery and error
recovery dominate several runs.

## Immutable evidence anchors

The reviewed publication decision commits this evidence under `evidence/`.
The complete corpus is small and synthetic, contains no credential, live
Chatwork data, repository source, or local absolute path, and is required to
retain losing and invalidated runs rather than only a winner summary. The
following digests identify the exact files used by this audit.

### Amended raw runs

| Candidate | Rows | SHA-256 |
|---|---:|---|
| C0 | 10 | `4a6610d5afd33738012fcc39d6e3bdaea3143baf9f2f641c777a8da4e7713038` |
| J | 10 | `d13cef2d138528ca13609c6dc8d73e00a80d74e50297eb63a2d0ad6b6d212905` |
| L | 10 | `b7c47b82344bacd426c0ea97c3c382f9067f565fc1a758140318772eac700d99` |
| P | 10 | `cec3473ac63f7701a1cb79e64f97bd2efad45286d548a0da2e042aaec58a4b33` |
| R | 10 | `40dbfab8ab6d7c28db7e312ed1d32e91e19ea40163da6021f8fe1b8fddf470d9` |

### Original scored runs and summaries

| Candidate | Scored JSONL SHA-256 | Summary JSON SHA-256 |
|---|---|---|
| C0 | `f1e4e114c564574bdd28b9d6b91b25a3c32878637b8ee21fd1b0e1239d0cd3a2` | `3f0cd16ac91ac3180a628cd586e250124aaa5565c22491439c5f43cd46ae2899` |
| J | `937f6743c2fd2799971210af0b29605d6e1bfe8732d34419be71edfb031987e9` | `70595834a59f5b849943a410924274679dac12b5f2842989de6e4801d4a10a59` |
| L | `cc407afdb81d5a8ae6282116994844fa2704145cb535ac85e1e1a00f7056bc22` | `fa14dd371351037337802813622f330c5e2d5fa0a0525d199247abbe3a6477ea` |
| P | `35453d8a3c7412166bce28badbf79a804d8da6ce95f5552a3eec8d214154c97f` | `dbe2bb81e9d4edbdcb606d399eddabfc8dab56f2d627f3809e5bd84c0257f15c` |
| R | `229cc12ba610485786e1f95769bfafc79b506a8dc902912d641b5afd116001e5` | `095db00850667745cf92d1d455fc03f85c0b592c730ff396fc894f40dced380b` |

### Retained invalidated runner-v1 submissions

| Candidate file | Rows | SHA-256 |
|---|---:|---|
| C0 | 1 | `0f4ba2530c88b6e5d5ebbde9f73553ae0cca428287613ce3353ac02d73e1e857` |
| L | 1 | `848da4c440c94267c79d31d9f223a983b69437ecd0627b7f98d86b595118db48` |
| P | 2 | `6976b4f599ebaf262bf734a97aa58a095e1bbc8f357e3f32dc78eae13a3605d1` |
| R | 1 | `18a6444cacaefa80aac5c9d10d8137cf6de9603c242405516bff65242dc8b3ab` |

These four files contain the five submissions invalidated by Amendment 1.
They are not included in any score above.

### Static latency measurements

| Candidate | Rows | Render-metrics SHA-256 |
|---|---:|---|
| C0 | 220 | `fffb9c8d2a04a8106643d18db69acb9233eb066fdfd7d00ee6f8d0cf6de305ca` |
| J | 220 | `7b142c4692bd7dd3acfad564cda8e1aa7880e6360e310984b9c6eaa576a0ede0` |
| L | 220 | `2ee2956d44b86cc0b125779bdb4e69760ed25ec761f4e53655aebc20677e20b2` |
| P | 220 | `e8f672c895f923334899063918eed48f3b909d6a18fb2906f24c78c8eb3c2813` |
| R | 220 | `b94030367d5a6d424edb8fe1335e8a52dae7e0ba20b6d46108599ad0034c948c` |

## Benchmark conclusion

The only conclusion available under the frozen rules is:

> No challenger was eligible, so no challenger was promoted. The baseline is
> retained by the protocol's inconclusive-result rule.

The competition supplied useful failure evidence and resource measurements,
but it did not establish a benchmark winner. A future scored comparison must
repair and refreeze the semantic oracle, recovery contract, command budget,
token denominator, independent-probe aggregation, statistical estimators, and
latency provenance before collecting replacement model runs.

## Separate owner product decision

On 2026-07-19, after reviewing the benchmark defects and the descriptive
resource evidence, the project owner chose a **simple subtractive task
projection** as the next product and compatibility direction. This is an
explicit owner decision about what to build and stabilize, not a claim that
candidate P or any challenger passed the frozen benchmark gates.

The decision favors a small, task-shaped reduction of the existing typed
contract and a controlled compatibility transition. It does not automatically
accept an experimental candidate commit, retroactively change its score, or
authorize semantic loss. Exact canonical references, trust framing, explicit
relations, bounds, completeness, mutation outcomes, and structured recovery
remain governed by the project theses and product contract.
