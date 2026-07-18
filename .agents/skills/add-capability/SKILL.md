---
name: add-capability
description: >
  Use when adding or changing a user-visible CLI command, workflow, integration,
  side effect, output contract, or external adapter. Guides thesis alignment,
  layer ownership, tests, documentation, and repository gates.
---

# Add a CLI Capability

Before designing the change, run:

```sh
go run ./tools/projectmeta --field profile
```

If the profile is `template`, stop and use `$bootstrap-derived-cli`. Do not add
a capability while the repository still has the foundry identity or generic
product reasoning.

Read `docs/00_theses.md` before designing the change. A capability is complete
only when its user outcome, safety boundary, discoverability, and verification
are explicit.

## 1. Define the user outcome

Write one sentence describing what the user can accomplish. Prefer task language
over an upstream API resource or implementation name.

Confirm:

- the capability belongs in this CLI;
- an existing command cannot express it naturally;
- input ambiguity has a deterministic resolution or is surfaced to the user;
- stdout, stderr, exit status, and machine-readable output are predictable;
- all side effects and external destinations are named.
- the command's stable capability ID is `public` in `.harness/capabilities.json`, or the upstream capability remains explicitly `internal`, `deferred`, or `excluded` with a reason.

If the thesis does not decide a design trade-off, update the thesis or an
architecture decision before implementation.

Separate discovery from action. Discovery commands may accept ambiguity and
must return stable, opaque IDs. Acting commands accept one opaque ID or another
explicitly unique selector; they must not guess among candidates. Record each
ID producer, its output field, and every consumer so contract tests can prove
that users and agents can pass the value through unchanged.

## 2. Place responsibilities in the four layers

- `internal/domain`: domain types, invariants, and pure policy.
- `internal/app`: a use case and consumer-owned ports.
- `internal/infra`: external systems and concrete port adapters.
- `internal/cli`: argument parsing, composition, help, and rendering.

Domain code imports no outer layer. Application code does not import
infrastructure or CLI code. Infrastructure code does not import application or
CLI code. The CLI is the composition root. `tools/archlint` enforces this
contract from the module path reported by `go list`.

Keep policy out of transport adapters and presentation code. Inject clocks,
filesystems, environment reads, network clients, and side-effect executors at
the narrowest useful boundary.

## 3. Declare the operation contract

For every external action, specify:

- effect: read, create, or write; unknown is never executable;
- target, scope, and all generic impact dimensions (cardinality, notification, access change, destructive);
- for create, exactly one required argument/flag opaque `parent_input`, no `target_id_input`, and no other `target_inputs`;
- for write, one required argument/flag opaque `target_id_input` whose reference kind equals `TargetKind`, plus an optional distinct opaque parent role whose input is required when present; `target_inputs` contains only those bound roles;
- validation performed before the external boundary;
- finite timeout, pagination/completeness, maximum attempts, and upstream idempotency behavior;
- which derived policy applies at `app/execution.Invoker`; do not make the template assume approval, confirmation, OS authentication, or dry-run;
- audit-safe fields and secret fields;
- allowed network destination.

Route all equivalent effects through one central enforcement boundary. A new
command must not create a second raw transport or bypass validation.

## 4. Update the command catalog

Add the command to the canonical catalog and derive dispatch and help from that
entry. Complete its `AgentContract`: stable capability ID, user outcome,
described inputs and allowed values, formats, fields/types/descriptions,
completeness, non-auth prerequisites, optional secret-free authentication
requirement, stable faults with exact next commands, and mutation contract when
applicable. Nil collections mean unknown and are invalid; use explicit empty
collections for known none.

