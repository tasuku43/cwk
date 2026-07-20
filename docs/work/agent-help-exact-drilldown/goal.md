# Work Goal: Safe exact-command agent-help drilldown

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: None

## Outcome

An agent can discover a Chatwork task through bounded root or namespace indexes
and receives the complete machine-readable invocation, output, failure, and
workflow contract only after selecting one exact command. Human-facing help
explains the same drilldown, and the README provides one contract-accurate set
of investigation recipes without encouraging a whole-namespace contract dump.

## Why now

The released `v0.1.1` `messages` namespace agent-help response is 201,042
bytes, compared with 7,901 bytes for the root index. An answer-loop agent can
therefore consume a large amount of context merely by selecting a namespace,
even though the product thesis requires bounded discovery and exact-command
certainty.

## Non-goals

- Changing Chatwork provider calls, message semantics, authentication,
  command selection, effects, or mutation policy.
- Reintroducing `v0.1.0` command-selection filesystem workarounds already
  resolved in `v0.1.1`.
- Claiming that one bounded room window is complete room history.
- Recommending an unevaluated fixed message count as universally optimal.

## Acceptance criteria

- [x] Root agent help remains a compact all-command outcome index and points
  directly to exact-command contracts.
- [x] Namespace agent help returns only the selected namespace's compact
  command index and exact-command request pointer; it contains no complete
  input, output, error, mutation, authentication, or workflow contracts.
- [x] Exact-command agent help retains the complete contract and workflows in
  one invocation.
- [x] Agent-help schema version 4 records the intentional namespace-shape
  break, and shape/size tests enforce all three views.
- [x] Human help and README describe ordinary help, root discovery, and
  exact-command contract drilldown without advertising namespace contract
  aggregation.
- [x] README investigation recipes preserve the bounded 100-message source,
  one-hop reply context, exact sender meaning, cross-room/window limits, and
  exact-message `show` path.
- [x] No provider, credential, side-effect, or public-reference boundary
  changes.
- [x] `task check` passes.

## Governing documents

- Thesis: `docs/00_theses.md`, Axioms 1, 2, and 8
- Product contract: Public runnable surface and supported-outcome promise
- Architecture: Catalog as the public source of truth
- Security invariant: bounded output and exact opaque-reference flow
- Existing ADR: None

## Completion definition

The work is complete when schema-v4 behavior, navigation, byte growth, and
documentation are enforced by tests; the durable contracts agree; the agent
journey reaches one exact command without whole-namespace contract loading;
and `task check` passes.
