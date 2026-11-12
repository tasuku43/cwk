# Work Plan: Optional JST message timestamps

- Status: Proposed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Add a CLI-only boolean `--jst` option to `messages list` and `messages show`. Strip it before typed request construction, pass it to the task projection, and append `send-time-jst` only to actual message rows. Keep Unix seconds unchanged. Use a fixed +09:00 zone and precomputed representability bounds for four-digit RFC3339 years; render out-of-range seconds as `unavailable`.

## Alternatives considered

### Use the host local time zone

Rejected because output would vary across environments and could require host zone data.

### Replace Unix seconds with a formatted timestamp

Rejected because it would change existing semantics and make timestamp reuse/ranking less direct.

## Design

### Public contract

`messages list` and `messages show` accept an optional boolean `--jst`. Default output remains byte-for-byte unchanged. When set, every displayed message row also includes `send-time-jst`; list's collection schema marks the field optional. The field is the message's existing Unix second formatted in fixed UTC+09:00 as `YYYY-MM-DDTHH:MM:SS+09:00`; values whose JST local year is outside RFC3339's 0000..9999 range render `unavailable`. Quote-candidate metadata and oldest-reachable boundary metadata remain unchanged. No API call, request, effect, auth behavior, or reference flow changes.

### Layer changes

- Domain: None.
- Application: None.
- Infrastructure: None.
- CLI and catalog: Parse/strip `--jst`, declare conditional output field, preserve exact-help limits, and pass presentation option through byte-budget redraws.

### Data and control flow

Validated boolean flag -> CLI presentation options -> fixed-offset renderer. No value reaches application or infrastructure.

### Error and cancellation behavior

No new operational failures. Invalid `--jst=<value>` is rejected by existing boolean-flag parsing before authentication/provider I/O. Unrepresentable timestamp values render the documented `unavailable` value.

### Security and public boundary

No new assets or side effects. The new field is derived solely from typed `SendTime`; external text, references, and provider data are not modified.

## Implementation slices

1. Add formatter and renderer tests for normal, midnight, boundary, int64-extreme, context, and default compatibility cases.
2. Add catalog and CLI option handling plus request/provider guards.
3. Prove combined output/body-budget redraws and exact help size.
4. Update documentation and the relevant readiness scenario.
5. Run focused tests; root agent runs the repository gate.

## Verification

- Unit and contract tests: fixed-zone bounds and capsule output tests.
- Negative side-effect tests: provider/auth guard confirms no request field or extra call.
- Structured output/error, hostile-output, and recovery tests: existing contracts remain unchanged; default goldens pass.
- Agent-readiness scenario and discovery-round-trip count: update the relevant list/show scenario to invoke `--jst` directly with zero extra provider calls.
- Manual observation: compare `messages list --help` to the 2,048-byte limit.
- Required profiles: `task check` by the root agent.
- Generated-diff or artifact checks: none.

## Rollout and rollback

Additive optional flag; omitting it preserves output. Rollback removes the flag and conditional field without affecting requests or stored state.

## Documentation promotion

Promote behavior to `docs/01_product_contract.md`, `docs/02_architecture.md`, `docs/04_harness.md`, README and `.agents/skills/add-capability/SKILL.md`; update the `messages list` scenario in `docs/09_agent_readiness_validation.md`.
