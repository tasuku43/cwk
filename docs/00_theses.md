# Project Theses

This document decides ambiguous product and engineering choices for Chatwork CLI (`cwk`). A thesis states a representation-independent product hypothesis, its consequences, and the evidence that can disprove it. Concrete presentation designs remain replaceable until reviewed evidence and an explicit compatibility decision promote one into a public contract.

## North Star

**An agent can translate a user's Chatwork request into one exact `cwk` task, invoke it without guessing, and understand a bounded, trustworthy result with no routine external reconstruction. Among outputs that meet the required understanding and safety quality, `cwk` minimizes token cost.**

The primary user is a developer or operator who delegates Chatwork work to a coding agent from a shell or automation environment. Human usability remains important, but command certainty, agent understanding quality, and token efficiency are the first optimization targets.

The product is not an endpoint mirror and is not measured by the compactness of one syntax. Its first complete implementation nevertheless has a finite coverage obligation: every operation in the official 2026-07-18 Chatwork API snapshot must be reachable through at least one reviewed user-task workflow. A smaller output is worse when it causes command mistakes, hides missing context, weakens identity, or makes an agent infer relationships.

## Axiom 1: A supported outcome is operationally closed

When `cwk` claims to support a user outcome, it owns the deterministic selection, joining, interpretation, and task-specific transformation needed to use the result.

### Consequences

- Routine success paths do not require `jq`, `grep`, custom joins/parsers, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Direct extraction of a declared field or opaque reference is allowed; reconstructing product semantics is not.
- Repeated external processing is evidence of a missing or overbroad capability, not a workaround to teach every agent.
- Commands express user outcomes rather than provider endpoints.
- A common deterministic workflow belongs in an application use case, not an agent prompt.

### Enforcement

- Catalog entries declare one outcome and complete input/output semantics.
- Agent transcripts count external processing and fail a supported scenario when the count is nonzero.
- Work packets record repeated pipelines as thesis evidence.

## Axiom 2: An agent reaches an executable task without guessing

An agent that knows the user's desired outcome should reach the exact command contract through bounded, machine-readable discovery.

### Consequences

- Root agent help is a compact outcome index.
- An unknown outcome needs at most the root index and one scoped request; a known path needs one scoped request.
- Scoped help declares inputs, effects, authentication, output semantics, completeness, failures, recovery, and reference workflows.
- Commands do not silently search again, choose a display-name match, or rely on hidden defaults.
- Structured recovery names an exact next command rather than prose that the agent must reinterpret.

### Enforcement

- Catalog, routing, and help derive from `cli.Catalog`.
- Root entries retain the 512-byte per-command budget.
- Agent-readiness tests reject command probing, prose scraping, and undeclared follow-up calls.

## Axiom 3: Semantics precede presentation

Chatwork data is converted into a typed, provider-independent task result before presentation. Presentation may reorganize or encode that result, but cannot invent, strengthen, or silently discard semantics required by the task.

### Consequences

- Provider wire JSON is not the public domain model.
- Explicit To, reply, quote, identity, ordering, coverage, and unresolved-reference facts remain distinguishable when relevant to the outcome.
- To does not become a reply; quoted prose, display names, and time proximity do not create relationships.
- Missing or out-of-bound context remains observable rather than being hidden to make an output look complete.
- Canonical opaque references remain available for declared next actions.
- Domain, application, and infrastructure contracts do not depend on a candidate presentation grammar.

### Enforcement

- Shared semantic fixtures have an answer key independent of any renderer.
- Negative tests reject fabricated relationships and silent completeness claims.
- Candidate presentations are evaluated against the same semantic facts and canonical references.

## Axiom 4: Presentation is versioned and replaceable

No concrete output syntax is a thesis. The first complete implementation deliberately selected the context-capsule presentation, candidate C (`cwk-context-capsule/1`), so API work could close against one high-quality contract. It remains the first stable baseline. The current default is the P-derived task projection (`cwk-task-projection/1`), adopted through an explicit owner compatibility decision after Competition 1 was inconclusive and hardened beyond the frozen candidate. That decision accepted a breaking text-schema migration; it is not a claim that P won the defective benchmark. Later presentation changes remain evidence-informed and require an explicit compatibility decision.

### Hard constraints

Every eligible presentation must:

