# Work Plan: Member name discovery and command-help recipes

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Add `members find` as a read-only discover command. It consumes an exact room
reference plus a display-name query, reuses one existing complete member-list
provider call, and returns locally filtered candidates with explicit selection
provenance. Add structured human-help recipe metadata to command catalog data;
exact-command text help renders a recipe when the selected command is one of
its steps and every step is present in the active catalog.

## Alternatives considered

### Let `--sender` accept a name

Rejected because an act command would rediscover and silently bind external
display text, breaking the discover/act boundary and making ambiguity unsafe.

### Add `--query` to `members list`

This is mechanically smaller, but the compact root agent index would still
present the task as a complete member listing. A dedicated outcome-named
`members find` makes the intended discovery step selectable before scoped help.

### Put free-form recipe strings directly in help rendering

Rejected because examples could drift from routing and active command
selection. Structured command metadata keeps exact paths auditable, lets each
exact help pull related recipes, and removes unavailable workflows as one unit.

## Layer changes

- Domain: add the member-find task, validated query, and explicit selection
  metadata.
- Application: translate member-find to one members-list provider request,
  preserve typed order, filter exact substrings, and validate the derived result.
- Infrastructure: keep the existing members endpoint unchanged and prove the
  query does not cross the port.
- CLI/catalog/presentation: bind `--query`, declare discover/reference flow,
  render candidate provenance, point `--sender` help to discovery, and render
  related catalog-backed recipes in exact human help.
- Readiness/docs: add the two-command name-to-sender scenario and promote the
  help/discovery contract.

## Verification

- Domain truth tables for missing/invalid query and inconsistent selection.
- Application tests for zero/one/many matches, stable order, exact substring
  semantics, one provider call, and no query leakage.
- Catalog/parser/help tests for roles, refs, input binding, exact-help relevance,
  active-view recipe suppression, and unchanged root/namespace/agent indexes.
- Capsule goldens for explicit zero and ambiguous candidates plus hostile text.
- Synthetic readiness for `members find` then `messages list --sender`.
- Final `task check`.

## Rollout and rollback

This is an additive pre-1.0 read capability and human-help extension. Removing
the new leaf and its recipes restores the old behavior; no remote or local data
migration exists.
