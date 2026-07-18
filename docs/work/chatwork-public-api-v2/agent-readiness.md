# Agent-readiness evidence

This file records synthetic, repeatable evidence for the fixed completion goal.
It is not a live-account transcript and contains no Chatwork credential or real
provider data.

## Room-message outcome

User request: “Get the messages in this room and preserve replies.”

The public interaction is bounded to selection plus scoped invocation detail:

```text
cwk help --format agent
cwk help messages list --format agent
cwk rooms list
cwk messages list --room <exact room_ref> --window recent
```

The root index identifies `rooms list` as discovery and `messages list` as the
room-scoped read. Scoped help declares the required `chatwork-room` reference,
the exact usage, authentication selection, result fields, coverage semantics,
and typed failures. The caller passes the canonical `room_ref` from the first
context capsule unchanged; a compact display alias is never accepted as input.

`TestRunChatworkRendersResolvedMessageContextWithoutPostProcessing` executes
the final task with synthetic typed messages and proves that the candidate-C
capsule contains both canonical message references and a resolved reply edge.
It needs no `jq`, provider-notation parser, join, grep, or guessed relationship.
`TestScopedAgentHelpIsACompleteProjectionOfEveryCatalogCommand` and catalog
reference-graph validation prove that the discovery and invocation metadata
are derived from the same executable command source.

## Mutation and recovery outcome

Representative message operations use the exact references already emitted:

```text
cwk messages send --room <exact room_ref> --body <text>
cwk messages update --room <exact room_ref> --message <exact message_ref> --body <text>
cwk messages show --room <exact room_ref> --message <exact message_ref>
```

The synthetic runtime tests prove parsing, exact target binding, one
authenticated adapter call, complete-or-no-output behavior, and secret-safe
unknown-result collapse. The adapter matrix executes all 32 fixed HTTP
operations against local servers and checks method, path, form or multipart
shape, bounded response handling, provider faults, and cancellation. Catalog
tests prove that every mutation has an exact read-only reconciliation command
and never recommends replaying a write after an uncertain outcome.

## OAuth outcome

The local public discovery command was executed without an authentication
selector or registration environment and returned:

```text
cwk-auth-profiles/1
profile_ref: cwk_chatwork_oauth_public_v1
method: oauth2
api_selector: CWK_AUTH_METHOD
allowed_api_methods: pat,oauth2
callback_model: authorization_code_pkce_s256_manual_callback
credential_storage: operating_system
```

The synthetic OAuth suite covers state and PKCE S256, exact custom-scheme
redirect verification, callback bounds, code exchange, required-scope checks,
OS-store absence/denial, overwrite refusal, expiry and refresh rotation,
`GET /v2/me` account binding, redaction, local-only logout, and zero Chatwork
task calls after a failed selection or refresh. CLI selection tests prove that
`pat` and `oauth2` never probe or fall back to the other credential source.

Live OAuth authorization and the user-authorized isolated-room sequence remain
an optional environment validation. They are not required by the repeatable
test suite and must not broaden into contact changes, room deletion, membership
changes, invite links, files, or pre-existing data.
