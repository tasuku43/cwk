# Presentation Decision: Subtractive Task Projection

- Decision date: 2026-07-19
- Decision owner: Project owner
- Current schema: `cwk-task-projection/1`
- Replaced schema: `cwk-context-capsule/1`
- Benchmark status: inconclusive; no challenger passed the frozen gates

## Decision

The default Chatwork success presentation is a simple subtractive task
projection. It starts from the catalog task contract and omits fields that the
task does not declare. It does not introduce a general compression language,
an alias dictionary, adaptive summaries, or a second public output selector.

This is an explicit product and compatibility decision by the project owner.
Competition 1 did not select a winner: its strict scorer rejected every
challenger, and the audit found a wrong relationship oracle plus an ambiguous
recovery scenario. Candidate P supplied the closest implementation seed and
the best descriptive token result, but it is not recorded as having passed the
frozen promotion gate.

## Output rule

The task projection emits only:

- its versioned schema and exact task;
- catalog-declared task fields in provider order;
- validated canonical references directly under their semantic field names;
- coverage kind, a positive provider bound when one exists, completeness, and
  message-relation uncertainty;
- external text under explicit `untrusted` framing; and
- explicit zero, false, empty, absent, acknowledgement, and parent facts when
  the task contract distinguishes them.

It omits:

- display-only aliases and their reference dictionary;
- the global alias policy and generic result wrapper;
- icon URLs, profile fields, counters, timestamps, descriptions, and defaults
  not declared for that task;
- zero coverage limits that do not represent a bound;
- authored coverage prose already represented by typed kind/bound/completeness
  fields; and
- separate renderer-derived raw-notation records when typed To, reply, and
  quote facts already exist. The declared message body remains visible as
  untrusted external text even when it contains provider notation.

Sender display name remains in message results because it is a useful declared
identity label, is structurally framed as external text, and avoids forcing a
second lookup for ordinary message attribution. The catalog now declares it as
`sender_name` rather than leaving it as renderer-only data.

## Safety corrections before promotion

The experimental P renderer exposed a kind-laundering risk: a structurally
valid account reference in a room field could otherwise be printed as a room
reference. `chatwork.Result.Validate` now enforces the contextual kind of every
room, account, message, task, file, invite, request, recipient, reply target,
and quote target before presentation. Reply and quote kinds are also checked.
Optional unresolved targets and an absent file-message reference remain
explicitly representable; a resolved relation requires its correctly typed
target.

These checks reject invalid semantic input and do not change valid competition
fixture bytes. Projection tests additionally reject task-irrelevant profile
canaries and bind the message fields to the catalog.

## Compatibility and migration

This is an intentional breaking change to the default text contract:

- the header changes from `cwk-context-capsule/1` to
  `cwk-task-projection/1`;
- aliases such as `r1`, `a1`, and `m1` and the `refs` dictionary disappear;
- canonical values move directly to `room-ref`, `account-ref`, `message-ref`,
  `task-ref`, `file-ref`, `invite-ref`, and `request-ref` fields;
- the generic `result` hierarchy becomes a task-named collection or outcome;
  and
- relation and coverage lines use the task-projection grammar.

Agents and scripts must consume the canonical value beside the task field and
pass it unchanged to the next command. They must not reconstruct the former
aliases. No persisted state or provider data migration is required. Rollback
is the old binary; mixed-version text consumers must branch on the first-line
schema identifier.

The public command paths, arguments, effects, authentication, provider calls,
typed semantic boundary, failure schema, exit statuses, and JSON help contract
do not change.

## Evidence and implementation

- Frozen and audit result: [evaluation-audit.md](evaluation-audit.md)
- Raw runs, losing candidates, invalidated attempts, static measurements, and
  commit bindings: [evidence/manifest.json](evidence/manifest.json)
- Candidate P seed: `b804f8efdaeb318e94b1b5e9d6144c00149e4674`
- Integration seed commit: `258087d`
- Semantic kind enforcement: `3751fec`, refined by `a832f69`
- Redundant coverage removal and catalog alignment: `4669d41`
- Subtractive projection contract tests: `07e6961`

The extra coverage subtraction occurred after the owner decision and was not
retroactively attributed to the frozen P benchmark. It removes redundant prose
without changing answer-key facts. No additional model call was used.

## Non-goals

- Repairing and rerunning competition 1.
- Claiming that P passed the frozen benchmark.
- Shipping L, R, J, a hybrid, or a public format switch.
- Adding lossy summaries, inferred relationships, raw-provider output, or
  model-generated presentation.
- Reopening authentication, API coverage, multiple accounts, or GUI scope.