- preserve the semantic answer key and exact canonical references;
- expose task-relevant bounds, missing context, and uncertainty;
- keep external text structurally separate and marked as untrusted data;
- be deterministic for the same typed input;
- require no undocumented parsing convention or external post-processing for the evaluated outcome;
- preserve stdout, stderr, exit, failure, and completeness contracts.

The current task projection additionally follows a subtractive rule: it emits
only catalog-declared task fields, exact canonical references, task-relevant
bounds/completeness/uncertainty, and structural trust framing for external
text. It does not add display aliases, raw Chatwork notation as semantic
structure, undeclared provider or wire fields, duplicated coverage prose, or
helpful non-contract defaults. Declared message bodies remain visible as
untrusted external text; they are not reparsed by presentation to invent
semantics.

### Optimization objectives

Among eligible candidates, prefer the Pareto frontier across:

- agent answer correctness and relationship understanding;
- correct next-command and reference selection;
- input/output token use;
- additional tool invocations and processing steps;
- serialized bytes, latency, and implementation/maintenance cost;
- human reviewability when it affects safe supervision.

Token count is not optimized below the required understanding-quality floor. Numerical thresholds, tokenizer/model versions, fixtures, and repetitions for later optimization are chosen in a presentation-competition work packet, not invented in this thesis or used to block the first complete implementation.

## Axiom 5: Presentation implementations stay behind one semantic boundary

The historical context capsule, the current task projection, and any future presentation hypotheses consume the same provider-independent semantic boundary. Candidate C became the first stable presentation by explicit product decision. Competition 1 did not produce an authoritative winner because its benchmark/oracle defects made the result inconclusive. The owner then deliberately superseded C with a P-derived projection as a breaking compatibility decision; the frozen P candidate was only its implementation seed. Future replacements still compete under comparable conditions before another default change.

### Consequences

- Each candidate consumes the same typed semantic input and provider-independent fixtures.
- Candidate worktrees cannot change semantics, coverage, or answer keys to improve their score.
- The evaluation pins task prompts, model/agent versions, invocation budgets, token accounting, repetitions, and scoring.
- A presentation owns only its grammar, ordering, schemas, and output modes; none leaks into domain or application semantics.
- Historical candidate-C aliases were display-local and never replaced canonical references accepted by commands; the current task projection emits exact canonical references directly and defines no display aliases.
- An inconclusive experiment does not establish a benchmark winner. A default may still change through a separately recorded owner compatibility decision that explicitly accepts the migration, as occurred for `cwk-task-projection/1`.

### Enforcement

- Candidate C retains its status and evidence as the deterministic first stable baseline; the current task projection receives the active golden, semantic-answer, hostile-output, canonical-reference, and subtractive-field tests.
- A future presentation-competition work packet defines candidates and measurement before experimental implementations begin.
- Comparison reports identify each worktree/commit and record raw results, benchmark defects, and audit findings, not only a winner summary. Later compatibility decisions must not relabel inconclusive evidence as a win.

## Axiom 6: Discovery owns ambiguity; actions bind one exact target

Room, account, and message discovery may return candidates. A read or mutation
acts on declared opaque references passed unchanged from an invocable producer.
When the product deliberately owns exactly one local instance and exposes no
selection, the exact command may instead bind one catalog-declared fixed
`tool_local` target. That is an explicit singleton contract, not a default
chosen from available remote or credential state.

### Consequences

- Every public command is `utility`, `discover`, or `act`.
- Display names, positions, browser URLs, and presentation-derived shorthand are not authorization identities.
- An action never case-folds, decodes, trims, reconstructs, or substitutes a Chatwork identifier.
- A fixed-target action declares a stable kind, ID, scope, and description,
  produces/consumes no target reference, and becomes invalid when the product
  admits multiple possible targets.
- Required-reference chains lead back to an invocable producer.

### Enforcement

- Reference kinds live on structured catalog inputs and outputs.
- Fixed singleton targets live on a separate structured catalog field and are
  mechanically excluded from provider-owned or reference-selected flows.
- Whole-catalog graph tests prove producer/consumer reachability.
- Round-trip and negative tests preserve exact identity and reject alternate forms before adapter access.

## Axiom 7: Effects and uncertain outcomes are visible before repetition

An agent must know what a command can affect and whether a previous mutation may have happened before repeating or recovering.

### Consequences

