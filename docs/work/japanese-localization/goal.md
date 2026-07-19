# Work Goal: 日本の利用者を既定とするローカライズ

- Status: Complete
- Owner: Maintainer
- Target: Current change
- Related ADRs: None

## Outcome

日本の開発者・運用担当者が、英語の説明文を解釈しなくても `cwk` の導入、コマンド探索、入力、診断、失敗からの復旧、貢献および問い合わせを行える。既定の人間向け表示と公開文書は日本語とし、自動化で使う安定した機械識別子は従来どおり変更せず利用できる。

## Why now

対象ユーザーを日本とする明示的な要求に対し、現在は README、human help、TUI、エラーメッセージ、GitHub テンプレートが英語で一貫している。Chatwork の主要利用地域とも一致せず、人間が監督する際の認知負荷と誤操作リスクになる。

## Non-goals

- コマンドパス、フラグ名、環境変数、エラーコード、JSON キー、schema version、opaque reference kind の翻訳
- Chatwork から返る外部テキスト、外部 API fixture、過去の `docs/work/` 証跡の翻訳
- 実行時の多言語切替や OS locale 自動判定
- 選択済み成功出力 grammar の意味・順序・参照値の変更

## Acceptance criteria

- [x] root・namespace・exact human help と `config` TUI が日本語で表示される。
- [x] 公開エラーの message と next-action reason が日本語で、kind/code/exit status は不変である。
- [x] agent help の安定キーと識別子は不変で、自然言語の説明は日本語である。
- [x] README、サポート、セキュリティ、貢献、行動規範、Issue/PR テンプレートが日本語である。
- [x] 日本語表示方針と非翻訳対象が thesis・製品・architecture・security・harness に反映される。
- [x] 対象ファイルの英語回帰を検出する機械的チェックが `task check` から実行される。
- [x] `task check`（security・release・public を含む）が成功する。

## Governing documents

- Thesis: `docs/00_theses.md` Axiom 2, Axiom 4
- Product contract section: public runnable surface, compatibility boundary
- Architecture or security invariant: CLI presentation ownership, output and terminal safety
- Existing ADR: None

## Completion definition

日本語の利用者接点と安定した機械契約の境界が文書・コード・テスト・lint で一致し、必要な gate が成功した時点で完了する。
