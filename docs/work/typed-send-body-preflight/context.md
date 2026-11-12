# Work Context: Typed send body-file preflight

## Verified facts

- Before this follow-up, `Service.executeMessageSend` performed the exact reply-message GET before the infrastructure adapter resolved `BodyFile`.
- `docs/03_security_model.md` already requires invalid body source to make zero provider calls. A reply GET before file validation violated that contract even if POST remained zero.
- The application `Port` intentionally exposes only semantic task execution and has no filesystem or transport capability.
- Infrastructure already owns `boundedReadMessageBodyFile`, a context-carrying reader that rejects non-regular files and caps reads at 65,536 bytes.
- Prefix size depends on the sender returned by the reply GET, so final expanded-body overflow can only be checked after that GET; it still must make zero POST calls.

## Chosen design

- Add a narrow optional application port for infrastructure to pre-read and validate only a reply send's body file.
- Return an opaque domain marker handle; application code cannot inspect bytes or path.
- Bind the handle to its creating client, exact authentication binding, and send request; consume it once after reply validation without reopening the file.
- Clear the prepared bytes, source path/references, binding, and owner on consume or discard. Redacted `String`, `GoString`, and formatting tests cover body/path disclosure.
- Defer discard in the application path so reply-read errors and match failures clear an unused handle.

## Evidence

- `TestReplyBodyFileValidationFailuresMakeZeroProviderCalls` covers missing/unreadable, empty, invalid UTF-8, and oversized files; each is read once and yields zero GET/POST.
- `TestReplyBodyFileIsPreparedOnceBeforeGETThenSentWithExactSuffix` verifies GET then POST, exact form bytes including caller notation, same context/binding, one file read, and handle cleanup.
- `TestReplyReadFailureDiscardsPreparedBodyWithoutLeakingIt` verifies a successful file preflight followed by reply 404 makes one GET/no POST and clears the unused handle.
- `TestReplyReadSuccessThenExpandedBodyFailureMakesNoPost` covers the later prefix-overflow boundary.
- `TestLostReplySendResponseRemainsUnknownAfterPOST` preserves non-retryable unknown outcome after POST dispatch.
- Focused command: `go test ./internal/app/chatworkcmd ./internal/infra/chatworkapi ./internal/cli ./internal/domain/chatwork` passed after final cleanup changes.

## Unknowns / remaining work

- No implementation issue remains for this follow-up. The root agent owns `task check` and the final gate across concurrent repository changes.
