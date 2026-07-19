# Work Context: 日本の利用者を既定とするローカライズ

## Current behavior

- `go run ./cmd/cwk --help` は見出し、説明、案内を英語で表示する。
- `internal/cli/chatwork_catalog.go` などの catalog 自然言語は agent help と human help の双方に英語で投影される。
- `internal/cli/errors.go` は安定した英語ラベルと、各層で作られた英語 message を stderr に出力する。
- `README.md` とコミュニティ文書、GitHub Issue/PR テンプレートは英語である。
- コマンドパス、フラグ、JSON キー、fault kind/code、参照 kind は catalog と contract test が固定する機械契約である。

## Relevant structure

- Entry point: `cmd/cwk/main.go`
- Domain rule: `internal/domain/fault`, `internal/domain/operation`
- Application use case: existing tasks are unchanged
- Infrastructure boundary: provider text and fixtures are unchanged
- CLI catalog or presentation: `internal/cli/help.go`, `errors.go`, `config_tui.go`, catalog specs
- Existing tests and harness checks: CLI golden/snapshot tests, catalog validation, `scripts/check.sh`

## Constraints

- Stable machine identifiers and exact opaque bytes must not be localized.
- Current success projection is a reviewed compatibility contract; its grammar cannot be casually translated.
- External text remains untrusted data and must never be translated or rewritten by presentation.
- Repository documentation is English by default under the old policy; the thesis and `AGENTS.md` policy must be revised before Japanese active documentation is normalized.

## External facts

No external factual dependency is required. This change responds to a product-owner localization decision and uses only repository contracts.

## Unknowns

- [x] Whether help and error text are included: yes, they are direct user-facing CLI presentation.
- [x] Whether machine identifiers should be translated: no, compatibility and exact-reference contracts require preservation.

## Thesis evidence

- Repeated design decision or point of agent confusion: every user-facing surface currently assumes English while the target users are now explicitly Japanese.
- User outcome or friction observed in the minimal slice: root help and errors require English comprehension before a Japanese user can act or recover.
- Code workaround or exception being considered: per-command ad hoc bilingual prose would duplicate metadata and increase token cost.
- Proposed thesis revision: Japanese is the default human language; machine identifiers remain locale-neutral and stable.
- Downstream impact: product, architecture, security, catalog metadata, help/error/TUI presentation, public docs, tests, and harness lint.

## Security and public-boundary notes

- No credential, destination, side effect, or provider contract changes.
- Translation must not rewrite opaque references or untrusted provider text.
- Examples remain synthetic and no external content is fetched.
- Japanese Unicode is allowed only in trusted CLI-authored prose; existing hostile-output projection remains unchanged.

## Glossary

- 人間向け表示: human help、TUI、fault message、next-action reason、公開ガイドの自然言語。
- 機械識別子: command path、flag、environment variable、JSON key、fault kind/code、schema token、reference kind。
- 外部テキスト: Chatwork または fixture から受け取る、信頼されない利用者生成データ。
