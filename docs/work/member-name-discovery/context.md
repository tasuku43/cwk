# Work Context: Member name discovery

## Verified current behavior

- `messages list --sender` accepts only canonical `chatwork-account` references.
- `members list` returns every member of one exact room, including display name,
  role, and canonical account reference.
- Chatwork exposes no reviewed server-side member-name query for this endpoint.
- Root human help collapses leaf commands into namespaces; compact agent root
  help exposes each leaf summary but no detailed inputs or workflow.
- The R7 sender benchmark used five tool calls but still loaded the complete
  message window because the agent did not naturally discover an account ref.

## Constraints

- Name matching is discovery, not identity binding. Display text cannot become
  an act-command target without an explicit canonical reference choice.
- Matching must run over the one complete room-member response and perform no
  additional external I/O.
- External member names remain untrusted and visibly escaped in output.
- Recipes are exact-command human navigation hints, not a second router. Their
  command paths must resolve against the active catalog, the selected command
  must be one of their steps, and hidden commands suppress the whole recipe.
- The root and namespace agent index size contract must remain unchanged.

## Chosen semantics

- Query matching is exact, case-sensitive substring matching over valid UTF-8.
- No whitespace normalization, Unicode normalization, transliteration, fuzzy
  match, or ranking is performed.
- Candidate order remains provider member order.
- Empty, unique, and ambiguous results use the same candidate schema; cwk never
  promotes a candidate to a selected identity.
- The provider query remains `GET /rooms/{room_id}/members`; the application
  maps the derived task to that existing read and filters the typed result.

## Security and public-boundary notes

- The query and display names are not credentials or opaque identities.
- The room and returned accounts retain validated exact opaque references.
- All fixtures use synthetic names and numeric references.
- Authentication, call bounds, fixed destination, cancellation, and provider
  fault mapping remain unchanged.

## Implementation evidence

- `TaskMembersFind`, `MemberQuery`, and `MemberSelection` bind the derived
  outcome and prove that every returned account name contains the exact query.
- Application maps the public task to one filter-free `TaskMembersList` port
  request, preserves provider order, and returns non-nil explicit empty results.
- The adapter rejects a leaked application-derived task; the provider operation
  corpus remains 33 task mappings over the fixed 32-operation snapshot.
- `members find` is a discover-role catalog leaf whose canonical account output
  forms a generated exact-help workflow to `messages list --sender`.
- `messages list` exact human help renders the person and day recipes;
  `members find` exact human help renders only the shared person recipe. Root,
  namespace, and every agent-help shape remain recipe-free. Exact help
  suppresses a recipe whose steps are not all present in the active view.
- The active readiness fixture fixes two cwk invocations, two provider reads,
  zero external processing, and zero full-message pre-dumps.

## Verification evidence

- Focused domain, application, adapter, CLI, capsule, and presentation-eval
  tests passed on 2026-07-20.
- `go test ./...` passed with local httptest sockets enabled.
- `task check:fast` and the required full `task check` passed on 2026-07-20.
- `git diff --check` reported no whitespace errors.
