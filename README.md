# Chatwork CLI

Chatwork CLI（`cwk`）は、開発者、運用担当者、コーディングエージェントがChatworkの作業を正確なコマンドとして実行するための、タスク指向CLIです。APIエンドポイントをそのまま公開するのではなく、「ルームを探す」「会話を読む」「メッセージを送る」といったユーザー成果を公開語彙にしています。

- Go module: `github.com/tasuku43/cwk`
- Binary: `cwk`
- Display name: `Chatwork CLI`
- 認証: `CWK_API_TOKEN` によるプロセス単位のPAT認証

2026-07-18時点で固定したChatwork APIの公開REST操作32件を、レビュー済みのタスクワークフローでカバーします。コマンド、入出力、効果、失敗、復旧、canonical referenceの契約は1つの `cli.Catalog` から導出され、`task check` で機械的に検証されます。

## 設計上の特徴

- 公開コマンドは、プロバイダーのAPI操作ではなくユーザーのタスクを表します。
- コマンドを `utility`、`discover`、`act` に分類し、発見されたopaque IDを変更せず次の操作へ渡します。
- ドメイン、アプリケーション、インフラストラクチャ、CLIの4レイヤーで責務を分離します。
- すべての操作は `read`、`create`、`write` の効果と、型付けされた意図、対象、影響を宣言します。
- エージェント向けhelpは、前提条件、入力、出力、完全性、失敗、復旧アクションを機械可読な形で返します。
- Chatwork APIトークンをドメインやアプリケーション層へ渡さず、インフラストラクチャ内の一時的なbindingに閉じ込めます。
- ページング、タイムアウト、試行回数、応答サイズ、ミューテーション結果を有限の契約として扱います。
- 外部テキストを信頼できないデータとして構造的に分離し、canonical referenceは検証済みの完全一致値として保持します。
- アーキテクチャ、セキュリティ、公開可能性、リリースに関する主張を実行可能なチェックで検証します。

製品判断の詳細は [プロジェクトのテーゼ](docs/00_theses.md) と [製品契約](docs/01_product_contract.md)、認証と外部呼び出しの境界は [認証](docs/07_authentication.md) と [外部API契約](docs/08_external_api_contracts.md) を参照してください。

## 実行する

必要なGoのバージョンは `go.mod` を参照してください。リポジトリから直接、次のように実行できます。

```sh
go run ./cmd/cwk --help
go run ./cmd/cwk rooms --help
go run ./cmd/cwk rooms list --help
go run ./cmd/cwk help --format agent
go run ./cmd/cwk help rooms --format agent
go run ./cmd/cwk help messages list --format agent
go run ./cmd/cwk config
go run ./cmd/cwk doctor
```

人間向けhelpは階層化されています。ルートhelpはローカルコマンドとタスクのトップレベル名前空間を1回ずつ表示し、`<namespace> --help` はその名前空間のコマンド、`<exact-command> --help` はそのコマンドの使用法と入力契約を表示します。エージェント向けhelpは、ルートでは簡潔な成果索引を返し、スコープ指定時に完全な契約を返します。すべての表示とルーティングは、完全なカタログから導出した同じ有効コマンドビューを使用します。

## 認証する

`CWK_API_TOKEN` を、シェルまたはシークレットマネージャーからコマンドプロセスへ渡してください。トークンをargv、コマンド文字列、プロジェクトファイルへ含めないでください。`CWK_API_TOKEN` はChatwork認証情報の唯一の入力であり、認証方式の選択やloginコマンドはありません。

たとえば、入力を表示しないシェルプロンプトから値を受け取り、子プロセスへ渡す環境変数だけをexportできます。

```sh
read -r -s CWK_API_TOKEN
export CWK_API_TOKEN
cwk rooms list
unset CWK_API_TOKEN
```

トークンはプロセス内にのみ保持されます。`cwk` はXDG/AppData設定、OSの認証情報ストア、プロジェクトファイルへトークンを保存しません。不要になったら環境変数をunsetしてください。未設定または形式が不正な場合は、Chatworkへのリクエスト前に失敗し、スコープ指定のhelpが必要な環境変数として `CWK_API_TOKEN` を示します。

## コマンド表示を絞り込む

エージェントがすべてのワークフローを必要としない場合は、ローカルのコマンドセレクターを利用できます。

```sh
cwk config
```

`config` は対話可能なstdinとstdout端末を必要とします。Up/Downで移動し、Spaceで選択中のChatworkタスクを切り替え、Enterで検証して保存します。`q` は以前保存した設定を変更せず終了します。ASCII Spaceと、日本語入力システムなどが送るU+3000全角スペースは同じ操作です。

各行には `read`、`create`、`write` の文字による効果バッジが残ります。cyan、yellow、magentaは補助的な表示にすぎず、色だけで意味を示しません。画面に現在の正確なコマンドパスと効果を同時に表示できない場合は、端末を広げるまで移動、切り替え、保存を無効にします。この状態でも `q` なら保存せず終了できます。非対話環境では別の入力文法へ切り替えず、型付けされたエラーで失敗します。

