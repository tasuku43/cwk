# Tasks: Chatwork User-Agent

- [x] Validate injected version in infrastructure and build the safe header value.
- [x] Pass `CLI.Version` into the lazy infrastructure factory.
- [x] Set User-Agent in the shared authenticated HTTP execution path.
- [x] Test accepted and rejected version forms, including safe fallback.
- [x] Test exact User-Agent and PAT headers across the API task fixture corpus.
- [x] Test the lazy CLI factory passes the configured version scalar.
- [x] Record the contract in security and external API docs; add harness enforcement and add-capability guidance.
- [x] Run focused infrastructure and CLI factory tests; report unrelated package failures separately.

## Verification evidence

- `go test ./internal/infra/chatworkapi` — passed.
- `go test ./internal/cli -run '^TestNewBuildsChatworkAdapterLazilyWithInjectedVersion$' -count=1` — passed.
- `go test ./internal/cli` — one concurrent #12 test currently fails because `TestRunMembersListQueryNamesMembersFindBeforeAuthentication` supplies unsupported `--query` to `members list`; the focused User-Agent test passes.
- `git diff --check` for the touched implementation, contract, and packet files — passed.
