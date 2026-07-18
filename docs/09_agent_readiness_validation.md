# Agent Readiness Validation

This validation asks whether a coding agent can discover, execute, interpret, and recover from representative API-backed tasks with few CLI round trips. The scenarios are intentionally synthetic and public-safe. They model a project-collaboration CLI and a team-chat CLI without embedding a private roadmap, endpoint, account, or credential.

## What counts as a round trip

A round trip is one CLI invocation whose purpose is to learn what invocation should come next. The task invocation itself and a necessary authentication ceremony are counted separately. Parsing a declared JSON or TSV field is not an additional discovery round trip; scraping prose, guessing a URL, or probing variants is.

The target is:

- unknown surface to scoped task contract: at most two discovery invocations;
- known task path to executable invocation: one scoped help invocation;
- discover reference to read/write: no extra lookup or transformation invocation;
- classified failure to next corrective command: no prose interpretation or command guessing.

## Contract-level validation method

For each derived command, verify all four stages.

| Stage | Evidence |
|---|---|
| Discover | Root `view: index` exposes path, namespace, summary, capability, outcome, effect, and role plus a machine-readable `scope_request`; selected `view: scope` declares inputs, input sources, prerequisites, effect, output, authentication, errors, mutation facts, and workflow edges |
| Execute | Arguments are copied from declared fields or explicit configuration; the resolved command, effect, create-parent/write-target binding, runtime target, auth requirement, and impact validate before I/O |
| Interpret | Machine output has declared fields/types/completeness; structural runes are visibly projected; scoped I/O metadata marks external text as untrusted data; opaque references remain validated exact values |
| Recover | Failure kind/code/retryability/next actions are structured; auth, permission, ambiguity, missing targets, rate limits, temporary failure, cancellation, and contract failure remain distinct |

## Scenario A: project-collaboration CLI

### Outcome

Find a project by a human filter, obtain its canonical reference, and read its current summary.

### Expected path

1. The agent reads the compact root outcome index, chooses `commands[].path` or `commands[].namespace`, and applies the published `scope_request.invocation_template` without guessing help syntax.
2. Scoped help identifies a `discover` command, its filter input, authentication/scopes, exhaustive or paged output contract, and its produced `project` reference field.
3. The agent runs discovery in a machine format and selects an exact `project_id`. Multiple candidates remain data, not a hidden choice by the later action.
4. Scoped help for the read action declares `--project-id` as consuming that reference kind.
5. The agent passes the exact emitted bytes into the read action. It does not parse a browser URL, normalize case, or call discovery again.
6. The result declares whether it is complete and names every stable output field.

### Recovery probes

- No credential: `authentication`, not `permission`; next action names the configured login/status command.
- Valid identity without scope: `permission`; retrying login is not claimed to fix it unless the derived flow can request additional scope.
- No matches: successful empty discovery or a documented `not_found`, never a fabricated reference.
- Multiple matches: discovery returns candidates or `ambiguous`; action is not attempted.
- Stale project ID: `not_found` with discovery as a next action.
- Page cursor loop or local bound: contract failure, no partial successful output.
- Rate limit: `rate_limited`, retryable metadata, bounded retry-after; no duplicate logical operation.

### Acceptance

An agent that knows only the desired outcome reaches the read command with at most two discovery invocations, then reuses the reference without transformation. Every recovery probe selects its next action from structured metadata.

## Scenario B: team-chat CLI

### Outcome

Find a room, inspect its metadata, then send one message to the explicitly selected room.

### Expected path

1. A scoped query identifies room discovery and declares the exact output field carrying the room reference.
2. The read action consumes the same room reference and makes no hidden name search.
3. The send action declares `create` or `write` according to the derived thesis, cardinality `one`, notification `yes`, access-change/destructive declarations, authentication/scopes, idempotency behavior, and stable result fields. Creating a new message binds the selected room reference as `parent_input` and has no `target_id_input`; changing an existing message binds the message reference as `target_id_input` and may bind the room as a distinct `parent_input`.
4. The application mutation invoker validates the runtime intent and applies the project's policy. The template does not assume whether that policy is human confirmation, dry-run, OS authentication, or a role check.
5. The infrastructure adapter performs one logical send. It retries transport only if the upstream operation is safe or uses one stable idempotency key.
6. The result returns the canonical message and room references needed by later reads or updates.