Enterより前に `q`、入力終了、Ctrl-C、context cancellationが発生した場合、保存は行われません。Enterはreferenceとrecoveryの閉包性を検証し、端末状態を復元してから、設定を1回だけ置換します。端末復元に失敗した場合も保存しません。

置換結果が不確実な場合は、別の変更を実行する前に、常時利用可能な読み取り専用の `doctor` を実行してください。`command-selection` 診断は、source、state、有効・無効件数、カタログ順の選択から得た決定的な `sha256:` fingerprintを報告します。`help` はコマンド発見用であり、保存状態の照合には使いません。

`help`、`doctor`、`version`、`config` は常に有効で、Chatworkタスクだけを選択できます。有効なプロファイルを読み込めない場合も、これらのローカルコマンドとそのスコープ指定helpは利用できます。bare root helpは、壊れたプロファイルを正常な空ビューに見せず、型付けされた読み込みエラーを返します。シリアライズ形式だけが壊れている場合は明示的な `config` 修復フローを利用できますが、安全でない、またはアクセス不能なストレージはファイルシステム上で修復し、`doctor` で確認する必要があります。

保存後、無効にしたコマンドは、人間・エージェント向けhelp、recovery、reference workflow、ルーティングから消えます。呼び出すと、PATの解決やChatworkリクエストより前に、未知のパスと同じ `unknown_command` を返します。保存済みの正確なパスのallowlistが優先されるため、後のリリースで追加されたコマンドは、明示的に選択するまで無効です。設定ファイルがない場合は、後方互換性のため現在の設定可能コマンドをすべて有効にします。

設定ファイルは、macOS/Linuxでは `${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json`、Windowsでは `%AppData%\\cwk\\command-selection.json` です。PATやChatworkデータは含まず、廃止されたOAuth用 `config.json` とは別です。

この機能はエージェントの注意範囲を減らすためのものであり、認可、sandbox、セキュリティ制御ではありません。ローカルユーザーは `config`、ファイル編集、ファイル削除によりコマンドを再度有効にできます。コマンドを有効にしても、PAT認証、Chatwork上の権限、canonical reference、アクセス変更・破壊的操作の確認を迂回できません。

`doctor` はローカルの読み取り専用ユーティリティです。Chatworkの認証情報を読んだり、Chatworkへアクセスしたりせず、実行環境とコマンド選択の照合情報を報告します。

## canonical referenceを使う

Chatworkタスクはdiscoverからactへの公開ワークフローを提供します。たとえば、`rooms list` が返したcanonical `room_ref` を変更せず、`messages list --room` へ渡します。表示名、行位置、URL、出力内のエイリアスはコマンドIDではありません。

Chatworkの正常出力はタスク結果から直接始まります。テキスト契約は `cwk` のリリースでバージョン管理され、カタログフィールドとgolden testで検証されます。宣言されたタスク情報、canonical reference、該当する範囲・完全性、外部テキストの明示的な信頼境界だけを出力します。

たとえば、合成データのルーム一覧は次の形です。

```text
rooms count=2 complete=true
external-text=untrusted escaped
schema: room-ref "name" type role unread mentions tasks
4101 "Synthetic Lab" "group" "admin" 0 1 0
4102 "Synthetic Archive" "group" "member" 0 0 0
```

`4101` のような値を、宣言された `--room` 入力へそのまま渡します。provider organization ID、icon URL、空のdescription、空のdownload URL、0のcoverage limit、provider coverage kindなど、契約外のフィールドは出力しません。

## メッセージ結果を読む

有限のメッセージウィンドウは、`window=recent|changes`、`complete=false`、正のprovider `source-limit`、未解決の関係件数、型付けされたTo/reply/quote、`untrusted escaped` として扱う本文を宣言します。繰り返される送信者情報は文書ローカルのactor dictionaryにまとめますが、すべてのcanonical message referenceは次のコマンドでそのまま利用できます。

```text
messages room-ref=4101 count=2 window=recent source-limit=100 complete=false unresolved-relations=0
external-text=untrusted escaped
schema: #sequence message-ref actor sent [reply] [to] [quote] "body"
actors
  a1 account-ref=7001 name="Aki"
  a2 account-ref=7002 name="Beni"
#1 9001 a1 1700000000 "Release time?"
#2 9002 a2 1700000010 reply=#1 to=a1 "15:00 works."
```

固定スキーマが位置の意味を示すため、各レコードで `message-ref=`、`sent=`、`body=` を繰り返しません。`reply=#1` は文書内だけの関係です。後のコマンドが `--message` を必要とする場合、`#1` やactor aliasではなく、2番目のフィールド（`9001` または `9002`）を使います。

外部の後処理なしで結果を絞るには、1から100の `--limit` を指定します。型付けされた送信時刻から新しいprimary messageを選び、同じ時刻ではprovider上の後の位置を優先しますが、表示順と `#sequence` は元のprovider順を保ちます。`--sender` は最大100回指定でき、limitより前にOR条件の候補集合を作ります。同じ有限ウィンドウ内にある直接のreply parent/childも必要なときだけ `--context replies` を追加します。

