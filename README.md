# cwk

> エージェントのためのChatwork CLI。

`cwk` は、Chatworkでの作業をエージェントが迷わず進めるためのCLIです。長いAPIレスポンスをそのまま返すのではなく、必要な情報、関係、次の操作に使える参照をコンパクトにまとめます。

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

ChatworkのAPIトークン（PAT）を環境変数 `CWK_API_TOKEN` に設定します。トークンはシークレットマネージャーなど安全な場所で管理してください。`cwk` 自体が保存することはありません。

```sh
cwk rooms list
```

一覧で見つけた `room-ref` を使って、会話を読みます。

```sh
cwk messages list --room <room-ref>
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

通常のヘルプは人向けです。エージェントが入力、出力、エラー、復旧手順まで確認するときは、正確なコマンドを指定してagent形式を使います。

```sh
cwk help --format agent
cwk help messages list --format agent
```

## よく使う例

今日や昨日の会話だけを読みます。

```sh
cwk messages list --room <room-ref> --on today
cwk messages list --room <room-ref> --on yesterday
```

表示名から人物を探し、その人の投稿だけを読みます。

```sh
cwk members find --room <room-ref> --query <name>
cwk messages list --room <room-ref> --sender <account-ref>
```

`members find` は一致した候補をすべて返します。候補が複数あっても、cwkが人物を勝手に選ぶことはありません。

件数や期間も絞れます。

```sh
cwk messages list --room <room-ref> --count 10
cwk messages list --room <room-ref> --start-index 11 --count 20
cwk messages list --room <room-ref> \
  --since 2026-07-17T12:00:00+09:00 \
  --until 2026-07-17T18:00:00+09:00
```

返信の前後関係も含められます。

```sh
cwk messages list --room <room-ref> --sender <account-ref> --context replies
```

`messages list --help` には、そのコマンドに関係する短いレシピも表示されます。agent形式では、レシピの代わりに再利用できる参照の流れが構造化されます。

## メッセージ取得の範囲

`messages list` が読むのは、Chatworkから取得できる最新または差分の最大100件です。日付、送信者、順位による絞り込みは、その範囲に対してcwk内部で行います。出力は小さくなりますが、100件より古い履歴まで取得できるわけではありません。

1つのメッセージに返信先が複数ある場合も、`reply=[#1,#2]` のようにすべて表示します。取得範囲外に返信先がある場合は、同じルームの明示的な参照を既定で最大5件まで補います。取得できない情報や到達できない期間は、空だったかのように扱わず結果に明示します。詳しい境界は `cwk messages list --help` で確認できます。

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
