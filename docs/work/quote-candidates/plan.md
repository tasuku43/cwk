# Plan

## Chosen approach

Add an application-owned typed quote-candidate analysis distinct from the
provider quote relation. Compute it against the whole bounded source once,
using exact same-room sender and timestamp equality. Keep state and candidates
unresolved, retain `source-window` scope, canonical candidate message refs,
optional source sequences, and an explicit identity-unproven marker. Carry the
analysis through result validation and output planning without recomputing it
on selected subsets.

For `messages show`, use a recent source only when the existing reply or around
workflow already fetched it. Exact-only show reports source unavailable and no
candidate. Never fetch based on quote metadata.

Split unresolved output counts by relation kind and retain unknown-relation-set
count. Update semantic fixtures, answers, readiness scenario, contract docs,
skill, and active goldens together.

## Risks and controls

- **Subset ambiguity:** compute from the original source before local selection;
  later steps may only remove dangling optional sequence values.
- **Identity overclaim:** quote candidate remains unresolved in type validation
  and output. No candidate creates a reply edge or context fetch.
- **Partial source:** keep source coverage/access alongside the result and call
  candidates window-scoped, not globally unique.
- **Exact-only show:** represent absent source without probing or creating a
  candidate.
- **Compatibility:** accepted text field changes require active golden updates
  and an explicit pre-1.0 contract statement.

## Verification

Add unit, application, validation, renderer, golden, semantic-answer, and
readiness checks for exact and near timestamps, ambiguous/no/missing/unknown
matches, partial/unknown sources, filtered and truncated delivery, source order,
overflow, hostile output, exact show modes, no quote fetch, and no inferred
reply edge. Run the repository's prescribed `task check` gate after integration.
