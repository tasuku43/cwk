# Work Tasks: Chatwork 結果と変更意図の完全性

## Understand

- [x] Governing theses、product、architecture、security、harness を指定順で確認した。
- [x] Authentication、external API contracts、agent readiness を確認した。
- [x] `add-capability` skill と ready profile を確認した。
- [x] 5 項目の現行挙動を code/tests で確認した。
- [x] 一次資料、変更履歴、現行 Free plan 差分を `context.md` に記録した。

## Decide

- [x] Status と limitation header を別に扱う contract を選んだ。
- [x] Wire failure と relation enrichment unknown の境界を選んだ。
- [x] `--owner` alias を残さず authenticated `--account` scope に変更することを選んだ。
- [x] invite-link update を full explicit replacement にすることを選んだ。
- [x] rate timing evidence と retryability/automatic retry を分離することを選んだ。
- [x] Durable docs と harness propagation を完了する。

## Implement

- [x] Message limitation domain state と adapter classification を実装する。
- [x] Restricted list/show、normal empty/not found、invalid header tests を追加する。
- [x] Relation parse unknown と body-preserving list/show を実装する。
- [x] Exact notation、malformed To/reply/quote/code、hostile body tests を追加する。
- [x] `rooms create --account` と PAT `/me` exact binding を実装する。
- [x] Mismatch/cancel/rate/auth failures で room-create POST 0 回を証明する。
- [x] Invite full replacement、code validation、empty update pre-auth rejection を実装する。
- [x] Official x-ratelimit-reset、room-specific 10s、unknown timing を実装する。
- [x] read/mutation rate fault contract と one-attempt tests を更新する。
- [x] Catalog、help、presentation、golden、schema contract を更新する。
- [x] Durable documentation と claims-to-checks を更新する。

## Verify

- [x] Focused tests pass. Evidence: `go test ./internal/domain/chatwork ./internal/domain/fault ./internal/infra/chatworkapi ./internal/app/chatworkcmd ./internal/cli ./internal/cli/capsule ./tools/presentationeval -count=1` (2026-07-19, Go 1.26.5)。
- [x] `task check` passes. Evidence: `task check` (2026-07-19, Go 1.26.5) は fast、vet、race、security、release lint、public、contract lint を含めて成功した。
- [x] `task security` passes. Evidence: `task security` (2026-07-19, Go 1.26.5) は module verification、security repoguard、gosec/govulncheck を通過し、呼び出し可能な脆弱性 0 件だった。
- [x] `task public:check` passes. Evidence: `task public:check` (2026-07-19) は public repoguard と contractlint を通過した。
- [x] Synthetic agent-readiness probes meet discovery/provider/post-processing budgets. Evidence: targeted `tools/presentationeval` public-help/exact-reference probes、CLI mutation/rate probes、infrastructure limitation/identity/rate probes all pass (2026-07-19); provider attempts remain 1 except the declared room-create `/me` preflight + POST pair, external post-processing remains 0。
- [x] Generated diff and repository status are understood. Evidence: `git status --short` で変更対象を durable docs、work packet、Chatwork domain/app/infra/CLI、presentation fixtures/tests に限定して確認し、`git diff --check` は成功した。着手時の worktree は clean で、秘密・実レスポンス・生成バイナリは追加していない。

## Hand off

- [x] Acceptance criteria have evidence.
- [x] Durable decisions were promoted out of the work packet.
- [x] Temporary diagnostics and sensitive artifacts were removed。
- [x] Follow-up work is explicit and does not block this goal: invite description の明示的な空文字クリアは一次仕様で状態遷移を確定できるまで意図的に非対応とする。
- [x] Handoff summary explains outcome, why, checks, and remaining risks.