- Every public operation declares `read`, `create`, or `write`; unknown effects fail closed.
- Message creation binds its room parent and declares notification impact.
- Message update/deletion binds the existing message and declares destructive/notification impact.
- Authentication, permission, validation, and policy rejection cause zero downstream calls.
- An unclassified post-mutation outcome is non-retryable and points to read-only reconciliation.
- Credentials and unsafe provider causes remain inside infrastructure.

### Enforcement

- Domain intent and catalog mutation validation reject incomplete targets or impact.
- Fake-adapter tests prove pre-I/O rejection and safe post-I/O classification.
- Secret-canary and public-boundary checks cover output, errors, logs, and fixtures.

## Axiom 8: Claims remain executable

Command certainty, operational closure, semantic fidelity, understanding quality, token efficiency, output safety, and reference flow require repeatable evidence.

### Consequences and enforcement

- `scripts/check.sh` remains the canonical gate and `cli.Catalog` the public-command source of truth.
- Agent evaluations record discovery calls, external-processing steps, canonical references, semantic answers, recovery, tokens, and latency.
- Public examples use synthetic Chatwork-like data.
- `task check` decides implementation completion; higher-risk changes add the named security/public/release profiles.

## First complete implementation

The first complete implementation is bounded by the 32 REST operations linked from the official Chatwork documentation index on 2026-07-18. It must:

1. map every operation to at least one task-oriented public workflow and mechanically reject gaps or unreviewed extras;
2. implement single-account, process-local PAT authentication behind one secret-free binding, fixed-destination bounded transport, provider faults, and safe mutation intent;
3. support room discovery followed by a bounded recent-message result with explicit relationships, canonical references, hostile text, and partial coverage;
4. implement every remaining operation with the same catalog, reference, authentication, effect, and recovery contracts;
5. establish candidate C deterministically as the first stable presentation and prove its semantic answer, bounds, trust framing, and canonical reference flow;
6. pass full, security, public, local-provider E2E, and agent-readiness gates without live credentials.

Within that finite implementation, every provider operation has one attempt;
metadata/read and non-upload requests use a 20-second timeout and uploads 60
seconds. The checked contract also caps success/error bodies at 8 MiB/64 KiB,
complete output at 16 MiB, aggregate lists at 10,000 items, documented endpoint
lists at 100 items, and upload at 5 MiB. Exact typed invocation suffices for
ordinary creates/updates; the reviewed access-changing and destructive sets
add exact `--confirm=access-change` and `--confirm=destructive`, respectively.
Uncertain mutation outcomes reconcile only through read-only tasks.

Authentication has no selection step. `CWK_API_TOKEN` is the sole Chatwork
credential input and is read only from the command process. The CLI does not
accept it in argv, persist it in project or user configuration, expose an
authentication lifecycle command, or probe another credential source. Missing
or invalid token input fails before a provider task request. The fixed
production destination and secret-free ephemeral binding remain unchanged.

Future provider additions, OAuth grants and lifecycle commands, token
persistence, multiple accounts/profiles, GUI work, release publication,
alternative presentations, and further token optimization are outside this
completion boundary.

After that boundary closed, the owner accepted the breaking default-text
migration from `cwk-context-capsule/1` to `cwk-task-projection/1`. The semantic
task result and exact canonical references remain authoritative; context-capsule
headers, dictionaries, aliases, ordering, and grammar are not compatibility
promises of the current default.

## Explicit non-goals

- Mirroring Chatwork endpoints mechanically or exposing transport vocabulary, even though the fixed public-operation snapshot must be covered by user-task workflows.
- Raw routes, arbitrary headers/bodies, or transport passthrough.
- Treating either the historical context-capsule grammar or the current task-projection grammar, field selection, or ordering as an axiom or domain model.
- Hidden fuzzy target selection.
- Silent truncation or fabricated relationships.
- Default lossy or model-generated summaries.
- Claiming structural escaping prevents semantic prompt injection.
- Enabling writes before authentication, impact, retry, and reconciliation policies are concrete.

## Thesis lifecycle

Record agent confusion, repeated pipelines, extra discovery calls, answer errors, token regressions, unsafe identifier conversions, and comparison bias in the active work packet. Revise an axiom before normalizing a workaround. Presentation evidence updates the selected contract; it does not retroactively turn one implementation pattern into an axiom.