For `complete` output, do not declare a pagination binding. For deliberately
`paged` output, declare `AgentContract.Pagination` with the exact optional
cursor argument/flag and exact top-level string cursor output field. The cursor
is emitted beside `schema_version` and the JSON envelope; `CommandOutput.Fields`
describe only values inside the envelope. Both cursor endpoints must use one
dedicated opaque reference kind, and no extra input or output may reuse that
cursor kind. A paged command supports only JSON and defaults to JSON. Pass the
emitted cursor back unchanged. Declare the only supported completion rule,
`completion: "empty_cursor"`, and emit the top-level cursor on every successful
page. An empty string is complete; omission, JSON `null`, and non-string values
are contract failures. Do not use `paged` to make silent truncation or a local
traversal limit look successful.

For a mutation, fill `MutationContract` from required argument or flag inputs rather than
maintaining an informal target description. Treat a missing or unbound role,
optional or non-CLI target, duplicate or extra target input, non-reference input, or target-kind mismatch as
a catalog error. Do not defer that ambiguity to command parsing, policy, or the
adapter.

Do not hand-maintain `ProducedRef` or `ConsumedRef`. Reference compatibility,
workflows, and next actions derive from structured input/output reference kinds.
An act command must require at least one opaque reference. Give semantically
different references different kinds; sharing a kind declares them
interchangeable across every matching field/input edge. Ensure required
reference chains lead back to a command that can run without an unresolved
required reference rather than forming a closed cycle.
Verify that root agent help adds only the command's path, namespace, summary,
capability, outcome, effect, and role. Then use an exact-command or
namespace-scoped invocation to verify the complete contract and workflows.
Root help must not regain inputs, output detail, authentication, errors,
mutation facts, or workflows as the catalog grows, and each encoded command
entry must remain within the 512-byte catalog budget.

Keep every recovery `command` executable under the template's small grammar:
use one exact catalog path, or `help` plus an exact path/canonical namespace.
Do not append flags, values, or guessed selectors. If a derived product needs a
fixed argument-bearing recovery, introduce a typed argument contract and
parser-aware validation before publishing it.

Add bidirectional contract tests so every public catalog entry has a dispatch,
help, and fixture, and no removed or internal entry remains exposed. For JSON,
compare the executable schema version, envelope, and item keys with
`CommandOutput`; do not accept a declaration-only or renderer-only change.

Keep command paths disjoint from their word-boundary namespaces. Match argv
`Required` flags to bracketed versus non-bracketed usage syntax and keep a
written `a|b` list exactly aligned with `AllowedValues`; do not apply this
grammar to stdin, environment, or configuration inputs. Declare the common
cancellation/output failures and, for mutations, every standard invoker
contract or policy failure before exposing the command.

Treat mutation cancellation by phase. Before the action, `operation_canceled`
is retryable because the invoker proves zero attempts. After the action begins,
return a valid structured adapter fault for a known classification and let the
invoker strip its private cause. Never return a raw cancellation as proof that
the write did not happen. Unclassified post-action errors become non-retryable
`unclassified_mutation_outcome`; declare that code in the mutation catalog and
point its next action only to an exact read/discover reconciliation command.

## 5. Add authentication and API boundaries only when needed

Read `docs/07_authentication.md` and `docs/08_external_api_contracts.md` for an
external API capability.

- Keep raw PATs, OAuth tokens, refresh material, token sources, authorization
  headers, and credential-store handles inside `internal/infra`.
- Use `app/authn.Gate` with a secret-free requirement/session and prove auth,
  permission, mismatch, and cancellation failures make zero downstream calls.
- Treat non-nil `AgentContract.Authentication` as a binding to that gate. Declare
  every standard gate fault with its exact code, kind, and retryability, plus
  each provider-specific pass-through fault and a command-valid recovery action;
  catalog validation rejects an incomplete standard set before dispatch.
- Make each authenticated application task port accept `authn.BindingID`.
  Pass the ID from the validated `Session` unchanged; the infrastructure
  adapter resolves that process-local binding and revalidates or refreshes the
  exact private authentication record immediately before I/O. Never pass an
  OAuth client, token source, PAT wrapper, authenticated transport, provider
  SDK type, or credential-store handle into application code.
