# Work Plan: Chatwork 結果と変更意図の完全性

- Status: Accepted for implementation
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

HTTP status、外部本文、任意 CLI 入力を暗黙の provider fact として扱わず、各境界に最小の typed state を追加する。message list coverage は access limitation を別軸で保持し、message notation は wire response 失敗ではなく message 単位の relation parse state にする。room creation の provider API にない owner 語彙は authenticated account scope へ変更し、同じ PAT の `/me` で mutation 前に照合する。invite-link PUT は patch と呼ばず、provider が暗黙適用する code/approval/description をすべて explicit replacement input にする。rate-limit timing は一次資料で確認できた header または exact operation-specific error だけから作る。

## Alternatives considered

### Status/body だけで分類する

見送る。200/204/404 は制限時にも通常時にも使われ、error body/summary は provider prose で安定した public contract ではない。limitation boolean header の exact value だけを分類根拠にする。

### Malformed notation で message または一覧を失敗させる

見送る。公式 API は body を string として返し、不完全記法の挙動を保証していない。wire shape、UTF-8、ID、size は strict failure を維持するが、semantic enrichment は unknown として継続する。

### `--owner` をそのまま mutation scope として使う

見送る。POST `/rooms` に owner field はなく、caller input と PAT subject の一致を証明できない。公開 flag は `--account` に変更し、account show が生成する reference を同じ credential の `/me` と照合する。

### invite-link PUT を optional patch として送る

見送る。code omission はランダム再生成、need_acceptance omission は default 1 であり、空/部分 form が無変更になる根拠がない。明示的 code または regeneration、approval、description を full replacement として要求する。

### `Retry-After` または endpoint 名だけから待機を推測する

見送る。公式 header は `x-ratelimit-reset` であり、二つの POST は generic と room-specific の両方の 429 を返し得る。exact bounded error だけが 10 秒根拠になる。

## Design

### Public contract

- `messages list` は `access-limitation=none|partial|all` を collection header に必ず持つ。`complete=false` は room history/window completeness、access limitation は閲覧制限という別の意味を保つ。
- `messages show` の restricted 404 は `permission/chatwork_message_restricted`、通常 404 は `not_found/chatwork_not_found`。
- message record は関係抽出が確定できないとき `relation-state=unknown` を出し、body は従来どおり `untrusted` と visible escaped で出す。unknown message から reply context を拡張しない。
- `rooms create --owner` は削除し、`--account <account-ref>` を required creation scope とする。これは owner assignment ではなく same-PAT identity precondition であることを help に明記する。
- `invite-link update` は `--code` と `--regenerate-code` の正確に一つ、required `--approval`、required non-empty `--description` を要求する。これは full replacement であり、empty update は存在しない。空文字で説明を消去できるかは一次資料で確定できないため、`--clear-description` は追加しない。
- invite-link create は公式の任意 `--description` を公開する。update の explicit fields と、非対応にする入力がないことを scoped help と work evidence で確認する。
- rate-limit read fault は retryable、mutation fault は non-retryable。どちらも一回しか transport attempt しない。known timing は根拠付き最小待機情報、unknown は明示する。

### Layer changes

- Domain: message access limitation enum、relation parse state、invite replacement presence、rate timing/retryability invariant。
- Application: unknown relation を解決/selection edge に使わない。認証 gate は room-create account requirement を exact match する。
- Infrastructure: limitation headers、bounded notation parser、`/me` account-bound PAT session、full invite form、official rate headers/error classification。
- CLI and catalog: renamed account flag、explicit invite replacement inputs、new output/fault fields、unknown rendering、read/mutation rate signatures。

### Data and control flow

```text
message HTTP status + limitation headers + bounded body
  -> infrastructure classifies access separately from wire JSON
  -> exact notation becomes typed facts; ambiguous notation becomes relation unknown
  -> application resolves only typed complete facts
  -> CLI renders bounds, limitation, unknown state, and escaped body

rooms create --account
  -> parse/reference validation
  -> confirmation policy
  -> auth requirement.AccountID
  -> infrastructure PAT binding + GET /me exact comparison
  -> POST /rooms once

invite-link update explicit replacement
  -> CLI/domain cross-field validation
  -> confirmation policy
  -> one PUT with reviewed form fields
```

### Error and cancellation behavior

- malformed argv/reference/code/cross-field replacement は auth/I/O 前に invalid input。
- `/me` mismatch は authentication_context_mismatch、room create POST は 0 回。実 account ID や token を error に含めない。
- invalid limitation header combination は response contract failure であり、制限なしへ downgrade しない。
- restriction summary と error body は公開しない。
- mutation 429 は non-retryable で scoped help へ進み、automatic retry は 0。unknown mutation outcome は従来どおり read-only reconciliation。
- valid structured faults remain authoritative over generic cancellation。

### Security and public boundary

固定 Chatwork destination、PAT private binding、response/error byte limits、visible escape、synthetic fixtures、one-attempt mutation policy を維持する。新規 dependency、credential source、destination override、persistent identity state は追加しない。

## Implementation slices

1. Work packet と failing boundary tests
2. Message limitation と relation unknown domain/adapter/presentation
3. Account-bound room creation と explicit invite replacement
4. Official rate-limit timing と read/mutation contract
5. Durable docs、harness claims、agent readiness、repository gates

## Verification

- Unit and contract tests: domain enums/cross-field invariants、notation parser、limitation headers、rate header parser、provider/local clock-skew baseline。
- Negative side-effect tests: invalid invite inputs 0 auth/PUT、account mismatch 0 POST、mutation 429 1 attempt。
- Opaque-reference and complete-pagination tests: account show -> rooms create exact account ref、message/room refs unchanged。pagination contract unchanged。
- Structured output, hostile-output, and recovery tests: restricted/empty/unknown output、body controls、summary/body/token canaries、text/JSON rate timing。
- Agent-readiness scenario and discovery-round-trip count: root + one scope maximum、external post-processing 0、provider calls list/show=1、rooms create=identity read 1 + create 1。
- Manual observation: synthetic local HTTP only; no live Chatwork credential。
- Required profiles: `task check`, `task security`, `task public:check`。
- Generated-diff or artifact checks: gofmt、fixture digest drift、git diff review。

## Rollout and rollback

pre-1.0 breaking CLI changeとして `rooms create --owner` を `--account` に置き換え、invite-link update を full explicit replacement にする。message text output は access/unknown fields を追加する。安全な rollback は旧挙動へ戻すことではなく、問題の command を公開 catalog から一時停止して typed failure を返すことになる。旧挙動は誤認または暗黙 mutation を許すため compatibility alias として残さない。

## Documentation promotion

- `docs/00_theses.md`: completeness signal と semantic enrichment failure isolation
- `docs/01_product_contract.md`: message access states、room account binding、invite replacement、rate timing/retry distinction
- `docs/02_architecture.md`: header/body mapping、dynamic account requirement、unknown relation flow
- `docs/03_security_model.md`: restriction summary/error non-disclosure、pre-mutation identity check、omission safety
- `docs/04_harness.md`: claims-to-checks と required negative tests
- `docs/07_authentication.md`: account-bound PAT `/me` check
- `docs/08_external_api_contracts.md`: official rate header、message limitation、full PUT input
- `docs/09_agent_readiness_validation.md`: synthetic restricted/unknown/identity/invite/rate probes
