# Work Context: Chatwork 結果と変更意図の完全性

確認日: 2026-07-19 JST。ここでは確認済み事実と未確認事項を分ける。

## Baseline before this change

- `internal/infra/chatworkapi/transport.go` は message limitation headers を読まず、`messages list` の 204 を常に通常空、`messages show` の 404 を常に `chatwork_not_found` にする。
- `internal/infra/chatworkapi/notation.go` は malformed To/reply/quote、未閉鎖 code、複数 reply を `chatwork_notation_malformed` にし、`response.go` の `mapMessages` は一件の error で一覧全体を破棄する。
- `internal/cli/chatwork_catalog.go` は `rooms create --owner` を認証済みアカウントの作成スコープとして宣言するが、`request.go` の POST `/rooms` form に owner 項目はない。
- `internal/infra/chatworkapi/auth.go` の PAT session は `AccountID` が空で、`SubjectID` は provider identity ではない固定文字列である。現行 `--owner` は認証主体と比較されない。
- `invite-link update` の code と approval はどちらも任意で、`chatwork.Request.Validate` は空更新を受理する。`request.go` は空 form の PUT を構築できる。
- infrastructure の invite form は `description` を送れるが、invite-link create/update の catalog には対応入力がない。
- 429 mapping は未確認の `Retry-After` を相対秒として読み、公式 `x-ratelimit-reset` を無視する。catalog は read/mutation とも同じ retryable=true を宣言する。
- 全 operation の transport attempt は 1、mutation idempotency は unsafe であり、自動再試行は現行も行わない。

## Relevant structure

- Entry point: `internal/cli/chatwork.go`
- Domain rule: `internal/domain/chatwork`, `internal/domain/fault`
- Application use case: `internal/app/chatworkcmd`, `internal/app/authn`, `internal/app/execution`
- Infrastructure boundary: `internal/infra/chatworkapi`
- CLI catalog or presentation: `internal/cli/chatwork_catalog.go`, `internal/cli/capsule`
- Existing tests and harness checks: domain/application/adapter/CLI contract tests, `.harness/chatwork_api_v2.json`, `docs/04_harness.md`

## Constraints

- typed semantic result が relation、coverage、uncertainty を presentation より先に所有する。
- 外部本文と provider prose は untrusted data のままで、既存 visible escape を通す。
- 200/204/404 の status だけで message restriction を推測しない。
- mutation は exact target、complete impact、confirmation、one attempt、read-only reconciliation を維持する。
- PAT と error body は infrastructure 外へ出さない。
- 公式で定義されない omission、relation、reset 値を補完しない。

## External facts

### メッセージ取得と閲覧制限

- Chatwork「チャットのメッセージ一覧を取得する」: https://developer.chatwork.com/reference/get-rooms-room_id-messages （checked 2026-07-19）。最大 100 件、force=0 は差分、force=1 は最新範囲、通常 0 件は 204。
- Chatwork 2022/09/06 API change notice: https://developer.chatwork.com/changelog/2022-09-06-notice （checked 2026-07-19）。list は 200 + `chatwork-message-limitation:true` が一部制限、204 + true が当該取得範囲の全件制限。show は 404 + true が指定対象の制限。非適用時は header を省略し `false` を返さない。
- Chatwork 2024/08/29 notice: https://developer.chatwork.com/changelog/20240829-notice と https://go.chatwork.com/ja/news/update/202427.html （checked 2026-07-19）。旧 5,000 件条件は撤廃済み。現行 Free plan は直近 40 日制限であり、旧条件を current contract に残さない。

### Chatwork 記法

- Chatwork「メッセージ記法について」: https://developer.chatwork.com/docs/message-notation （checked 2026-07-19）。公式形は To `[To:{account_id}]`、reply `[rp aid={account_id} to={room_id}-{message_id}]`、quote `[qt][qtmeta aid={account_id} time={timestamp}]...[/qt]` と time 省略形。quote に message_id はない。
- Chatwork Help「投稿に装飾はできますか？」: https://help.chatwork.com/hc/ja/articles/203127904-%E6%8A%95%E7%A8%BF%E3%81%AB%E8%A3%85%E9%A3%BE%E3%81%AF%E3%81%A7%E3%81%8D%E3%81%BE%E3%81%99%E3%81%8B （checked 2026-07-19）。完全な `[code]...[/code]` は内側を文字列として保持する用途が確認できる。API response での backtick 正規化や不完全 code の意味は未確認。

### Room creation と invite link