```sh
go run ./cmd/cwk messages list --room 4101 --limit 10
go run ./cmd/cwk messages list --room 4101 --sender 7001
go run ./cmd/cwk messages list --room 4101 \
  --sender 7001 --sender 7002 --limit 10 --context replies
go run ./cmd/cwk messages list --room 4101 --window changes
```

`--window` を省略すると、最新の有限な `recent` ウィンドウを選びます。providerの差分取得を意図する場合だけ、明示的に `--window changes` を指定してください。どちらもルームの完全な履歴ではありません。

選択後も元の `#sequence` を保持するため、番号に欠番が生じることがあります。`--limit` を指定した場合は、limit適用前のcandidate countとrequested primary limitも1回だけ記録します。`--context replies` により追加されるレコードは1-hopのreply contextであるため、headerの `count` が `--limit` を超える場合があります。default contextは `none` です。

providerの `source-limit=100` は別の取得上限です。このコマンドは、文書化された `force` queryだけを使ってChatworkへ1回リクエストし、paginationを行いません。`[To]`/`[rp]` の本文から関係を推測したり、thread全体をたどったり、欠けたparentを取得したり、2人だけの会話だと断定したりしません。

## ファイル結果を読む

ファイル発見も同じ規則に従い、元メッセージがない状態を明示します。

```text
files count=2 limit=100 complete=false
external-text=untrusted escaped
schema: file-ref room-ref account-ref message-ref "name" size
6302 4101 7001 9001 "release.txt" 0
6301 4101 7002 absent "notes.txt" 4096
```

最初のファイルを確認するには、1番目と2番目の位置を変更せず渡します。

```sh
cwk files show --room 4101 --file 6302
```

`absent` は状態を表すリテラルであり、canonical referenceではないため、コマンド入力へ渡してはいけません。

## 出力とエラーの契約

正常データは、有限の結果全体をレンダリングできた後にstdoutへ書き込みます。失敗は安定したテキストまたはスキーマバージョン付きJSONとしてstderrへ出力し、invalid input、authentication、permission、not found、ambiguous、rate limited、unavailable、policy rejection、cancellation、unsupported、contract violation、internal faultを終了ステータスで区別します。

schema-v3のroot agent helpは、簡潔なoutcome/capability indexです。機械可読な `scope_request` が、正確なコマンドまたは名前空間のhelpを示します。完全なI/O、output、error、role、prerequisite、authentication、mutation、reference flow契約は、スコープ指定の応答だけが返します。

## リポジトリ構成

```text
cmd/cwk/                     薄い実行エントリーポイント
internal/domain/             純粋な型、fault、effect、API envelope
internal/app/                task use case、auth/pagination/execution gate
internal/infra/              外部システムの具体的なadapter
internal/cli/                catalog、routing、rendering、composition root

docs/                        長期的な製品・エンジニアリング判断
docs/decisions/              採択済み・置換済みのarchitecture decision
docs/work/                   進行中の変更を扱う有限のwork packet
tools/                       repository-aware lintと補助ツール
scripts/                     canonical checkとrelease helper
.harness/project.json        製品identityと機械可読policy
.agents/skills/              Codex向けの開発・capability workflow
```

文書の読む順序と責務は [文書マップ](docs/README.md) を参照してください。コントリビューターとコーディングエージェントは [AGENTS.md](AGENTS.md) も読む必要があります。

コミュニティ参加とサポートについては、[行動規範](CODE_OF_CONDUCT.md)、[コントリビューションガイド](CONTRIBUTING.md)、[サポートポリシー](SUPPORT.md)、[セキュリティポリシー](SECURITY.md) を参照してください。

主要な設計・運用文書は次のとおりです。

- [プロジェクトのテーゼ](docs/00_theses.md)
- [製品契約](docs/01_product_contract.md)
- [アーキテクチャ](docs/02_architecture.md)
- [セキュリティモデル](docs/03_security_model.md)
- [検証ハーネス](docs/04_harness.md)
- [公開リポジトリ方針](docs/05_public_repository.md)
- [リリース方針](docs/06_release.md)
- [認証](docs/07_authentication.md)
- [外部API契約](docs/08_external_api_contracts.md)
- [エージェント準備検証](docs/09_agent_readiness_validation.md)

## 検証プロファイル

すべてのエントリーポイントは `./scripts/check.sh` に委譲します。

| コマンド | 目的 |
|---|---|
| `task check:fast` | 短いフィードバックループ向けのformat、architecture、focused test |
| `task check` | merge前に必須のfull gate |
| `task security` | credential、dependency、egress、public-boundary check |
| `task release:check` | packagingとrelease-contract check |
| `task public:check` | public readinessと公開境界check |

CIが完了判定の基準です。ローカルhookは高速なプロファイルを実行できますが、ポリシーを再実装せず、同じスクリプトを呼び出す必要があります。

このリポジトリでは、例とフィクスチャに公開可能な合成データを使用します。機密情報をソース、文書、生成ファイル、ビルドログ、Git履歴へ含めないでください。公開リポジトリの方針は [公開リポジトリガイド](docs/05_public_repository.md) を参照してください。

## ライセンス

Chatwork CLI は [MIT License](LICENSE) の下で提供されています。
