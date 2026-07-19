# Work Goal: Chatwork 結果と変更意図の完全性

- Status: Complete
- Owner: Project owner and Codex
- Target: 2026-07-19 improvement request
- Related ADRs: [ADR 0003](../../decisions/0003-chatwork-pat-only.md)

## Outcome

利用者とエージェントが、Chatwork のメッセージ取得結果について通常の空、閲覧制限による欠落、対象不存在、権限拒否を区別できる。外部本文の不完全な記法は本文取得を失敗させず、関係だけを不明として扱う。ルーム作成と招待リンク更新は、認証済み主体および実際に送信される変更内容と公開 CLI 契約が一致する。レート制限は公式ヘッダーまたは公式に定義された操作別制限だけを根拠に待機情報を示し、変更操作を自動再試行しない。

## Why now

変更着手時の実装は、Chatwork が閲覧制限を通知するレスポンスヘッダーを捨て、一覧の全件制限を通常の 0 件、一件取得の制限を不存在として扱っていた。また、一件の不完全な Chatwork 記法で一覧全体を破棄し、`rooms create --owner` は認証主体と照合せず、`invite-link update` は空フォームや省略による暗黙変更を送信できた。429 応答では公式の `x-ratelimit-reset` ではなく未確認の `Retry-After` を読んでいた。

## Non-goals

- Chatwork の非公開仕様、制限で隠れた件数・ID、または不完全記法の意図を推測すること
- メッセージ本文を信頼済み命令や構造へ昇格すること
- 変更操作の自動再試行、transport attempt の増加、または idempotency の推測
- provider の制限理由本文、429 本文、PAT、認証ヘッダーを公開出力へ転記すること
- 公開 API の新しい operation を追加すること

## Acceptance criteria

- [x] `messages list` は通常の 204、200 + 一部制限、204 + 全件制限を別の型付き結果と出力で表す。
- [x] `messages show` は 404 + 制限ヘッダーを閲覧制限、ヘッダーなし 404 を不存在として別の stable fault にする。
- [x] 不完全な To・返信・引用・code らしき記法は本文を保持し、関係集合を `unknown` として一覧の他項目も返す。
- [x] 完全な公式記法だけが typed relation になり、To・引用・名前・時刻・本文から返信を推測しない。
- [x] `rooms create` の作成スコープは `/me` で確認した同じ PAT のアカウントと一致し、不一致時は作成リクエスト 0 回で失敗する。
- [x] Chatwork に存在しない room owner 入力を公開契約として主張しない。
- [x] `invite-link update` はコードの明示値または明示的再生成、承認設定、非空の説明設定を明示し、空または暗黙の更新を認証・変更 I/O 前に拒否する。
- [x] 招待コードは公式の 1..50 文字と `[A-Za-z0-9_-]` だけを認証前に受け付ける。
- [x] 429 は有効な `x-ratelimit-reset` または公式 room-post 制限の完全一致だけから待機情報を作り、取得不能時は `unknown` を示す。
- [x] read と mutation の retryable 契約を区別し、全 Chatwork operation の `MaxAttempts: 1` を維持する。
- [x] root/scoped help から二回以内で変更後契約へ到達でき、外部パーサーなしで不完全性と次の安全な操作を判断できる。
- [x] `task check`、`task security`、`task public:check` が通る。

## Governing documents

- Thesis: `docs/00_theses.md` Axiom 1, 3, 7, 8
- Product contract section: supported-outcome promise, agent-output axioms, side effects, authentication and external-call decisions
- Architecture or security invariant: semantic/presentation boundary, Chatwork notation trust, controlled execution, error ownership
- Existing ADR: ADR 0003 PAT-only

## Completion definition

受け入れ条件の機械的証拠があり、採用契約と見送り案が durable docs と harness の claims-to-checks へ反映され、focused/full/security/public gate と関連 agent-readiness probe が成功し、秘密・実レスポンス・一時診断を残していないとき完了とする。