- Call `authn.NewBindingID` only from production infrastructure. Architecture
  lint rejects issuance in domain, application, CLI, and command packages, so
  argv or configuration values cannot be promoted into authentication bindings.
- Treat gate expiry as admission metadata rather than a lease. Keep refresh
  thresholds, reuse, cache/storage, account selection, and approval policy in
  the derived security model, while making stale or mismatched bindings fail
  before a provider task request.
- Do not implement OAuth protocol machinery. Add a reviewed OAuth library only
  in a derived project whose accepted security model selects OAuth.
- Make one-page adapters return an opaque cursor envelope. Use bounded
  complete traversal, or declare a paged public result with the catalog-bound
  optional cursor input and next-cursor output.
- Keep wire DTOs in infrastructure. Add publishable fixtures under `testdata`
  and bind their path, digest, provenance, and license in
  `.harness/schemas.json`.
- Map provider failures once into stable `fault.Error` values. Never expose a
  raw response body or cause.

## 6. Test at each boundary

Add the smallest set that proves the capability:

- domain tests for invariants and edge cases;
- application tests with fake ports for ordering and failure behavior;
- adapter contract tests for exact requests and bounded responses;
- CLI tests from argv through stdout, stderr, exit code, and captured effects;
- rejection tests proving invalid input causes zero external calls;
- catalog tests rejecting missing, extra, duplicate, optional, non-CLI, non-opaque, and reference-kind-mismatched mutation bindings;
- catalog tests rejecting optional act references and closed required-reference cycles;
- authentication/policy/cancellation tests proving zero downstream mutation;
- mutation outcome tests proving structured deadline/cancellation causes retain their typed classification, unstructured post-action errors are non-retryable, confirmed success is not overwritten, and reconciliation cannot point to a mutation;
- authentication-binding tests with simultaneous accounts/authorities,
  missing, stale, wrong-account, and cross-session IDs, typed-nil task ports,
  expiry races, refresh identity mismatch/failure, and zero unintended provider
  task requests;
- pagination tests for empty, one, many, repeated-cursor, budget, and mid-page failure paths;
- catalog tests rejecting missing, extra, required, non-string, non-opaque, and
  reference-kind-mismatched public cursor bindings;
- hostile-output tests for ESC/newline, bidi and zero-width format characters, U+2028/U+2029, pre-existing backslashes, JSON-looking and prompt-like printable data, oversized content, and writer failure;
- tests proving structural escaping does not claim to filter semantic instructions and does not change an opaque reference;
- regression fixtures for stable TSV/JSON output and structured error output.

Tests must use temporary directories, fixed clocks, fake credentials, and local
test servers. They must not require a developer account or live network.

## 7. Keep claims enforceable

Update the claim-to-enforcement table in `docs/04_harness.md` when a safety claim
changes. Update architecture and operating documentation when a boundary
changes. Do not rely on prose alone when a lint, type, test, or workflow can
enforce the rule.

Run:

```sh
./scripts/check.sh fast
./scripts/check.sh full
./scripts/check.sh security
```

Before public release, also confirm `./scripts/check.sh public` passes with
`profile: ready` in `.harness/project.json`.

## 8. Validate the agent journey

Replay the relevant scenario from `docs/09_agent_readiness_validation.md`.
Record how many invocations were needed to discover the task, where each input
came from, whether output passed unchanged into the next task, and whether each
failure selected a next command without prose interpretation. Extra command
guesses are thesis/product evidence, not an agent workaround to document.

## 9. Feed implementation learning back into the thesis

Implementation is an iterative design probe. When code or tests reveal a new
constraint, do not leave the decision only in a local comment. Revisit the
thesis, refine it when the lesson is general, then propagate that decision into
architecture documentation, the command catalog, typed contracts, tests,
linters, and this skill. The repository should become less ambiguous after
each capability is added.
