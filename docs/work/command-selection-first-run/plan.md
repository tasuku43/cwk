# Work Plan: Require first-run command selection

- Status: Complete
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Treat an absent profile as an explicit unconfigured state whose active catalog contains only always-on control commands. Root human help explains why the reduced view is shown. A known configurable path attempted in this state receives a stable `command_selection_required` fault and exact `config` recovery before authentication. Saving any valid profile, including an empty one, completes first-run configuration.

## Alternatives considered

### Advertise everything but block execution

This preserves pre-configuration inspection but makes help disagree with routing and pays the full agent token cost before selection.

### Automatically open config from another command

This makes a read invocation unexpectedly interactive, behaves poorly without a TTY, and creates ambiguity about resuming the original command.

## Design

### Public contract

The existing `config` utility and fixed local target remain unchanged. Missing local state becomes a prerequisite failure for known configurable commands. The stable fault is `kind=rejected`, `code=command_selection_required`, non-retryable, with `next_action=config`. Root help succeeds but contains an explicit unconfigured notice and only the active control commands. Unknown paths and saved-disabled paths remain ordinary `unknown_command` failures.

### Layer changes

- Domain: no new provider or command-selection value is required unless diagnostics need an explicit source/state token.
- Application: doctor/config load interpretation distinguishes absent from saved without changing storage.
- Infrastructure: storage absence contract remains `(profile, configured=false, nil)`.
- CLI and catalog: derive an always-on-only view for absence, classify known configurable invocation before active matching, render the first-run help notice, and declare the fault contract.

### Data and control flow

Load the preference before command matching. If absent, derive the always-on active catalog. Render root help from that view with an unconfigured notice. If argv names a known configurable command or scoped help, return the prerequisite fault before lazy PAT creation. `config` continues through the existing terminal and save boundary.

### Error and cancellation behavior

`command_selection_required` is non-retryable and maps to exit 10. It recommends exact `config`. Cancellation, invalid/unsafe/unavailable preference handling, non-TTY config rejection, and uncertain-save reconciliation remain unchanged.

### Security and public boundary

No new credential, destination, dependency, or persistent field is added. Tests prove zero PAT/provider calls. The selector remains a local attention preference and not authorization.

## Implementation slices

1. Contract and failing tests
2. Active-view and dispatch behavior
3. Help and doctor presentation
4. Harness and durable documentation
5. README first-run flow

## Verification

- Unit and contract tests: active view, root/scoped help, error text/JSON, doctor, config defaults
- Negative side-effect tests: zero PAT factory and provider calls before configuration
- Opaque-reference and complete-pagination tests: unchanged
- Structured output, hostile-output, and recovery tests: fault schema and exact next action
- Agent-readiness scenario and discovery-round-trip count: initial root help exposes only control commands and exact config recovery
- Manual observation: clean temporary `XDG_CONFIG_HOME`
- Required profiles: `task check`
- Generated-diff or artifact checks: included in full gate

## Rollout and rollback

Existing saved profiles are unchanged. Installations that relied on deleting or never creating the preference must run interactive `cwk config` once. Rolling back restores the former all-enabled absence behavior without changing saved-profile bytes.

## Documentation promotion

Update theses, product contract, architecture, security model, harness claims, and README. No ADR is needed because this is a direct refinement of the existing active-view thesis rather than a competing architecture.
