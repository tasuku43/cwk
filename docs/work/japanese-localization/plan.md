# Work Plan: 日本の利用者を既定とするローカライズ

- Status: Complete
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

日本語を単一の既定 human locale として source of truth に直接反映する。実行時 locale 分岐は追加せず、catalog の自然言語、human help、TUI、公開 fault message/recovery reason、利用者向け文書を日本語化する。安定した機械識別子、成功出力の positional schema、外部テキスト、歴史証跡は変更しない。

## Alternatives considered

### Runtime i18n catalog

将来の多言語対応には有効だが、現時点では言語選択、fallback、全 message key、互換性の新しい公開契約を必要とし、単一の日本向け製品には過大である。

### 英日併記

help と agent contract のサイズ・認知負荷・token cost を恒常的に増やす。機械識別子を英語のまま残すため、自然言語まで二重化する必要はない。

## Design

### Public contract

- Human locale: Japanese (`ja-JP` semantics, no runtime selector).
- Stable identifiers: unchanged English ASCII tokens.
- Faults: message/reason Japanese; kind/code/retryability/exit mapping unchanged.
- Success output: selected grammar and JSON/TSV field names unchanged.
- Agent contract: JSON keys and stable values unchanged; descriptive prose Japanese.

### Layer changes

- Domain/Application/Infrastructure: public fault prose only where needed; no behavior or boundary changes.
- CLI and catalog: primary localization owner.
- Harness: scan selected active user-facing files for forbidden legacy English UI markers.

### Error and cancellation behavior

Classification, retryability, next command, and exit status remain unchanged. Only human-readable message and reason change. Dynamic values remain structurally validated and are embedded in Japanese grammar without transforming identifiers.

### Security and public boundary

Opaque references, PAT values, provider text, URLs, and external fixtures are never translated. No new dependency, network destination, storage, or side effect is introduced.

## Implementation slices

1. Durable locale contract and failing localization tests.
2. Human help/TUI/error presentation.
3. Catalog/agent descriptive prose.
4. Public docs and repository templates.
5. Harness enforcement and full gates.

## Verification

- Focused CLI and catalog tests.
- Text and JSON fault snapshots with unchanged stable fields.
- Exact opaque-reference and hostile-output regressions.
- Runtime examples for root/namespace/exact help and invalid input.
- `task check`, `task security`, `task public:check`.

## Rollout and rollback

This is an intentional pre-1.0 text compatibility change. Automation continues to use unchanged codes, keys, command paths, and references. Rollback is a source revert; no stored data migration is required.

## Documentation promotion

Promote the language and compatibility boundary to theses, product contract, architecture, security model, harness, README, and `AGENTS.md`.
