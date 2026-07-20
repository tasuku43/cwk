# cwk

> エージェントのためのChatwork CLI。

`cwk` は、エージェントがChatworkを迷わず読み、安全に操作するためのCLIです。
APIのレスポンスをそのまま流すのではなく、依頼を進めるために必要な情報だけを構造化して返します。足りない情報は、結果に含まれる参照を使って追加取得できます。

そのため、エージェントは長いJSONを読み解いたり、名前や文面から対象を推測したりする必要がありません。ルームを探す、会話を読む、メッセージを送る、といった作業をそのままコマンドにできます。

## インストール

```sh
brew install tasuku43/tap/cwk
```

[Homebrew 6.0以降のtap trust](https://docs.brew.sh/Tap-Trust)では、公式ではないtapのFormulaを読み込む前に信頼の指定が必要です。上の完全修飾名によるインストールは、tap全体ではなく`cwk`だけを信頼します。

すでに`tasuku43/tap`を追加済みで、短縮名の`cwk`を使いたい場合は、Formula単位で信頼してからインストールします。

```sh
brew trust --formula tasuku43/tap/cwk
brew install cwk
```

リリース済みのアーカイブは [GitHub Releases](https://github.com/tasuku43/cwk/releases) から取得できます。

### dotfilesで設定ディレクトリを管理している場合

`cwk` はmacOSとLinuxで `${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json` を使います。`~/.config` や `XDG_CONFIG_HOME` 自体がシンボリックリンクでも、実体が通常のディレクトリなら利用できます。

一方、`cwk` が書き込む `cwk` ディレクトリと `command-selection.json` 自体はシンボリックリンクにできません。Unixではディレクトリに `700`、ファイルに `600` の権限も必要です。インストール直後に `command_selection_unsafe` が表示された場合は、保存先を確認してください。

```sh
config_home="${XDG_CONFIG_HOME:-$HOME/.config}"
ls -ld "$config_home" "$config_home/cwk" \
  "$config_home/cwk/command-selection.json" 2>/dev/null
cwk doctor
```

通常のディレクトリとファイルなら、次のように権限を修復できます。シンボリックリンクや特殊ファイルだった場合は、必要な内容を退避してから通常のディレクトリまたはファイルとして作り直してください。

```sh
config_home="${XDG_CONFIG_HOME:-$HOME/.config}"
test ! -d "$config_home/cwk" || test -L "$config_home/cwk" || \
  chmod 700 "$config_home/cwk"
test ! -f "$config_home/cwk/command-selection.json" || \
  test -L "$config_home/cwk/command-selection.json" || \
  chmod 600 "$config_home/cwk/command-selection.json"
```

## 使い始める

`cwk` は、環境変数 `CWK_API_TOKEN` からChatworkのAPIトークン（PAT）を読み取ります。

普段使う場合は、`.zshrc` などのシェル起動設定から `CWK_API_TOKEN` を読み込めるようにしておくのがおすすめです。PATはシークレットマネージャーなど、安全な場所に保存してください。

環境変数を設定したシェルでは、そのまま実行できます。

```sh
cwk rooms list
```

シェル起動設定を使う場合は、ファイルの権限と平文保存のリスクを確認してください。`cwk` がトークンを保存することはありません。現在の認証方式はPATのみで、OAuthには対応していません。

利用できる作業はhelpから探せます。普段のコマンド選択と入力確認には、人向けの名前空間または正確なコマンドのhelpを使います。目的に合うコマンドがまだ分からない場合だけ、エージェント向けの短い全コマンド索引から探せます。結果・エラー・復旧・参照ワークフローまで必要な場合は、正確な1コマンドに絞って機械可読契約を確認します。

```sh
cwk --help
cwk messages --help
cwk messages list --help
cwk help --format agent
cwk help messages list --format agent
```

`cwk help messages --format agent` のような名前空間指定も短い索引だけを返します。完全な機械可読契約を複数コマンド分まとめて返すことはありません。

## コマンド表示を絞る

`cwk config` は、エージェントに表示するChatworkコマンドを対話型ターミナルで選ぶための画面です。上下キーで移動し、Spaceで切り替え、Enterで保存します。`q` なら保存せず終了します。

```text
コマンド選択
常に有効: doctor, help, version, config
  [x] [read]   account show - 認証済みのChatworkアカウントを表示する
  [x] [read]   contacts list - Chatworkコンタクトを検索する
> [x] [read]   rooms list - 参加中のChatworkルームを検索する
  [ ] [create] rooms create - メンバーを正確に指定してグループチャットを作成する
  [x] [read]   messages list - 選択可能な上限付きメッセージ範囲を取得する
  [ ] [create] messages send - 完全一致するルームへメッセージを送信する
↑/↓ 移動  Space 切替  Enter 保存  q 終了
```

保存後は、結果を短い日本語で表示します。

```text
コマンド表示を保存しました。
12件を表示し、21件を非表示にしました（22件変更）。
```

現在は存在しない設定や旧形式の設定を整理した場合だけ、その件数をもう1行表示します。保存結果が不確かなエラーになったときは、再保存する前に `cwk doctor` で照合してください。

オンにしたコマンドだけがhelpや候補に現れ、呼び出せるようになります。オフにしたコマンドはエージェントにとって存在しないコマンドになり、パスを直接指定しても実行できません。これにより、作業に関係のないコンテキストを最初から渡さずに済みます。

ただし、利用禁止を強制する仕組みではありません。エージェントを含むローカルユーザーは、`cwk config` をもう一度実行すればコマンドを有効にできます。Chatworkの権限、認可、sandboxの代わりにはなりません。

## 出力はこうなる

たとえば、メッセージ一覧は次のように表示されます。

```text
messages room-ref=4101 count=2 window=recent source-limit=100 complete=false access-limitation=none unresolved-relations=0 unknown-relation-sets=0
external-text=untrusted escaped
schema: #sequence message-ref actor sent [reply] [to] [quote] [relation-state] "body"
actors
  a1 account-ref=7001 name="Aki"
  a2 account-ref=7002 name="Beni"
#1 9001 a1 1700000000 "Release time?"
#2 9002 a2 1700000010 reply=#1 to=a1 "15:00 works."
```

これは人間向けのチャット画面を再現したものではありません。エージェントが少ないトークンで内容と関係を正確に理解できるよう、送信者の重複をまとめ、返信関係と再利用できる参照を明示しています。

まず必要な範囲だけを表示し、必要なら送信者や1始まりの順位で絞り、直接の返信関係を追加できます。`--count` は終了順位ではなく、`--start-index` からの最大件数です。

```sh
cwk messages list --room 4101 --count 10
cwk messages list --room 4101 --start-index 11 --count 20
cwk messages list --room 4101 --sender 7001 --context replies
```

2行目は新しい順の順位11〜30を選びます。どちらもChatworkからは上限100件の範囲を1回取得してローカルで選ぶため、別々の実行の間にメッセージが増減すると順位は移動し得ます。

出力の `room-ref`、`message-ref`、`account-ref` は、次のコマンドへそのまま渡せます。表示名から対象を探し直す必要はありません。

## Chatworkを調査する

`messages list` が扱うのは、1ルームから取得した最新または差分の上限付き範囲です。最大100件の取得元上限があり、完全なルーム履歴ではありません。調査目的に応じて次の形を使います。

- 取得できた最新範囲のテーマを横断する: `cwk messages list --room <room-ref>`。ローカル選択を付けず、返された範囲を一度に読みます。
- 直近の小さい範囲から確認する: `cwk messages list --room <room-ref> --count <count>`。件数は調査に必要な文脈量に合わせ、続きは出力の `next-start-index` を次の `--start-index` に使います。同じ送信者条件を保ち、別実行の間にメッセージが増減すると順位が動き得ることに注意します。
- 特定人物が送ったメッセージを確認する: `cwk messages list --room <room-ref> --sender <account-ref>`。対象はその人物が送信した、取得範囲内のメッセージです。全履歴や、その人物宛てのメッセージを意味しません。
- 絞り込んだ結果の直接の返信経緯を補う: `--sender`、`--start-index`、または `--count` とともに `--context replies` を指定します。取得範囲内の明示的な返信元・返信先を1ホップだけ追加し、スレッド全体や窓外のメッセージは取得しません。絞り込まない一覧にも、判明している返信辺は表示されます。
- プロバイダーの差分範囲が必要と分かっている再調査: `--window changes` を明示します。通常の状況把握は、既定の最新範囲 `recent` を使います。
- 別ルームまたは現在の窓外へ関係が続く: 出力にある正規のルーム参照とメッセージ参照を使い、対象ルームで別途 `messages list` または `messages show` を実行します。1ルームの上限付き窓だけで窓外の関係を解決したとは扱いません。
- 正規のメッセージ参照が分かっている1件を深掘りする: `cwk messages show --room <room-ref> --message <message-ref>`。範囲探索のための `list` を繰り返しません。

## 設計原則

**ノイズを減らし、文脈は残す。**

`cwk` が減らすのは、API由来の重複や作業に不要な情報です。判断に必要な文脈、取得範囲、不確かな関係、次の操作に使う参照は削りません。

- 必要な情報だけを、後処理しやすい形で返す
- 足りない情報は、参照を変えずに追加取得できる
- 読み取りと書き込みを区別し、操作の意図・対象・影響を先に明示する
- 外部の文章を信頼できないデータとして扱い、推測で関係を作らない

書き込みコマンドは「何を、どう変えるか」を実行前に確定し、曖昧な名前から対象を選び直しません。結果が不確かな場合も成功扱いにせず、読み取り専用の確認手順を返します。

設計のヒントの一つは、AIエージェントに渡るコマンド出力からノイズを減らす [RTK（Rust Token Killer）](https://www.rtk-ai.app/) です。`cwk` はRTKの派生製品ではなく、その考え方をChatworkの作業に応用しています。優先するのは出力の短さそのものではなく、エージェントが正しく理解して安全に作業を完了できることです。

詳しい製品判断は [プロジェクトのテーゼ](docs/00_theses.md) と [製品契約](docs/01_product_contract.md) を参照してください。

## 開発

必要なGoのバージョンは `go.mod` を参照してください。リポジトリから直接実行できます。

```sh
go run ./cmd/cwk --help
go run ./cmd/cwk doctor
```

```text
cmd/cwk/          実行エントリーポイント
internal/domain/  型と不変条件
internal/app/     タスクごとのユースケース
internal/infra/   Chatworkなど外部システムとの接続
internal/cli/     コマンド、出力、依存関係の組み立て
```

変更前に [AGENTS.md](AGENTS.md) と [文書マップ](docs/README.md) を参照してください。検証は次の1コマンドで行います。

```sh
task check
```

コントリビューションについては [CONTRIBUTING.md](CONTRIBUTING.md)、脆弱性の報告は [SECURITY.md](SECURITY.md) を参照してください。

## ライセンス

`cwk` は [MIT License](LICENSE) の下で提供されています。
