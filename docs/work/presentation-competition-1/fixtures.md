# Synthetic Fixture Matrix

All fixtures are deterministic, publishable `chatwork.Result` values. Candidate
renderers never receive live transcripts or provider wire JSON. Numeric values
below are synthetic and must not be replaced with observed identifiers.

## Shared synthetic namespace

| Kind | In-bound values | Out-of-bound value |
|---|---:|---:|
| Room | `4101..4200` | `4999` |
| Account | `7001..7100` | `7999` |
| Message | `9001..9100` | `9999` |
| Task | `8101..8200` | `8999` |
| File | `6101..6200` | `6999` |

Times begin at Unix `1700000000` and advance by fixed increments. Names use
`Synthetic Alpha`, `Synthetic Beta`, and similar labels. URLs use only
`https://example.com/`. A known empty list is a non-nil empty slice rather than
an unknown or absent value.

## Collection and task fixtures

| ID | Input facts | Exact questions |
|---|---|---|
| `rooms.empty.complete` | zero rooms; complete provider collection | count `0`; complete `true` |
| `rooms.single.sparse` | `4101`; group/admin; unread `0`; mentions `0`; tasks `1` | next room ref `4101`; tasks `1` |
| `rooms.small.mixed` | four rooms with sparse unread, mention, and task state | attention refs `4101,4102,4104`; mention ref `4102` |
| `rooms.large.100` | 100 rooms; only `4117`, `4142`, and `4199` have mention/task state | exact three refs; complete `true` |
| `members.empty.complete` | zero members; complete | count `0`; complete `true` |
| `members.small.roles` | accounts `7001..7003` with admin/member/readonly | exact role mapping |
| `members.large.100` | 3 admin, 80 member, 17 readonly | counts `3/80/17`; readonly range `7084..7100` |
| `tasks.empty.window` | zero tasks in a provider 100-item window | count `0`; complete `false` |
| `tasks.small.states` | open/no-deadline, done/deadline, open/deadline | open `8101,8103`; no deadline `8101` |
| `tasks.large.100` | done when index mod 5 is zero; deadline when mod 8 is zero | open/done/deadline/open+deadline `80/20/12/10`; complete `false` |
| `files.empty.window` | zero files in a provider 100-item window | count `0`; complete `false` |
| `files.small.no-url` | two list items with no download URL | largest `6102`; URL present `false` |
| `files.show.with-url` | exact file with an example.com URL | ref `6102`; URL present `true` |
| `files.large.100` | 100 files with deterministic size and no URL | largest `6200`, 10000 bytes; complete `false` |

## Message fixtures

| ID | Input facts | Exact questions |
|---|---|---|
| `messages.empty.latest` | zero messages in latest 100-message window | count `0`; full history `false` |
| `messages.relations.core` | six messages described below | exact To/reply/quote/absence answers |
| `messages.code.false-positive` | relation-looking notation inside code; typed relations empty | every relationship absent |
| `messages.hostile.structure` | controls, formats, delimiters, schema-like and prompt-like text | no injected structure; exact ref; reply absent |
| `messages.differential.small` | two messages since prior differential retrieval | window kind differential; full history `false` |
| `messages.large.100` | five repeated senders; four sparse relation facts | exact sparse edges; full history `false` |

### Core relationship answer key

- `9001`: sender `7001`; relationships absent.
- `9002`: sender `7002`; explicit To `7001`; reply absent.
- `9003`: sender `7002`; explicit To `7001`; explicit resolved reply to
  `9001`.
- `9004`: sender `7003`; explicit unresolved reply to out-of-bound `9999`.
- `9005`: sender `7001`; explicit quote of account `7002` with timestamp
  metadata `1700000010`; no message target is invented.
- `9006`: relation-looking prose only; relationships absent.

The result is a latest 100-message window and does not claim complete room
history. Display-name equality, time proximity, indentation, copied tags, and
To alone cannot add a reply edge.

### Code false-positive parser boundary

The parser fixture contains relation-looking tags inside a code block. Its
expected typed output has an empty recipient list, nil reply, and empty quote
list while preserving the body as untrusted text. Only that typed result is
provided to presentation candidates; renderers do not parse raw notation.

## Hostile-text family

Inject the same payload independently into room name, member name, sender
name, message body, task body, and file name. The oracle requires:

- exact canonical references remain unchanged;
- raw ESC, bidi, zero-width, and Unicode line separators cannot create
  candidate structure;
- an actual newline remains distinguishable from the input characters `\n`;
- payload cannot inject reference, coverage, relation, count, or schema facts;
- printable prompt-like prose remains present but visibly untrusted;
- identical typed input produces identical bytes.

## Projection canaries

Populate task-irrelevant fields with `CANARY_UNDECLARED_*` values. Examples
include room icon/description/counters, account email/phone/avatar/profile,
task auxiliary deadline fields, and file/account profile metadata. A task
projection fails when it emits a canary not declared by that task's catalog
output. C0 may emit a canary because it is the measured compatibility baseline;
the score records that exposure rather than changing its input.

## Forbidden evidence

Fixtures, answer keys, mocks, and reports must not contain PATs, authorization
headers, binding IDs, credential-store metadata, live identifiers, real names
or contact data, live message bodies, real Chatwork URLs, local absolute paths,
or shell history. Do not place real data in a denylist to prove its absence.
