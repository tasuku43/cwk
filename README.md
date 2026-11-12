![cwk — ノイズを削り、文脈を残すChatwork CLI](docs/assets/readme/cwk-hero.png)

---

`cwk` は、AIエージェントがChatworkのルーム、メッセージ、タスクなどを操作するためのCLIです。Chatwork APIのレスポンスをそのまま表示せず、操作に必要な情報と参照を整理して出力します。

## インストール

```sh
brew install tasuku43/tap/cwk
```

[Homebrew 6.0以降のtap trust](https://docs.brew.sh/Tap-Trust)では、公式ではないtapをFormula単位で信頼できます。上のコマンドはtap全体ではなく `cwk` だけを対象にします。

ビルド済みのアーカイブは [GitHub Releases](https://github.com/tasuku43/cwk/releases) から取得できます。

## クイックスタート

最初に、この `cwk` から実行できるChatworkコマンドを選びます。

```sh
cwk config
```

上下キーで移動し、Spaceで切り替え、Enterで保存します。保存した範囲の外にあるコマンドは、この `cwk` から実行できません。必要なコマンドだけを有効にすると、エージェントの選択ミスとトークン消費も減らせます。

この範囲は現在の `cwk` プロセスのディスパッチに適用されます。同じOSユーザーは設定ファイルや `XDG_CONFIG_HOME` を変更したり、端末入力を自動化したり、別の実行ファイルやPATでChatworkへ直接アクセスできます。コマンド選択は人の識別やOSアカウントの保護ではなく、PATの検証、Chatwork側の権限、対象参照、変更確認とは別の制御です。選択外のコマンドを使う必要があるときは、エラーの `command_disabled` に従って、ユーザーに対話端末から `cwk config` を実行して有効化するよう依頼してください。

<img width="953" height="477" alt="image" src="https://github.com/user-attachments/assets/43d1c5c3-6bf4-4b23-8e50-d04319e39f79" />

ChatworkのAPIトークン（PAT）を環境変数 `CWK_API_TOKEN` に設定します。トークンはシークレットマネージャーなど安全な場所で管理してください。`cwk` 自体が保存することはありません。

認証に失敗したら `cwk doctor` を実行してください。値を表示せずに、`CWK_API_TOKEN` の有無・形式と、設定ファイル／キーチェーンが未対応であることを診断します。現時点の認証探索先はコマンドプロセスの `CWK_API_TOKEN` だけです。

```sh
cwk rooms list
```

`rooms list` が表示する `room-ref` を、そのまま
`--room <room-id>` に渡します。値は先頭ゼロなしの1〜32桁の正の10進数です。

```sh
cwk messages list --room <room-id>
```

結果は、エージェントが少ないトークンで正確に読める形にまとまります。

```text
messages room-ref=4101 count=2 window=recent source-limit=100 complete=false access-limitation=none unresolved-replies=0 unresolved-quotes=0 unknown-relation-sets=0 oldest-reachable-message-ref=9001 oldest-reachable-send-time=1700000000
relation-resolution fetch-limit=5 fetch-attempts=0 targets=0
external-text=untrusted escaped
schema: #sequence message-ref actor sent [reply] [to] [quote] [quote-candidate] [relation-state] "body"
actors
  a1 account-ref=7001 name="Aki"
  a2 account-ref=7002 name="Beni"
#1 9001 a1 1700000000 "Release time?"
#2 9002 a2 1700000010 reply=#1 to=a1 "15:00 works."
```

`room-ref`、`message-ref`、`account-ref` は、次のコマンドへそのまま渡せます。返信などの関係も本文から推測せず、型付きの情報として表示します。

利用できるコマンドはヘルプから探せます。

```sh
cwk --help
cwk messages
cwk messages --help
cwk messages list --help
```

ヘルプ経路はこのtext形式だけです。コマンド選択を設定済みの場合、
`cwk messages` と `cwk messages --help` は同じ名前空間ヘルプを表示します。
設定前はどちらも `command_selection_required` になり、選択設定を安全に
読み取れない場合は静的カタログのヘルプを表示して `cwk doctor` に案内します。
静的ヘルプは完全なカタログを案内するもので、表示されたChatworkタスクを
この実行範囲で利用できるとは限りません。実行可能なコマンドを確かめるには
`doctor` で設定状態を確認してください。
名前空間のヘルプには配下の各
コマンドのusageと必須フラグが出るため、その場でコマンドを書き始められます。
トップのヘルプは、有効なコマンドの成果を名前空間ごとに最大3件示し、
同じ説明はまとめます。続きは名前空間ヘルプで確認できます。
リンクから参照を取り出す案内は `messages resolve-link` が有効な場合に、
参照から発言を読む案内は `messages show` が有効な場合に表示します。
抽出だけが有効なら、読む操作も使えるよう人に設定変更を依頼してください。
無効なコマンドの成果はトップの案内には出ません。
正確なコマンドのヘルプは2KB以内で、usage、フラグ、併用制約、例を最大2件
だけ表示します。失敗時の復旧は、エラーに出る `kind`、`code`、
`next_action` を使います。廃止済みの `--format agent` をhelpへ渡すと、
`--help` を使うよう明示して失敗します。既存の JSON 出力は
`cwk doctor --format json` で引き続き利用できます。Chatwork タスクの
成功出力は従来どおり text 形式です。

## 終了コードとエラー分類

正常終了は `0` です。構造化エラーの `kind` に対応するプロセス終了コードは次のとおりです。`code` は個別のエラーを識別する別の値で、終了コードは `kind` から決まります。

| `kind` / 状態 | 終了コード |
| --- | ---: |
| `成功` | `0` |
| `invalid_input` | `2` |
| `internal` | `3` |
| `authentication` | `4` |
| `permission` | `5` |
| `not_found` | `6` |
| `ambiguous` | `7` |
| `rate_limited` | `8` |
| `unavailable` | `9` |
| `rejected` | `10` |
| `canceled` | `11` |
| `unsupported` | `12` |
| `contract` | `13` |

## よく使う例

ルーム名の一部しか分からないときは、候補を絞ります。大文字小文字を
区別せず、日本語を含む部分一致で全候補を返します。

```sh
cwk rooms list --query コスト
cwk messages list --room <room-id>
```

名前、`rid442804132`、`chatwork-room:442804132`、ChatworkのURLは
`--room` へ直接渡せません。候補に表示された10進数の `room-ref` をそのまま
使ってください。

今日や昨日の会話だけを読みます。

```sh
cwk messages list --room <room-id> --on today
cwk messages list --room <room-id> --on yesterday
```

表示名から人物を探し、その人の投稿だけを読みます。

```sh
cwk members find --room <room-id> --query <name>
cwk messages list --room <room-id> --sender <account-ref>
```

`members find` は一致した候補をすべて返します。候補が複数あっても、cwkが人物を勝手に選ぶことはありません。
候補に表示される `account-ref` は、`--sender` にそのまま渡せます。

ルームタスクを作成する `--limit` には、従来の正のUnix秒に加えて、秒精度RFC3339を
明示タイムゾーン付きで指定できます。cwkがUnix秒に変換します。`--limit-type date|time`
の意味は変わりません。

```sh
cwk room-tasks create --room <room-id> --body "確認する" --assignee <account-ref> \
  --limit 2026-11-08T12:00:00+09:00 --limit-type time
```

件数や期間も絞れます。

```sh
cwk messages list --room <room-id> --count 10
cwk messages list --room <room-id> --start-index 11 --count 20
cwk messages list --room <room-id> \
  --since 2026-07-17T12:00:00+09:00 \
  --until 2026-07-17T18:00:00+09:00
```

旧 `--limit` は `--count` の別名として少なくとも2マイナーバージョン
受理され、成功時にstderrへ改名案内を出します。新しい呼び出しでは
`--count` を使ってください。不明なフラグは、近い有効名があれば候補を
示して失敗し、勝手に置き換えて実行しません。

件数ではなく出力サイズを先に制限したい場合は、1 byte〜16 MiBの予算を
指定できます。

```sh
cwk messages list --room <room-id> --max-bytes 8192
```

`--max-bytes` は表示だけを制限します。Chatworkへの一覧取得は従来と同じ
1回で、取得範囲や `complete`、通常の `next-start-index` は変わりません。
予算で打ち切った結果は、たとえば
`delivery max-bytes=8192 output-complete=false output-items=4 output-total-items=10 output-next-start-index=5`
と絶対最終行に示します。`output-items` は表示できた主メッセージ数、
`output-total-items` はこの呼び出しで配送対象だった主メッセージ総数、
`output-next-start-index` は次に指定する新しい順の順位です。同じ絞り込みを
保ったまま `--start-index 5 --max-bytes 8192` とすれば続きを取得できます。
元の呼び出しに `--count` があった場合は最終行に `output-next-count` も出ます。
続きでは `--count` をその残り件数へ置き換えるため、元の指定範囲を超えません。
主メッセージ、その直接の返信コンテキスト、そこから到達する関係情報は
一まとまりで、本文や行の途中では切りません。固定部分または最初の一まとまり
さえ予算に入らない場合は `output_budget_too_small` で失敗し、stdoutには
何も書きません。最終行の末尾改行も予算に含みます。予算付き出力の
`relation-resolution targets=` は実際に後続出力したrelation target数、
`fetch-attempts` は打ち切り前に実行済みの総取得回数です。
`--max-bytes` を省略した出力は従来とbyte単位で同一です。

各メッセージ本文だけを短く表示する場合は `--max-body-bytes` を使います。
値は0〜65535のASCII十進数で、生UTF-8のバイト数として本文ごとに適用します。
UTF-8文字の途中では切らず、選択・返信文脈・関係解決を終えたあとに
表示用の本文だけを短くします。切れた行は `body-truncated=true` と
`original-body-bytes=<元のバイト数>` を本文の前に示します。本文語句の照合や
返信関係には元の全文を使い、取得範囲や完了状態は変わりません。
`--max-bytes` と併用すると、本文を短くして安全にエスケープした後で
標準出力全体の予算を測ります。

```sh
cwk messages list --room <room-id> --contains <text> --max-body-bytes 64
```

送信者、日付、期間、順位で絞る場合は、明示的な返信元・返信先を1ホップ
含める `replies` が既定です。返信コンテキストを省いて出力を軽くする場合だけ
`--context none` を指定します。

```sh
cwk messages list --room <room-id> --sender <account-ref>
cwk messages list --room <room-id> --sender <account-ref> --context none
```

ユーザーからメッセージリンクを受け取った場合は、`messages resolve-link` で
参照を取り出し、その値を変更せずに `messages show` に渡せます。両方の
コマンドを `cwk config` で有効にしてください。

```sh
cwk messages resolve-link --url 'https://www.chatwork.com/#!rid42-101'
cwk messages show --room 42 --message 101
```

抽出はローカルで行い、通信や認証を行いません。参照先の存在やアクセスは
未確認です。リンク自体は `messages show` の参照引数には渡せません。

指定したメッセージの前後も確認するときは、`--around` に0〜100件を指定します。
返信コンテキストも必要なら同じ最新範囲を一度だけ読み、隣り合うだけでは
返信関係と判断しません。アンカーが範囲外、または閲覧制限で位置不明なら、
周辺メッセージを推測せず状態を表示します。

```sh
cwk messages show --room <room-id> --message <message-ref> --around 2
cwk messages show --room <room-id> --message <message-ref> --context replies --around 2
```

送信時刻を日本時間でも確認するには、`messages list` または `messages show` に
`--jst` を付けます。従来の Unix 秒を残したまま、各メッセージに固定の
`+09:00` の日時を併記します。表現できる年の範囲外は `unavailable` です。

メッセージに結び付いたファイルやルームタスクは、本文の `[info]` や
`[download]` などの記法を解析せず、各リソースが持つ完全一致のメッセージ参照から
探せます。`--kind` は `files`、`tasks`、`all` のいずれかです。`all` はファイル、
ルームタスクの順に各一覧を一度ずつ読み、それぞれ最大100件の取得範囲から一致を
返します。`complete=false` なので、一致0件でもリンクが存在しないとは限りません。
表示されたファイル・タスク参照は、同じルーム参照と組み合わせて変更せず詳細表示へ
渡してください。

```sh
cwk messages resources --room <room-id> --message <message-ref> --kind all
cwk files show --room <room-id> --file <file-ref>
cwk room-tasks show --room <room-id> --task <task-ref>
```

`messages list --help` には、そのコマンドに関係する短い例を最大2件表示します。

長文やChatwork記法を含む本文は、シェル展開を避けてファイルまたは標準入力から
送信・更新できます。`--body` と `--body-file` はどちらか一方だけを指定します。

```sh
cwk messages send --room <room-id> --body-file ./message.txt
cwk messages update --room <room-id> --message <message-ref> --body-file ./message.txt
cwk messages send --room <room-id> --body - < ./message.txt
```

本文は空でないUTF-8の1〜65535バイトです。改行、Chatwork記法、`$` や `>`
などの文字は本文としてそのまま扱われます。ファイルはインフラストラクチャの
制御された通常ファイル読み取り境界で65536バイトまで読み、本文の妥当性確認に
失敗した場合はChatworkへ送信しません。標準入力のpipe読み取りは呼び出し元の
キャンセルに追従します。

送信結果が不明なエラーになった場合は、送信を再実行せず、
送信時と同じルーム・本文入力・順序の `--to`・任意の `--reply-to` を指定して
直近の完全一致候補を確認します。

```sh
cwk messages reconcile-send --room <room-id> --body-file ./message.txt
```

`--body` を使った場合は同じ本文を指定し、宛先や返信先も送信時と同じ値と順序で
指定します。このコマンドは現在の認証アカウントを一度確認し、直近最大100件の
うち、そのアカウントが投稿した本文完全一致候補を返します。取得件数は絞り込み前
の範囲です。以前の全投稿者を含む候補からの変更です。

結果の `outcome` は常に `unknown` です。同文の過去投稿、送信時と現在の
アカウントの違い、反映遅れ、取得範囲や閲覧制限、編集・削除、変更された本文や
返信先投稿者により、候補の有無だけでは送信成否を証明できません。標準入力で
送信した場合は同じ本文をもう一度渡してください。本文とファイルパスは出力されません。

## メッセージ取得の範囲

`messages list` が読むのは、Chatworkから取得できる最新または差分の最大100件です。日付、送信者、順位による絞り込みは、その範囲に対してcwk内部で行います。出力は小さくなりますが、100件より古い履歴まで取得できるわけではありません。

1つのメッセージに返信先が複数ある場合も、`reply=[#1,#2]` のようにすべて表示します。取得範囲外に返信先がある場合は、同じルームの明示的な参照を既定で最大5件まで補います。取得できない情報や到達できない期間は、空だったかのように扱わず結果に明示します。詳しい境界は `cwk messages list --help` で確認できます。

引用に記録された相手と時刻が、同じルームの取得範囲内にあるメッセージの送信者・時刻と完全一致すると、そのメッセージの `message-ref` を引用候補として表示します。これは観測した範囲内の候補にすぎず、引用を解決したり返信関係を作ったりしません。`unresolved-replies`、`unresolved-quotes`、`unknown-relation-sets` は別々に数え、候補が1件でも引用は未解決のままです。部分的な取得で候補が見つからない場合は不明として扱い、取得範囲や閲覧制限の表示を確認してください。

## Chatwork API による制約

2026年8月10日時点の公式 Chatwork API v2 には、グローバルな
メッセージ全文検索とリアクション取得の操作が公開されていません。
そのため `cwk messages search` とリアクション集計コマンドはありません。
ルーム名からは `rooms list --query` で入り、`messages list --room`
の `--sender` / `--on` / `--since` / `--until` を使って、公式 API が返す
最大100件のルーム内範囲を絞り込んでください。根拠は
[API エンドポイント一覧](https://developer.chatwork.com/ja/docs/endpoints) と
[OAuth スコープ一覧](https://developer.chatwork.com/docs/oauth) です。

## この `cwk` の実行範囲を選ぶ

`cwk config` では、この `cwk` プロセスが実行できるChatworkコマンドをいつでも選び直せます。設定前に実行できるのは、`help`、`doctor`、`version`、`config` だけです。保存した設定で無効になっているコマンドは `command_disabled` になり、名前空間ヘルプには名前だけが表示されます。使う必要がある場合は、ユーザーに `cwk config` で有効にするよう依頼してください。

```sh
cwk config
```

`q` なら保存せず終了します。有効にしたコマンドだけが実行と詳しいヘルプの対象になり、作業に関係のないコンテキストや誤操作を減らせます。

これは現在の `cwk` のディスパッチ範囲です。同じOSユーザーは設定ファイルや `XDG_CONFIG_HOME` を変更したり、端末入力を自動化したり、別の実行ファイルやPATでChatworkへ直接アクセスできます。人の識別やOSアカウントの保護ではなく、PATの検証、Chatwork側の権限、対象参照、変更確認とは別の制御です。

## 設定ファイル

macOSとLinuxでは、コマンド表示の設定を次の場所に保存します。

```text
${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json
```

`~/.config` や `XDG_CONFIG_HOME` がシンボリックリンクでも、実体が通常のディレクトリなら利用できます。一方、cwkが管理する `cwk` ディレクトリと設定ファイルはシンボリックリンクにできません。Unixではディレクトリに `700`、ファイルに `600` の権限が必要です。

`command_selection_unsafe` は、cwk管理のディレクトリや設定ファイルのシンボリックリンク・特殊なファイル、Unix権限がディレクトリ `700`・ファイル `600` と異なる場合、壊れた `XDG_CONFIG_HOME` や `~/.config` の別名などで返ることがあります。競合など別の状態も同じエラーになるため、原因を一つに断定できません。保存先を確認し、データを削除・移動せずに外部で安全な通常ファイル・ディレクトリへ修復してから、`cwk doctor` で診断してください。

## 設計原則

**ノイズを減らし、文脈は残す。**

`cwk` は出力を短くするだけのツールではありません。判断に必要な文脈、取得範囲、不確かな関係、次の操作に使う参照は残します。外部の文章を信頼できないデータとして扱い、名前や本文から対象や関係を作りません。

考え方のヒントの一つは、AIエージェントに渡るコマンド出力からノイズを減らす [RTK（Rust Token Killer）](https://www.rtk-ai.app/) です。`cwk` はRTKの派生製品ではなく、その考え方をChatworkの作業へ応用しています。

詳しい製品判断は [プロジェクトのテーゼ](docs/00_theses.md) と [製品契約](docs/01_product_contract.md) にまとめています。

## 開発

必要なGoのバージョンは `go.mod` を参照してください。

```sh
go run ./cmd/cwk --help
go run ./cmd/cwk doctor
task check
```

変更前に [AGENTS.md](AGENTS.md) と [文書マップ](docs/README.md) を確認してください。コントリビューションについては [CONTRIBUTING.md](CONTRIBUTING.md)、脆弱性の報告は [SECURITY.md](SECURITY.md) を参照してください。

## ライセンス

`cwk` は [MIT License](LICENSE) の下で提供されています。
