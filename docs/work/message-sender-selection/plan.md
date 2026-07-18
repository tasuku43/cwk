# Work Plan: Message sender selection

- Status: Completed
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Extend `messages list` with repeatable exact `--sender` filters and an optional
`--context none|replies` enum. Resolve the complete typed provider window once
inside application outcome assembly, select sender anchors in provider order,
optionally add one-hop typed reply neighbors, and return explicit source
sequence/anchor metadata for presentation.

## Alternatives considered

### Sender only

This is smallest and remains available through the default `context=none`, but
as the only interface it makes a reply parent disappear and weakens immediate
conversation understanding.

### Boolean `--include-related`

This is short but “related” cannot tell an agent whether To, quotes, reply
children, parents, or whole threads are included. A typed enum names the only
supported context and permits a later reviewed value without changing flag
meaning.

### Arbitrary filter expressions

An expression language is powerful but reintroduces parsing, precedence,
discoverability, escaping, and maintenance cost before repeated task evidence
justifies it. Finite typed flags cover this requested outcome.

## Design

### Public contract

```text
cwk messages list --room <room-ref> [--window changes|recent]
  [--sender <account-ref>] [--context none|replies]
```

- Repeating `--sender` matches any supplied canonical account reference.
- `--context` is optional, defaults to `none`, and is invalid without at least
  one sender.
- `replies` includes direct parent and child message nodes only when an existing
  typed reply edge resolves inside the source window and touches a sender match.
- Sender/context selection is an AND between the sender predicate and context
  expansion mode, not a general Boolean expression.
- Output remains text-only and adds selection metadata only when filtering is
  active: source count, sender refs, context mode, source sequences, and anchor
  sequences. Canonical message refs remain positional and unchanged.

### Layer changes

- Domain: add validated message-filter inputs and selection metadata.
- Application: move full-window reply resolution into service outcome assembly
  and add pure stable selection/context expansion.
- Infrastructure: continue one existing request; prove filter fields never
  become undocumented query parameters or additional calls.
- CLI/catalog: bind repeated account refs and context enum, render source
  sequence gaps and one selection line, and update scoped help.

### Data and control flow

```text
validated room/sender/context inputs
  -> one existing bounded Chatwork message request
  -> typed provider-order message window
  -> resolve typed in-window replies
  -> select sender anchors and optional direct reply neighbors
  -> downgrade displayed replies whose parent was omitted
  -> render source sequences, anchors, canonical refs, and typed edges
```

### Error and cancellation behavior

Invalid, missing, duplicate, or excessive filter references, an unknown context,
or context without sender fail as invalid input before authentication/provider
I/O. Provider faults, cancellation, retryability, timeout, and recovery remain
unchanged. Selection itself performs no I/O and introduces no partial success.

### Security and public boundary

No secret, dependency, destination, persistence, or mutation is added. External
text remains structurally escaped. Canonical account filters are validated
exact bytes and raw text cannot influence selection.

## Implementation slices

1. Commit this bounded work packet.
2. Add domain filter/selection types and application semantic tests.
3. Bind catalog/parser/runtime and verify one provider request.
4. Render filtered source sequences and selection metadata; update goldens.
5. Add active no-post-processing readiness fixture and durable documentation.
6. Run final gates, close the packet, and commit the completion record.

## Verification

- Domain/application truth tables for no filter, one/many senders, zero matches,
  direct reply parent/child context, interleaved branches, omitted parents, raw
  tags, duplicate refs, and stable source order.
- Adapter request contract proves only the existing room/force query and one
  transport call.
- CLI argv/help/runtime tests prove repeatability, enum validation, pre-I/O
  rejection, source sequence gaps, anchors, and direct reference reuse.
- Existing unfiltered message golden remains byte-identical.
- Active synthetic agent scenario completes with one cwk invocation and zero
  external post-processing.
- `task check` on final HEAD.

## Rollout and rollback

This is an additive pre-1.0 input contract. Unfiltered invocations retain their
existing output bytes. Removing the new optional flags and selection metadata
is the safe code rollback; no remote or local state migration exists.

## Documentation promotion

Promote finite typed filtering, application-owned selection, source sequence,
reply-context bounds, and agent-readiness evidence into theses, product,
architecture, harness, readiness docs, README, and `$add-capability`.
