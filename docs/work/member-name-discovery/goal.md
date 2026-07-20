# Work Goal: Discover room members by display name

- Status: Completed
- Owner: Project owner and Codex
- Target: Current implementation cycle
- Related ADRs: None

## Outcome

An agent that knows a person's display name can use one bounded cwk discovery
command to obtain matching canonical account references, then pass the selected
reference unchanged to `messages list --sender`. Exact-command human help shows
short, catalog-backed recipes related to that command without expanding root,
namespace, or agent indexes.

## Why now

The R7 benchmark removed message reachability failures and wasted boundary
exploration, but the sender task still fell back to a full message dump because
`--sender` requires an account reference and no visible task names the display
name-to-reference step. The owner also requested command-help recipes that
answer "when this, do that" for common workflows.

## Non-goals

- Accepting display names directly in `messages list --sender`
- Automatically selecting one candidate or continuing when names are ambiguous
- Provider-side member search, fuzzy matching, normalization, or case folding
- Adding recipes to root help, namespace help, or any agent-help shape
- Changing Chatwork authentication, destinations, mutation behavior, or message
  window semantics

## Acceptance criteria

- [x] `members find --room <room-ref> --query <text>` returns only room members
  whose display name contains the exact query and preserves canonical refs.
- [x] The result reports the query, total source-member count, match count, and
  completeness; zero and multiple matches remain explicit.
- [x] Filtering is application-owned after one existing members endpoint call;
  the query never becomes a provider parameter or second provider call.
- [x] `messages list --sender` exact help points display-name-only users to
  `members find`, while retaining exact-reference-only input semantics.
- [x] Exact-command human help renders related catalog-backed recipes only when
  every referenced command is visible in the active catalog; root, namespace,
  and agent help remain recipe-free.
- [x] Root and namespace agent indexes remain bounded selection surfaces and
  exact help retains complete reference workflows.
- [x] Synthetic readiness proves the display-name discovery followed by exact
  sender selection without a full message pre-dump or external text processing.
- [x] `task check` passes.

## Governing documents

- Thesis: Axiom 2 discovery ownership and Axiom 4 agent-native presentation
- Product contract: role/reference flow, hierarchical help, Chatwork members and
  bounded message selection
- Architecture: catalog source of truth and application-owned local selection
- Security: opaque reference integrity and untrusted external display text
- Existing ADR: None required; no dependency or trust-boundary change

## Completion definition

The work is complete when the domain, application, catalog/parser,
presentation, human help recipes, readiness evidence, durable documents, and
full repository gate agree on the same non-ambiguous reference flow.
