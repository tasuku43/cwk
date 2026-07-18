# Context: Flat message adjacency presentation

## Verified facts before implementation

- `.harness/project.json` has profile `ready`.
- `internal/infra/chatworkapi` maps provider array order directly into
  `Result.Messages`; the application relation resolver clones that slice and
  does not reorder it.
- `Message.Reply`, `Message.Recipients`, and `Message.Quotes` are typed semantic
  facts. `ResolveMessageRelations` resolves a reply only when its explicit target
  is present in the same room window; it never inspects body text.
- An unresolved reply may carry a canonical message target or no target. Absence
  is valid only while unresolved.
- Canonical references are validated positive decimal provider values and are
  accepted unchanged by downstream command inputs. Display aliases do not belong
  to `chatwork.Reference`.
- The current projection repeats room, sender, trust, and relation-state fields
  per message. It uses `quoted` projection to expose controls, formats, Unicode
  line separators, backslashes, and structural characters safely.
- `messages list` success output supports text only. JSON success formats belong
  to other commands; agent help and structured error JSON are separate contracts
  and must remain unchanged.
- The semantic result currently loses the requested room when a message list is
  empty. The result needs an exact, request-bound room scope so the header can be
  correct without reconstructing it in presentation.
- Recipient values contain canonical account references but no recipient names.
  Sender accounts contain names. Therefore the actor dictionary covers distinct
  known senders. A To target not present among those senders remains a canonical
  account reference instead of receiving a fabricated name or alias.
- Existing message fixtures and agent-readiness scenarios cover resolved and
  unresolved reply, To, quote, absent relation, hostile body text, bounded
  coverage, and exact reference flow. They require updates, not replacement.

## Superseded direction

- The initial packet selected a physically reordered indented reply tree. Before
  implementation, the owner superseded that decision with a single flat format.
  No indented renderer was implemented, and no switch or legacy alternative will
  be retained.

## Constraints

- The adjacency list is a presentation of typed facts, not a second relation resolver.
- `#N` is the one-based position in the provider-returned slice. Physical
  reordering is forbidden; output lines remain in that exact order.
- Each node retains full `message-ref`; `reply=#N` is only a local cross-reference.
- Reply, To, and quote are explicit fields; none is represented by indentation.
- Resolved is the default and omitted. Only unresolved relations carry an
  explicit `?` marker.
- Actor aliases are assigned by first sender occurrence in provider order.
  Canonical account identity, not display name, deduplicates actors.
- Repeated observations of one account must agree on its displayed name; an
  inconsistent semantic window fails closed instead of silently discarding text.
- Every body is quoted on the same physical line as its message record. Newlines
  and hostile structural text are escaped by the existing terminal-safe rule.
- No blank lines appear between message records.
- Existing collection bounds and completeness remain once on the header because
  they are required uncertainty facts, not per-message repetition.
- Send timestamps remain available in the typed semantic fixture. The flat list's
  public temporal contract is provider sequence; absolute timestamp selection is
  an implementation decision to verify against the updated catalog and golden.

## Unknowns closed by this packet

- No alternate grammar competition will be started; the owner explicitly chose
  this chronological adjacency-list grammar.
- No recipient-name lookup will be added; it would add calls and semantics beyond
  this change.
- Quote relations will not become reply edges. Their existing typed facts must not
  be silently reclassified or inferred.
