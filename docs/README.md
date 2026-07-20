# ドキュメント案内

このディレクトリには Chatwork CLI の継続的な設計判断を収録しています。常に `00` を読み、変更対象に応じて product (`01`)、architecture (`02`)、security (`03`)、harness (`04`)、公開・release (`05`–`06`)、external API (`07`–`09`) を追加で読みます。複数境界をまたぐ場合は番号順に読み、scopeが曖昧またはthesisを変更する場合は `00`–`04` をすべて読んでください。

通常の capability 変更には [`$add-capability`](../.agents/skills/add-capability/SKILL.md) を使います。

| 文書 | 目的 | 主な読者 |
|---|---|---|
| [00_theses.md](00_theses.md) | North Star、thesis の学習サイクル、曖昧な判断を解く原則 | 全員 |
| [01_product_contract.md](01_product_contract.md) | 利用者、対応 outcome、公開語彙、互換性、非目標 | Product owner、contributor、agent |
| [02_architecture.md](02_architecture.md) | 4 layer、catalog、型付き effect/intent、実行 flow | Contributor、agent、reviewer |
| [03_security_model.md](03_security_model.md) | Asset、actor、trust boundary、abuse case、必須 control | Side effect や data handling を変更する全員 |
| [04_harness.md](04_harness.md) | 文書上の claim を local/CI check にする方法 | Contributor、agent、maintainer |
| [05_public_repository.md](05_public_repository.md) | Clean-room derivation、sanitization、license、公開準備 review | Maintainer、release owner |
| [06_release.md](06_release.md) | Versioning、artifact 構築、provenance 判断、release 手順 | Release owner |
| [07_authentication.md](07_authentication.md) | PAT-only の secret-free boundary と将来の認証拡張要件 | Security owner、adapter author、agent |
| [08_external_api_contracts.md](08_external_api_contracts.md) | Pagination、retry/idempotency、schema、capability、API adapter contract | Adapter author、agent、reviewer |
| [09_agent_readiness_validation.md](09_agent_readiness_validation.md) | Scenario による発見・実行・解釈・復旧の検証 | Product owner、agent、reviewer |

追加ディレクトリは、異なる寿命の情報を扱います。

- [ADR template](decisions/0000-template.md) から永続的な architecture decision record を作成します。過去の判断を隠すために ADR を書き換えず、新しい ADR で supersede します。
- [work packet の goal template](work/_template/goal.md) から期限付きの作業 packet を作成します。通常packetは完了時にdurableな結論を昇格して削除し、保持理由と見直し条件を明示した `evidence` だけを現行treeに残します。

root の community 文書は慣例的な固定場所にあります。

| 文書 | 目的 |
|---|---|
| [`README.md`](../README.md) | 利用者向け入口 |
| [`AGENTS.md`](../AGENTS.md) | 人間と agent 共通の canonical contribution policy |
| [`CONTRIBUTING.md`](../CONTRIBUTING.md) | Contribution workflow と review 期待値 |
| [`CODE_OF_CONDUCT.md`](../CODE_OF_CONDUCT.md) | 参加基準と非公開の conduct report |
| [`SUPPORT.md`](../SUPPORT.md) | Support channel、必要な evidence、対応範囲 |
| [`SECURITY.md`](../SECURITY.md) | 対応 version と非公開の vulnerability report |
| [`LICENSE`](../LICENSE) | Repository の license terms |

## 判断の優先順位

文書間で矛盾がある場合は、次の順序を使います。

1. Theses
2. Security と architecture の invariant
3. Accepted ADR
4. Active work packet の goal と context
5. Active plan
6. Task checklist

root の [AGENTS.md](../AGENTS.md) がこの順序を contribution policy として運用します。下位文書は上位 invariant の例外を許可できません。

## 永続的な判断と作業時の手順

永続文書は、system が現在の形である理由を説明します。作業時の instruction は、繰り返し発生する変更を安全に行う方法を説明します。長い手順 checklist を thesis や architecture overview に置かず、focused work template、repository tool、agent skill に置いて governing invariant へ link してください。

反対に、永続的な product/security 判断を work plan だけに残してはいけません。work packet を閉じる前に、番号付き文書または ADR へ昇格してください。

## 日本語化の境界

対象利用者は日本です。README、community 文書、GitHub template、human help、TUI、公開 fault message と recovery reason は日本語を既定とします。一方、command path、flag、environment variable、JSON key、schema token、fault kind/code、capability/reference kind、opaque reference は automation の安定識別子なので翻訳しません。Chatwork から受け取る外部 text も意味を変えず、そのまま untrusted data として扱います。

番号付きの engineering 文書は、厳密な contract 用語や既存 evidence を保持する必要がある箇所で英語を残せます。ただし、そこで新たに定める利用者向け instruction は日本語の active entry documentation からも到達可能でなければなりません。`docs/work/` で明示的に保持する実験・release evidence は、後から観測の意味を変えないため原文を保持します。

## Product 文書の review

実際の capability を追加する前に、次を確認します。

1. Generic な North Star と成功指標を具体化する。
2. Primary user、対応 task、明示的な非目標を定める。
3. Credential、data store、subprocess、filesystem write、network destination を文書化し、必要なら authentication と external API の判断を完了する。
4. Compatibility と release の約束を決める。
5. 重要な claim を type、test、lint、release check のいずれかに結び付ける。
6. `task check` と `task public:check` を実行する。

派生 project が別の対象地域・言語を採用する場合は、thesis と product contract を明示的に変更し、machine identifier と external data を翻訳対象から分離してください。