- Chatwork「グループチャットを作成する」: https://developer.chatwork.com/reference/post-rooms （checked 2026-07-19）。入力は name（1..255文字）、description、link 系、members_*_ids、有限の icon_preset で、owner 指定はない。
- Chatwork「自分の情報を取得する」: https://developer.chatwork.com/reference/get-me （checked 2026-07-19）。同じ credential が表す account_id を確認できる read operation である。
- Chatwork「チャットへの招待リンクを変更する」: https://developer.chatwork.com/reference/put-rooms-room_id-link （checked 2026-07-19）。公開入力は code、need_acceptance、description。code 省略はランダム文字列、need_acceptance の default は 1 であり、PUT を patch とみなす根拠はない。

### Rate limit

- Chatwork「エンドポイントについて」: https://developer.chatwork.com/ja/docs/endpoints （checked 2026-07-19）。一般上限は 5 分 300 回。`x-ratelimit-limit`、`x-ratelimit-remaining`、`x-ratelimit-reset`（次回 reset の Unix 秒）を定義する。
- 同じ応答の有効な HTTP `Date` がある場合は reset と同じ provider clock domain の待機基準に使い、ない場合だけ local clock を使う。両者から見て公式 5 分窓を外れる reset は不明として扱う。
- 同資料は POST `/rooms/{room_id}/messages` と POST `/rooms/{room_id}/tasks` の合算で、同一 room 10 秒 10 回の追加制限を定義し、完全一致する error では約 10 秒待つよう案内する。
- Chatwork 2025/04/03 notice: https://developer.chatwork.com/me/changelog/202501-notice （checked 2026-07-19）。現行応答例も x-ratelimit-* を使い、公式資料に `Retry-After` は確認できない。

## Unknowns

- [x] 制限で隠れた件数・ID: response から取得不能。公開結果は件数を作らず「一部/当該窓の全件」だけを表す。
- [x] 不完全・入れ子・複数 reply の provider 意味: 公式未定義。relation set を unknown にし本文を保持する。
- [x] PUT invite-link の omission: code と need_acceptance の暗黙動作は文書化済み。description omission は保持/消去を断定できないため、update では全設定を明示する。
- [x] 429 の `Retry-After`: 公式根拠なし。読まない。
- [x] local/provider clock skew: 有効な `Date` があれば `reset-Date`、なければ `reset-local`。どちらの基準から見ても 5 分窓内の場合だけ待機時間を公開する。
- [x] invite description に空文字を送ることで clear になるか: schema は string を許すが状態遷移の明記はなく、一次資料では確定不能。推測を公開契約にせず、update は非空説明を必須として clear を意図的に非対応にする。

## Thesis evidence

- HTTP status だけを成功/空/不存在の意味にすると、provider が同じ status に付加した completeness signal を失う。
- raw body 記法を「すべて parse できるか、結果全体を失うか」の二択にすると、外部本文が supported read の可用性と truthfulness を支配する。
- optional PUT field を patch とみなすこと、認証 token と caller-supplied account を同一とみなすこと、非公式 retry header を使うことはいずれも同じ「provider omission を product fact に昇格する」失敗である。
- Axiom 3 と 7 を、message access limitation、relation parse state、authenticated account binding、explicit replacement、rate-limit evidence へ伝播する必要がある。

## Reproduction or observation

```sh
go test ./internal/infra/chatworkapi ./internal/app/chatworkcmd ./internal/cli ./internal/domain/chatwork ./internal/domain/fault
```

変更前 baseline は成功。missing coverage は synthetic local HTTP server と pure parser fixtures で再現し、live credential は使わない。

## Security and public-boundary notes

- Assets and side effects involved: Chatwork message confidentiality、room creation、invite-link exposure、rate-limit budget。
- Credentials or confidential data involved: `CWK_API_TOKEN` は既存 private binding 内。`/me` identity probe も同じ binding と固定 destination を使う。
- New dependencies, destinations, files, processes, or generated content: なし。
- External schema provenance, publication rights, and drift evidence: 既存 synthetic fixture と公式 URL の要約のみ。provider prose を fixture 以外へコピーしない。
- Pagination, timeout, retry, idempotency, and cancellation facts: message window 100、20 秒、1 attempt。identity probe は room create の mutation 前 read 一回。mutation は unsafe/no automatic retry。
- Publication and licensing concerns: synthetic IDs、example domains、固定時刻だけを tests に使う。

## Glossary

- **access limitation**: Chatwork が message-limitation header で明示した閲覧制限。100 件 window の incomplete history とは別軸。
- **normal empty**: list の 204 かつ limitation header なし。当該 invocation の結果が 0 件であり、room history 不存在の証明ではない。
- **relation unknown**: 不完全記法により reviewed relation set の完全性を証明できない状態。relation absent とは異なる。
- **authenticated account scope**: 同じ PAT で `/me` が返した exact account reference。Chatwork POST `/rooms` の owner field ではない。