### Recovery probes

- Room name supplied where an ID is required: `invalid_input`; next action is room discovery.
- Room reference maps to multiple accounts/tenants: `ambiguous`; account-selection command is explicit.
- Missing send scope: `permission`; zero send attempts.
- Policy denial or missing impact dimension: `rejected` or `contract`; zero send attempts.
- Missing, extra, non-opaque, or reference-kind-mismatched mutation binding: catalog/contract rejection; zero send attempts.
- Timeout before execution: `canceled`/`unavailable`; zero or explicitly classified transport attempts.
- Timeout after an unknown upstream result: do not claim a safe retry unless idempotency proves it; provide a read/status action when available.
- Hostile room/message text: raw controls, format runes, line/paragraph separators, and delimiters cannot alter terminal or TSV/JSON structure. Existing backslashes remain distinguishable from projected controls. Printable JSON-looking or prompt-like prose remains present as untrusted data; the CLI makes no semantic prompt-injection-prevention claim.

### Acceptance

The agent never sends to a room selected implicitly by display name, can identify the exact input supplying the create parent or existing write target, can tell that sending has a notification side effect before executing it, and does not repeat an unsafe send after an uncertain failure. It treats every external text field as data rather than as a CLI-authored instruction.

## Runnable template probes

The synthetic sample flow is the executable minimum for these scenarios:

```sh
go run ./cmd/agentic-cli-foundry help --format agent
go run ./cmd/agentic-cli-foundry help sample --format agent
go run ./cmd/agentic-cli-foundry sample list --format json
go run ./cmd/agentic-cli-foundry sample read --id smp_2f4a6c8e0b1d --format json
go run ./cmd/agentic-cli-foundry --error-format json sample read --id smp_000000000000
```

The root agent contract must be schema version 3 with `view: index`, reveal the `sample` namespace and both exact paths, and contain no input, output, authentication, error, mutation, or workflow detail. Its `scope_request` must identify the selector fields, exact invocation template, two-invocation unknown-outcome bound, and one-invocation known-path bound. The scoped contract must use `view: scope`, contain only the relevant list/read commands and their reference workflow, and provide the complete global and command contracts. Its `io_contract` must publish `external_text_trust: untrusted_data`, `external_text_projection: visible_escape`, and `opaque_reference_policy: validated_exact_bytes`. The `id` selected from the list JSON is field extraction, not identifier transformation: pass its exact string bytes to read. The final probe must fail as `not_found`, use the dedicated exit status, write no success data to stdout, and name `sample list` as the structured next action on stderr.

Validation must also cover:

- every list-emitted sample ID passed unchanged to read;
- URL, name, partial, uppercase, whitespace, and control-character variants rejected before repository access;
- catalog/output snapshots detecting field or semantic changes;
- root-versus-scoped agent-help shape snapshots and a per-command root-size growth bound;
- executable checks that JSON schema versions, envelopes, and item keys equal their `CommandOutput` declarations;
- adversarial TSV/JSON/stderr fixtures containing ESC, actual newline, bidi and zero-width format runes, U+2028/U+2029, literal backslash escapes, JSON-looking fragments, and prompt-like printable text;
- exact opaque-ID round trips alongside hostile labels/content, proving presentation never rewrites identity;
- complete pagination or no result;
- typed not-found recovery pointing back to discovery;
- structured contract visibility for effect, prerequisites, fields, completeness, errors, and next actions.
- declared default formats, JSON envelopes/schema versions, stdout/stderr ownership, and the complete exit-code map;
- successful output emitted only after complete pagination, validation, bounding, and rendering;
- root help that never embeds complete command contracts, plus namespace/exact scoped help that does not force the agent to ingest unrelated detail.

The sample is not evidence that a real API adapter is secure. A derived CLI repeats the scenario with fake adapter fixtures, authentication failures, pagination, cancellation, policy denial, and upstream error mappings before enabling a real network integration.

## Review record

Record the invocation transcript, number of discovery round trips, selected output/reference fields, and each recovery probe in the active work packet. If an agent needs prose interpretation, source inspection, URL parsing, hidden filtering, or an extra command guess, treat that as product/thesis evidence rather than teaching the agent a workaround.
