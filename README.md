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

最初に、エージェントへ表示するChatworkコマンドを選びます。

```sh
cwk config
```

上下キーで移動し、Spaceで切り替え、Enterで保存します。必要なコマンドだけを表示すると、エージェントが読む候補とトークン消費を減らせます。

<img width="953" height="477" alt="image" src="https://github.com/user-attachments/assets/43d1c5c3-6bf4-4b23-8e50-d04319e39f79" />

ChatworkのAPIトークン（PAT）を環境変数 `CWK_API_TOKEN` に設定します。トークンはシークレットマネージャーなど安全な場所で管理してください。`cwk` 自体が保存することはありません。

認証に失敗したら `cwk doctor` を実行してください。値を表示せずに、`CWK_API_TOKEN` の有無・形式と、設定ファイル／キーチェーンが未対応であることを診断します。現時点の認証探索先はコマンドプロセスの `CWK_API_TOKEN` だけです。

```sh
cwk rooms list
```

一覧で見つけた `room-ref`（先頭ゼロなし・1〜32桁の正の10進数）を、
`room-id` としてそのまま使って会話を読みます。

```sh
cwk messages list --room <room-id>
```

結果は、エージェントが少ないトークンで正確に読める形にまとまります。

```text
messages room-ref=4101 count=2 window=recent source-limit=100 complete=false access-limitation=none unresolved-relations=0 unknown-relation-sets=0 oldest-reachable-message-ref=9001 oldest-reachable-send-time=1700000000
relation-resolution fetch-limit=5 fetch-attempts=0 targets=0
external-text=untrusted escaped
schema: #sequence message-ref actor sent [reply] [to] [quote] [relation-state] "body"
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
cwk messages --help
cwk messages list --help
```

ヘルプ経路はこのtext形式だけです。名前空間のヘルプには配下の各
コマンドのusageと必須フラグが出るため、その場でコマンドを書き始められます。
正確なコマンドのヘルプは2KB以内で、usage、フラグ、併用制約、例を最大2件
だけ表示します。失敗時の復旧は、エラーに出る `kind`、`code`、
`next_action` を使います。廃止済みの `--format agent` をhelpへ渡すと、
`--help` を使うよう明示して失敗します。既存の JSON 出力は
`cwk doctor --format json` で引き続き利用できます。Chatwork タスクの
成功出力は従来どおり text 形式です。

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

送信者、日付、期間、順位で絞る場合は、明示的な返信元・返信先を1ホップ
含める `replies` が既定です。返信コンテキストを省いて出力を軽くする場合だけ
`--context none` を指定します。

```sh
cwk messages list --room <room-id> --sender <account-ref>
cwk messages list --room <room-id> --sender <account-ref> --context none
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

## メッセージ取得の範囲

`messages list` が読むのは、Chatworkから取得できる最新または差分の最大100件です。日付、送信者、順位による絞り込みは、その範囲に対してcwk内部で行います。出力は小さくなりますが、100件より古い履歴まで取得できるわけではありません。

1つのメッセージに返信先が複数ある場合も、`reply=[#1,#2]` のようにすべて表示します。取得範囲外に返信先がある場合は、同じルームの明示的な参照を既定で最大5件まで補います。取得できない情報や到達できない期間は、空だったかのように扱わず結果に明示します。詳しい境界は `cwk messages list --help` で確認できます。

## Chatwork API による制約

2026年8月10日時点の公式 Chatwork API v2 には、グローバルな
メッセージ全文検索とリアクション取得の操作が公開されていません。
そのため `cwk messages search` とリアクション集計コマンドはありません。
ルーム名からは `rooms list --query` で入り、`messages list --room`
の `--sender` / `--on` / `--since` / `--until` を使って、公式 API が返す
最大100件のルーム内範囲を絞り込んでください。根拠は
[API エンドポイント一覧](https://developer.chatwork.com/ja/docs/endpoints) と
[OAuth スコープ一覧](https://developer.chatwork.com/docs/oauth) です。

## 表示するコマンドを選ぶ

`cwk config` では、エージェントに見せるChatworkコマンドをいつでも選び直せます。設定前は、`help`、`doctor`、`version`、`config` だけが表示されます。

```sh
cwk config
```

`q` なら保存せず終了します。オンにしたコマンドだけがヘルプや候補に現れるため、作業に関係のないコンテキストを減らせます。

これは権限管理ではありません。ローカルユーザーはいつでも設定を変更できます。Chatworkの権限やsandboxの代わりにはなりません。

## 設定ファイル

macOSとLinuxでは、コマンド表示の設定を次の場所に保存します。

```text
${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json
```

`~/.config` や `XDG_CONFIG_HOME` がシンボリックリンクでも、実体が通常のディレクトリなら利用できます。一方、cwkが管理する `cwk` ディレクトリと設定ファイルはシンボリックリンクにできません。Unixではディレクトリに `700`、ファイルに `600` の権限が必要です。

`command_selection_unsafe` が表示されたら、保存先を確認してから `cwk doctor` を実行してください。

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
