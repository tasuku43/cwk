# Work Plan: Message send reconciliation

> Historical design: the former all-author candidate scope was superseded by `docs/work/reconcile-own-messages/` and the durable current-account contract.

- Status: Implementation complete; Terra review pending
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Add one `TaskMessagesReconcileSend` read task whose application use case asks infrastructure to validate and snapshot a body before provider reads, optionally fetches and validates the exact reply target, finalizes an opaque exact-payload matcher using infrastructure's existing send expansion helper, then reads one `force=1` message window and returns a body-free candidate projection. The existing typed task request and message-list result remain application inputs; the reconciliation-specific file reader returns no contents, the matcher returns only booleans, and application drops temporary source-body references immediately after comparison. Reuse the authenticated task port and exact provider request adapters; never route the new logical task directly to HTTP.

Add `messages reconcile-send` as a catalog-backed read-only discover command that requires room and exactly one of the existing literal/stdin or body-file sources, accepts the same ordered recipient and optional reply references, produces canonical candidate message refs, and has the same authentication requirement as other message tasks. Unknown message-send mutation outcomes point to this exact catalog path. Namespace/profile validation then requires the recovery task be enabled with the send command.

Render a headerless fixed task result with outcome unknown, candidate status/count, source count/limit/completeness/access limitation, all canonical candidate refs, sender refs and escaped names, and Unix send times. Include fixed Japanese uncertainty text. Do not print body, path, or additional provider fields.

## Alternatives considered

### Compare raw body and typed-prefix body as separate candidate kinds

Rejected by the owner decision: broadens candidate matching and can blur the exact payload generated for the original send. One exact full expanded-body equality is the reconciliation predicate.

### Use only parsed reply/recipient semantics

Rejected because the current parser deliberately keeps relation semantics separate and does not preserve exact body positions or byte suffix. Exact full-body comparison through the common send builder is the reviewed contract.

### Return file bytes to the application

Rejected because the existing body-file contract keeps filesystem access and prepared body bytes inside infrastructure. The app receives only comparison booleans from a one-use opaque matcher.

### Filter to the authenticated sender

Rejected because the current command has no typed authenticated-account identity in the same result and candidate matches may be earlier, duplicate, delayed, or authored by another account. Return all exact candidates with their authors.

## Verification

- Domain request/result validation tests for exact body source, limits, source metadata, candidate uniqueness, and unknown outcome.
- Application tests for call order, same context/binding, optional reply GET, exact all-candidate selection, and no send port call.
- Infrastructure tests for safe file snapshot, common expanded body builder parity, byte-exact matching, binding/request ownership, one-use consumption, zeroization, and no HTTP write.
- CLI/capsule/catalog tests for body source XOR, exact references, escaped hostile author names, no body/path output, and recovery closure.
- A synthetic agent-readiness transcript reuses room/body/recipients/reply inputs, invokes only `messages reconcile-send`, reads only displayed exact candidates, and makes no external grep/parser call.
