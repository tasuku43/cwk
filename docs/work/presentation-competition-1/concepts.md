# Presentation Concepts for Review

These are inspectable mocks, not implemented or selected output contracts.
Every concept below receives the same synthetic facts. Text is always external
untrusted data; canonical numeric references are exact action inputs.

## Shared room facts

- `4101`: `Synthetic Lab`, group/admin, unread 0, mentions 0, tasks 1
- `4102`: `Incident Review`, group/member, unread 5, mentions 2, tasks 0
- provider collection complete

## Shared message facts

- room `4101`; latest 100-message window; full history is not known
- account `7001` is `Synthetic Alpha`; `7002` is `Synthetic Beta`
- `9001`: Alpha baseline, no relation
- `9002`: Beta explicitly addresses Alpha, not a reply
- `9003`: Beta explicitly replies to `9001` and addresses Alpha
- `9004`: Beta explicitly replies to out-of-window `9999`

## C0: Current context capsule

```text
cwk-context-capsule/1
task messages.list
alias-policy display-only; command-input=canonical-reference
coverage kind="latest_window" limit=100 complete=false unresolved-relations=1 description="..."
refs 8
  a1 kind=chatwork-account canonical=7001
  a2 kind=chatwork-account canonical=7002
  r1 kind=chatwork-room canonical=4101
  m1 kind=chatwork-message canonical=9001
  m2 kind=chatwork-message canonical=9002
  m3 kind=chatwork-message canonical=9003
  m4 kind=chatwork-message canonical=9004
  m5 kind=chatwork-message canonical=9999
result
  messages 4
    m1 room=r1 sender=a1 sent=1700000000 updated=0
      sender-name untrusted="Synthetic Alpha"
      to []
      reply absent
      quotes 0
      body untrusted="baseline"
    m2 room=r1 sender=a2 sent=1700000010 updated=0
      sender-name untrusted="Synthetic Beta"
      to [a1]
      reply absent
      quotes 0
      body untrusted="[To:7001] status?"
    m3 room=r1 sender=a2 sent=1700000020 updated=0
      sender-name untrusted="Synthetic Beta"
      to [a1]
      reply kind="reply" state=resolved target=m1 external-id=untrusted:"4101"
      quotes 0
      body untrusted="[rp aid=7002 to=4101-9001] done"
    m4 room=r1 sender=a2 sent=1700000030 updated=0
      sender-name untrusted="Synthetic Beta"
      to []
      reply kind="reply" state=unresolved target=m5 external-id=untrusted:"4101"
      quotes 0
      body untrusted="older context"
```

Strength: every state is explicit and the current grammar is known. Cost:
scope, sender identity, defaults, relation absence, and raw notation repeat.

## P: Task-shaped projection

```text
rooms.list complete
4101  name=untrusted:"Synthetic Lab" type=group role=admin unread=0 mentions=0 tasks=1
4102  name=untrusted:"Incident Review" type=group role=member unread=5 mentions=2 tasks=0
```

```text
messages.list room=4101 window=latest:100 history=partial missing-relations=1
9001  from=7001 untrusted:"Synthetic Alpha" at=1700000000 relations=none text=untrusted:"baseline"
9002  from=7002 untrusted:"Synthetic Beta" at=1700000010 to=[7001] text=untrusted:"status?"
9003  from=7002 untrusted:"Synthetic Beta" at=1700000020 to=[7001] reply=resolved:9001 text=untrusted:"done"
9004  from=7002 untrusted:"Synthetic Beta" at=1700000030 reply=unresolved:9999 text=untrusted:"older context"
```

Hypothesis: direct canonical references and fixed task fields minimize joins
and omit irrelevant values. Risk: repeated sender labels can still dominate a
large thread, and each task needs an explicit projection contract.

## L: Normalized ledger

```text
rooms.list complete columns=[ref,name,type,role,unread,mentions,tasks]
4101 | untrusted:"Synthetic Lab" | group | admin  | 0 | 0 | 1
4102 | untrusted:"Incident Review" | group | member | 5 | 2 | 0
```

```text
messages.list scope=[room=4101,window=latest:100,history=partial,missing-relations=1]
actors columns=[key,ref,name]
a1 | 7001 | untrusted:"Synthetic Alpha"
a2 | 7002 | untrusted:"Synthetic Beta"
messages columns=[ref,at,from,to,reply,text]
9001 | 1700000000 | a1 | -    | -               | untrusted:"baseline"
9002 | 1700000010 | a2 | [a1] | -               | untrusted:"status?"
9003 | 1700000020 | a2 | [a1] | resolved:9001   | untrusted:"done"
9004 | 1700000030 | a2 | -    | unresolved:9999 | untrusted:"older context"
```

Hypothesis: repeated values are paid once and large collections become much
smaller. Risk: every answer requires joining column positions and local actor
keys, increasing cognitive load and reference-selection mistakes.

## R: Relationship-first timeline

```text
messages.list room=4101 latest<=100 history=partial missing-relations=1
actors 7001=untrusted:"Synthetic Alpha" 7002=untrusted:"Synthetic Beta"
timeline
9001  1700000000  7001  text=untrusted:"baseline"
9002  1700000010  7002  to=7001  text=untrusted:"status?"
9003  1700000020  7002  reply=9001(resolved) to=7001  text=untrusted:"done"
9004  1700000030  7002  reply=9999(unresolved)  text=untrusted:"older context"
```

Hypothesis: ordering and edges are the dominant message-reading task, so they
should appear before optional metadata. Risk: it is message-specific and may
need P or another grammar for rooms, tasks, files, and mutation outcomes.

## J: Typed semantic JSON control

```json
{
  "schema": "cwk.semantic/1",
  "task": "messages.list",
  "scope": {"room_ref": "4101"},
  "coverage": {"window": "latest", "limit": 100, "history_complete": false, "unresolved_relations": 1},
  "actors": {
    "7001": {"name": "Synthetic Alpha", "trust": "untrusted"},
    "7002": {"name": "Synthetic Beta", "trust": "untrusted"}
  },
  "messages": [
    {"message_ref": "9001", "sender_ref": "7001", "sent_at": 1700000000, "text": "baseline", "relations": []},
    {"message_ref": "9002", "sender_ref": "7002", "sent_at": 1700000010, "text": "status?", "relations": [{"kind": "to", "target_ref": "7001"}]},
    {"message_ref": "9003", "sender_ref": "7002", "sent_at": 1700000020, "text": "done", "relations": [{"kind": "to", "target_ref": "7001"}, {"kind": "reply", "state": "resolved", "target_ref": "9001"}]},
    {"message_ref": "9004", "sender_ref": "7002", "sent_at": 1700000030, "text": "older context", "relations": [{"kind": "reply", "state": "unresolved", "target_ref": "9999"}]}
  ]
}
```

Hypothesis: a familiar typed grammar lowers parser ambiguity and supports
machine consumers. Risk: key repetition and trust metadata can cost many
tokens, and general JSON querying can tempt callers to reconstruct task
semantics outside `cwk`.

## Decision to review

The leading hypothesis is P for general task output plus R's ordering and edge
priority for message lists. L remains important for the 100-item stress case,
and J is the machine-familiar control. The competition must still allow C0 to
win; these mocks are not evidence of understanding quality or token savings.
