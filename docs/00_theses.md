# Project Theses

This document decides ambiguous product and engineering choices for Chatwork CLI (`cwk`). A thesis states a representation-independent product hypothesis, its consequences, and the evidence that can disprove it. Concrete presentation designs remain replaceable until reviewed evidence and an explicit compatibility decision promote one into a public contract.

## North Star

**An agent can translate a user's Chatwork request into one exact `cwk` task, invoke it without guessing, and understand a bounded, trustworthy result with no routine external reconstruction. Among outputs that meet the required understanding and safety quality, `cwk` minimizes token cost.**

The primary user is a developer or operator in Japan who delegates Chatwork work to a coding agent from a shell or automation environment. Human usability in Japanese remains important, while command certainty, agent understanding quality, and token efficiency are the first optimization targets.

## Language contract

Japanese is the single default language for user-facing prose. Human help,
interactive terminal guidance, public fault messages and recovery reasons,
README and community-facing repository documents use Japanese without a
runtime locale switch. Stable machine identifiers remain locale-neutral ASCII:
command paths, flags, environment variables, capability and reference kinds,
fault kinds and codes, JSON keys, schema versions, allowed values, and the
reviewed success-output grammar are not translated. Chatwork text remains
untrusted external data and is preserved rather than translated.

This boundary is a product and compatibility decision, not an implementation
shortcut. A future additional locale requires a reviewed fallback and message
catalog contract. Explicitly retained experimental or release evidence remains
in its original language so that localization cannot rewrite prior decisions.
Ordinary work packets are temporary review artifacts: before completion, their
durable conclusions move into numbered documents, ADRs, tests, or code, and the
packet leaves the current tree while Git preserves its history.

The product is not an endpoint mirror and is not measured by the compactness of one syntax. Its first complete implementation nevertheless has a finite coverage obligation: every operation in the official 2026-07-18 Chatwork API snapshot must be reachable through at least one reviewed user-task workflow. A smaller output is worse when it causes command mistakes, hides missing context, weakens identity, or makes an agent infer relationships.

## Axiom 1: A supported outcome is operationally closed

When `cwk` claims to support a user outcome, it owns the deterministic selection, joining, interpretation, and task-specific transformation needed to use the result.

### Consequences

- Routine success paths do not require `jq`, `grep`, custom joins/parsers, raw Chatwork-notation interpretation, source inspection, or exploratory API calls.
- Direct extraction of a declared field or opaque reference is allowed; reconstructing product semantics is not.
- Repeated external processing is evidence of a missing or overbroad capability, not a workaround to teach every agent.
- Commands express user outcomes rather than provider endpoints.
- A common deterministic workflow belongs in an application use case, not an agent prompt.

### Enforcement

- Catalog entries declare one outcome and complete input/output semantics.
- Agent transcripts count external processing and fail a supported scenario when the count is nonzero.
- Work packets record repeated pipelines as thesis evidence.

## Axiom 2: The selected command set bounds cwk dispatch

An agent that knows the user's desired outcome should reach an executable
invocation through bounded text help. Complete catalog metadata remains an
internal validation source; recovery detail is delivered by the structured
runtime error at the point where it is needed.

### Consequences

- `cwk --help`, `cwk <namespace> --help`, and exact-command `--help` are the
  only public help projection. The retired `help ... --format agent` form fails
  explicitly and directs the caller to `--help`; `--format json` remains a data
  output choice only where a task declares it.
- Root help is a compact namespace/outcome index. Namespace help emits every
  visible leaf's complete usage line, required flags, and required one-of input
  groups, so an agent can begin
  composing the command after one namespace lookup rather than opening every
  leaf in turn.
- Each nonempty root namespace derives up to three distinct Japanese outcome
  lines from active leaves' catalog `Summary` values. Each line points to that
  leaf's exact `--help` selector; duplicate summaries collapse, and the number
  of remaining distinct outcomes points to exact namespace help. Saved disabled
  leaves contribute names only, a disabled-only namespace has no outcomes, and
  an unconfigured view remains the four-command control plane. The message-link
  discovery guide appears only when the active catalog contains the exact
  `messages resolve-link` path and points to its exact `--help` selector. A
  separate exact `messages show --help` route appears only when that leaf is active.
  In a saved resolver-only view, the root guide asks a person to enable show via
  config rather than presenting it as executable. A marked static fallback
  projects the complete catalog and explicitly does not assert active selection.
- Root and static-fallback help byte counts are measured as UTF-8 encoded bytes.
  Root has no separate hard byte ceiling; concise catalog summaries and the
  per-namespace outcome limit keep the navigation projection bounded in
  practice. The exact-command 2,048-byte ceiling remains independently enforced.
- Exact-command help is at most 2,048 encoded bytes and contains only one usage
  line, each flag's name/type/required, required-one-of, or optional state and
  its complete one-line description, combination constraints, and at most two
  examples. If the complete projection exceeds the byte limit, remove optional
  examples from the end until it fits; never shorten descriptions or
  constraints. A command whose complete projection still exceeds the limit
  without examples is invalid. It does not preload
  output schemas, authentication, failure catalogs, mutation facts, or
  reference workflows.
- `AgentContract` and other catalog metadata remain complete, structured, and
  machine-validated behind that projection. Runtime structured errors own
  `kind`, `code`, actionable `next_action`, and the applicable usage instead of
  requiring a speculative full-contract help read.
- The complete catalog remains the product and release contract. A saved exact-
  path selection defines the operation scope that this `cwk` process will
  dispatch: a configurable task outside that set is not invocable through this
  process. The smaller active view also reduces agent token use and selection
  mistakes. Routing, active usage/examples/workflows, and typo suggestions use
  the active view; disabled task names may be exposed separately without their
  invocation details.
- `help`, `doctor`, `version`, and the sole interactive `config` selector remain
  available in every configured operation scope so local diagnosis and scope
  recovery remain possible. This is a boundary on dispatch by this `cwk`
  process, not protection from the same OS user: that user can edit or remove
  the preference, automate a terminal, change the configuration base, install
  another executable, or use a PAT outside `cwk`. It establishes no human
  identity and does not replace PAT validation, provider permissions, exact
  references, operation intent, or mutation confirmations.
- With a saved valid selection, an exact known disabled canonical command,
  catalog-declared alias, or exact scoped/trailing help request returns the
  non-retryable `command_disabled` fault and asks the user to enable that
  command through `config`. One exact catalog root namespace by itself is a
  help selector, equivalent to `help <namespace>` and `<namespace> --help`;
  it renders the active namespace view and may list disabled canonical paths
  in a separate name-only section. It never exposes disabled usage, examples,
  or workflows. Other unknown or incomplete paths remain `unknown_command`.
- Until a command-selection profile has been saved, the active view contains
  only those four local control commands, so this `cwk` process dispatches no
  Chatwork task. Human root help explicitly says that `config` is unset, why
  selecting only relevant tasks lowers agent token use and selection mistakes,
  and how to run `config`. A known configurable task, exact namespace, or its
  scoped help fails as `command_selection_required` before PAT resolution or
  provider I/O; an actually unknown or incomplete path remains
  `unknown_command`.
- `config` accepts one terminal-native interaction: Up/Down navigates, Space
  toggles, Enter validates and saves, and `q` leaves the prior selection
  unchanged. Each choice retains a literal read/create/write badge; optional
  color only supplements that text and never invents destructive meaning. If
  the complete current identity is not visible, mutation keys remain inactive.
- A non-terminal selector invocation fails explicitly. Before Enter, every
  exit restores terminal state and performs zero saves; Enter validates the
  active graph, restores the terminal, and only then crosses one mutation
  boundary.
- Invalid persisted state never masquerades as a deliberately empty
  dispatch scope. Static root, namespace, and exact-command help remains available from
  the immutable complete catalog, including through trailing `--help`; this is
  read-only invocation documentation, not a claim that those tasks are enabled.
  Root and namespace fallback help say that the selection could not be read and
  direct recovery to `doctor`; exact help keeps its bounded invocation-only form.
  Every non-help Chatwork invocation still returns the typed load fault.
  Read-only `doctor` reports command-selection source, state, enabled/disabled
  counts, and a deterministic fingerprint for uncertain-save reconciliation.
  `help` describes commands; it does not reconcile saved state.
- Public help declares the bounded invocation facts above; catalog validation
  separately proves effects, authentication, output semantics, completeness,
  failures, recovery, and reference workflows.
- Commands do not silently search again, choose a display-name match, or rely on hidden defaults. A dedicated discover command may filter a complete typed candidate collection by external display text, but it returns every match with canonical references and never promotes one match to an act-command identity.
- `rooms list --query <text>` owns the common room-name lookup without `grep`.
  It performs Unicode simple-case-folded substring matching over one complete
  typed room read, with no normalization, trimming, ranking, provider query, or
  automatic selection. Zero, one, and many matches are all successful discover
  outcomes in provider order. `members find` retains its separate exact
  case-sensitive substring contract.
- A catalog-declared deprecated flag alias may preserve a finite migration
  without changing task semantics. `messages list --limit` aliases `--count`
  for at least two minor releases, succeeds, and writes a migration warning to
  stderr. Unknown flags fail without correction but include a deterministic
  did-you-mean suggestion from that command's declared flags and aliases.
- Finite command-path compatibility spellings also belong to the owning
  `CommandSpec`; they are not a router-side registry. `messages get` normalizes
  to `messages show`, and `rooms members find|list|replace` normalize to the
  matching `members` command before active-view dispatch. The canonical path
  remains intent, help, recovery, and handler identity.
- The documented `messages list` default is the latest bounded `recent`
  window, matching the common conversation-understanding task. Provider-stateful
  differential retrieval is an explicit `--window changes` choice.
- A sender, period, or index selection defaults its direct typed context to
  `replies`, matching the measured common investigation path. Explicit
  `--context none` is the lower-output opt-out. Context cannot be supplied by
  itself because it has no primary anchor set.
- The shortest `messages list` also owns five declared exact-message fetch slots
  for every unique explicit same-room reply target missing from that bounded source.
  `--resolve-relations 0` is the opt-out and 1..100 is a deliberate alternate
  call budget; every result reports the limit, attempts, and target outcomes.
- Structured recovery names an exact next command rather than prose that the agent must reinterpret.

### Enforcement

- Catalog, routing, and help derive from `cli.Catalog`.
- Always-on selector facts and the profile-failure recovery island derive from
  that same catalog rather than another path list.
- Full-catalog validation and capability coverage remain independent of the
  catalog-derived active view. View validation rejects a visible recovery edge
  or required-reference consumer whose prerequisite command is hidden.
- Saved-profile disabled-command tests prove canonical paths and catalog
  aliases fail as `command_disabled` before dispatch, with exact `config`
  recovery and zero handler, PAT, or provider calls. Exact disabled help follows
  the same failure; bare catalog namespaces render active help with disabled
  names only, while unknown/incomplete non-namespace paths remain unknown and
  re-enabling a path restores its ordinary handler contracts.
- Root and namespace help tests prove disabled canonical names are separated
  from active usage details, including namespaces whose commands are all
  disabled. Bare namespace selectors match explicit namespace help and do not
  initialize a provider; workflows, examples, and typo suggestions remain
  active-only.
- Selector tests prove terminal restoration and zero saves before Enter,
  non-terminal rejection, deterministic effect badges and narrow rendering,
  blocked unseen mutations, declared uncertain-error grammar, doctor
  fingerprint reconciliation, and migration of only the formerly selectable
  `doctor` and `version` paths.
- First-run tests prove the control-only text-help view, the explicit
  human help explanation, `state=unconfigured source=missing` diagnostics,
  exact `config` recovery, and zero PAT/provider calls for known Chatwork paths.
- Catalog validation and text-help tests enforce the 2,048-byte exact-command
  ceiling, the exact-help content boundary, namespace leaf
  usage/required-flag/required-one-of projection, and explicit
  rejection of the retired agent format.
- Help tests prove that every catalog command belongs to exactly one root entry
  and round-trips through its canonical namespace; examples resolve against the
  active catalog and never exceed two for one exact command.
- Agent-readiness tests reject command probing, full-contract preloading,
  external help scraping, and undeclared follow-up calls.

## Axiom 3: Semantics precede presentation

Chatwork data is converted into a typed, provider-independent task result before presentation. Presentation may reorganize or encode that result, but cannot invent, strengthen, or silently discard semantics required by the task.

### Consequences

- Provider wire JSON is not the public domain model.
- Explicit To, reply, quote, identity, ordering, coverage, and unresolved-reference facts remain distinguishable when relevant to the outcome.
- Provider-authored access limitation is independent from a bounded window:
  normal zero, partial restriction, full restriction of the requested window,
  missing identity, and a restricted exact message are distinct outcomes.
- A malformed reviewed notation form preserves the terminal-safe external body
  and the rest of the result while marking that message's whole relation set
  `unknown`; it never turns an unproved relation into `absent` or retains a
  selectively stronger subset.
- To does not become a reply; quoted prose, display names, and time proximity do not create relationships.
- Quote author and timestamp may identify exact observed messages as bounded
  source-window candidates only when room, canonical author reference, and the
  whole Unix timestamp match. Such candidates always remain unresolved and
  identity-unproven; they never become reply edges or fetch targets. Compute
  them once from the original bounded source before local selection or byte
  delivery, and retain coverage/access limits alongside the candidate state.
- Unresolved replies and unresolved quotes are separate counts. A candidate
  quote remains in the unresolved-quotes count; unknown relation sets remain a
  separate count.
- Missing or out-of-bound context remains observable rather than being hidden to make an output look complete.
- A known same-room reply reference may be completed by a bounded exact read;
  fetched context remains separate from source-window membership, and a
  not-found, restricted, or unspent target remains typed rather than silently
  disappearing. No To, quote, name, time, or prose creates a fetch target.
- A trustworthy recent source identifies its oldest reachable message. A
  requested period wholly or partly before that boundary is reported as
  unreachable rather than being presented as an ordinary empty match.
- Recurring bounded selection is expressed through finite typed task inputs and
  applied to the typed result. It is not delegated to output parsing or pushed
  into undocumented provider query parameters.
- Canonical opaque references remain available for declared next actions.
- Domain, application, and infrastructure contracts do not depend on a candidate presentation grammar.

### Enforcement

- Shared semantic fixtures have an answer key independent of any renderer.
- Negative tests reject fabricated relationships and silent completeness claims.
- Candidate presentations are evaluated against the same semantic facts and canonical references.

## Axiom 4: Presentation is versioned and replaceable

No concrete output syntax is a thesis. The first complete implementation deliberately selected the context-capsule presentation, candidate C (`cwk-context-capsule/1`), so API work could close against one high-quality contract. It remains the first stable baseline. A P-derived task projection (`cwk-task-projection/1`) later replaced it through an explicit owner compatibility decision after Competition 1 was inconclusive. The default then received a reviewed, headerless subtraction. The latest explicit pre-1.0 decision refines only `messages list` into a flat chronological adjacency list with one actor dictionary and explicit reply edges; it does not physically reorder or indent the provider window. Normal success output still starts directly with the task result and carries compatibility through the release, catalog, documentation, and golden tests. These decisions accepted breaking text-contract migrations; none claims that P won the defective benchmark. Later presentation changes remain evidence-informed and require an explicit compatibility decision.

### Hard constraints

Every eligible presentation must:

- preserve the semantic answer key and exact canonical references;
- expose task-relevant bounds, missing context, and uncertainty;
- keep external text structurally separate and marked as untrusted data;
- be deterministic for the same typed input;
- require no undocumented parsing convention or external post-processing for the evaluated outcome;
- preserve stdout, stderr, exit, failure, and completeness contracts.

The current headerless task projection additionally follows a subtractive rule:
it emits only catalog-declared task facts, exact canonical references,
task-relevant bounds/completeness/uncertainty, and structural trust framing for
external text. `messages list` may factor repeated sender identity and name into
one deterministic document-local actor dictionary; each message still carries
its full canonical message reference, provider sequence, and explicit typed
edges. One fixed schema assigns the positional message reference, actor, send
time, and quoted body, so those records do not repeat field labels. The actor
alias is never command identity. The projection does not add
raw Chatwork notation as semantic structure, undeclared provider or wire fields,
duplicated coverage prose, derived depth/thread/root/child facts, or helpful
non-contract defaults. Declared message bodies remain visible as untrusted
external text; they are not reparsed by presentation to invent semantics.
The message header separately exposes `access-limitation=none|partial|all` and
the count of unknown relation sets. Only an affected record adds the optional
`relation-state=unknown`; omitting it means the reviewed relation set was
complete, not that external text contained no relation-like notation.
Every complete explicit `[rp]` is a typed reply edge in provider notation
order. One reply uses the compatible scalar form `reply=#N`; two or more use
the compact form `reply=[#N,#M]`. A valid multiple-reply set is not unknown.
Supplemental reply parents use distinct `relation-context` records with
`source` or `fetched` provenance; unavailable or budget-exhausted parents use
`relation-gap`. Neither form receives a provider source sequence. The separate
`relation-resolution` record exposes the configured exact-read limit and the
actual number of additional provider calls.

When `messages list` selects exact senders, repeated sender inputs use OR
semantics. An active selection defaults to `replies`, a bounded, direct one-hop expansion
over typed in-window reply edges; it does not mean a whole thread or include
To/quote/body-derived neighbors. Explicit `none` skips that expansion. Selection preserves the original provider
sequence, including gaps, distinguishes sender matches from added context once
per document, and does not claim that two selected speakers form an exclusive
conversation.

Repeated `messages list --contains <text>` terms are exact, case-sensitive
literal substrings of typed message bodies. Terms use OR; sender and period
predicates remain additional AND conditions. The product does not trim or
normalize terms, interpret Chatwork notation, search through the provider, or
extend the bounded source window. At most 100 nonempty, valid UTF-8 terms of
1,024 bytes each are accepted before authentication; duplicate terms retain
their supplied order and value.

Message periods are likewise finite typed selection over that same bounded
provider result. Inclusive `--since` and exclusive `--until` accept only
whole-second RFC3339 values with an explicit offset. `--on` instead accepts one
exact `YYYY-MM-DD` or the relative task words `today` and `yesterday`; it uses
the fixed `Asia/Tokyo` calendar, resolves an injected clock exactly once, and
publishes the effective concrete day and half-open Unix interval. `--on` cannot
be combined with explicit bounds. This selection neither becomes an
undocumented provider query nor claims access outside the returned window.

Optional `messages list --start-index` and `--count` use SCIM-derived index
semantics over the one bounded provider result without becoming provider
pagination. `start-index` is the one-based first primary-message rank and
defaults to 1; `count` is the requested maximum number of primary anchors, not
an inclusive end rank. Thus `--start-index 11 --count 20` selects ranks 11
through 30. The candidate set contains messages that satisfy the exact-sender
OR predicate, literal-body term OR predicate, and half-open period predicate
when those filters are present; different predicate groups combine with AND.
Typed send time establishes candidate rank;
equal times prefer the later provider position. Selected records are still
emitted in their original provider order with original source sequences.
The default `replies` context runs afterward unless explicitly set to `none`,
and may therefore make displayed
count exceed `count` or add a direct neighbor outside the primary period.
Typed selection metadata keeps effective period bounds, candidate count, start
index, requested count, actual items per page, and optional next start index
separate from the provider's 100-message `source-limit`. The text projection
shows `candidate-count` for body-term, period, or index/count selection;
sender-only output keeps its established shape. A source that exceeds its
declared coverage fails before selection can hide the violation. Because this
is a stateless rank over separate provider snapshots, intervening source changes
may shift later ranks; the command does not claim snapshot stability.

After selection and in-window reply context, `messages list` resolves unique
explicit same-room reply-parent chains referenced by displayed records. A parent
already present in the original source is attached as supplemental context
without spending a slot. Otherwise each target consumes at most one exact-read
slot, in breadth-first first-reference order, and the default public budget is
five. A supplemental message's explicit same-room parent joins the queue; a
visited set stops duplicates and cycles. Not-found and restricted are per-target results; transient,
rate, cancellation, malformed, and unclassified permission faults abort the
whole command instead of returning partial success. This closure does not
discover arbitrary history and does not follow cross-room references.

Optional `messages list --max-body-bytes` is a presentation-only cap on each
body record that is actually emitted, including selected anchors, in-window
reply context, and emitted relation context. It takes the longest prefix that
fits the raw UTF-8 byte limit without splitting a rune, then applies the normal
terminal-safe visible projection. A truncated record states
`body-truncated=true` and its `original-body-bytes`; this does not alter the
typed body used for contains matching or relation resolution and does not
change source, selection, or byte-delivery completeness. Zero is a valid cap
that yields an empty visible prefix for each nonempty body. If `--max-bytes` is
also present, this per-body projection happens before the escaped overall
output budget is measured. Omitting the option preserves the established
output bytes.

Optional `messages list --jst` and `messages show --jst` append a fixed
`+09:00` timestamp to each emitted message while retaining its Unix send time.
Out-of-range four-digit local years render as `unavailable`. This optional
projection adds no provider request or inferred relationship, and omitting it
preserves the established output bytes.

`messages show --context replies` owns the distinct exact-anchor workflow that
agents otherwise reconstruct with a second command. It performs the exact
message read first, then one declared `recent` message-list read in the same
room and authentication binding. Only provider-order messages whose typed,
same-room reply edge points to the anchor become direct children; To, quotes,
body text, names, and time proximity never qualify. The child source publishes
its count, fixed 100-message limit, completeness, access limitation, and
original sequences, so an empty list cannot imply that older or restricted
children do not exist. This fixed read does not consume
`--resolve-relations`; that separate 0..100 budget continues to resolve the
anchor's outgoing reply-parent chain. Omitted context and explicit `none`
retain the single exact read.

An explicit `messages show --around N` (N is 0..100) adds positional neighbors
from that same bounded recent source. When combined with reply context, the
command shares exactly one list read after the exact anchor read. It selects
only the up to N records immediately before and after the anchor in original
provider sequence, deduplicating a record also emitted as a direct child while
preserving its actual typed reply edge. Adjacency never creates a relationship.
The result states whether the anchor is in the source, outside an unrestricted
window, or unknown because access is restricted; a missing anchor has no
inferred neighbors. It also publishes requested and actual side counts,
positions, source count/limit/completeness, and access limitation. Explicit
`--context none` is valid with `--around`; omitted context behavior is
unchanged. Explicit `--around 0` still reads and reports the source/anchor
state, while returning empty before/after slices.

For a nonempty, unrestricted `recent` source with valid typed send times, the
oldest reachable boundary is the minimum source send time with provider order
breaking ties. Period selection classifies its bounds as within, partly before,
or wholly before that boundary. Differential, empty, access-limited, or
otherwise unprovable sources report `unknown`; the boundary is reachability of
this operation, never proof of complete room history.

The same fixed-schema rule applies to the reviewed homogeneous read
collections for contacts, rooms, members, personal tasks, room tasks, files,
and contact requests. Each declares external-text trust and its schema once,
then emits one terminal-safe provider-order record per item. Canonical
references remain literal reusable positions, and optional terminal suffixes
never shift required positions. Only the message projection uses aliases.

### Optimization objectives

Among eligible candidates, prefer the Pareto frontier across:

- agent answer correctness and relationship understanding;
- correct next-command and reference selection;
- input/output token use;
- additional tool invocations and processing steps;
- serialized bytes, latency, and implementation/maintenance cost;
- human reviewability when it affects safe supervision.

Token count is not optimized below the required understanding-quality floor. Numerical thresholds, tokenizer/model versions, fixtures, and repetitions for later optimization are chosen in a presentation-competition work packet, not invented in this thesis or used to block the first complete implementation.

## Axiom 5: Presentation implementations stay behind one semantic boundary

The historical context capsule, the current task projection, and any future presentation hypotheses consume the same provider-independent semantic boundary. Candidate C became the first stable presentation by explicit product decision. Competition 1 did not produce an authoritative winner because its benchmark/oracle defects made the result inconclusive. The owner then deliberately superseded C with a P-derived projection as a breaking compatibility decision; the frozen P candidate was only its implementation seed. Future replacements still compete under comparable conditions before another default change.

### Consequences

- Each candidate consumes the same typed semantic input and provider-independent fixtures.
- Candidate worktrees cannot change semantics, coverage, or answer keys to improve their score.
- The evaluation pins task prompts, model/agent versions, invocation budgets, token accounting, repetitions, and scoring.
- A presentation owns only its grammar, ordering, schemas, and output modes; none leaks into domain or application semantics.
- Historical candidate-C aliases were display-local and never replaced canonical references accepted by commands. The current `messages list` actor aliases have the same non-identity status: canonical account and message references remain directly visible and are accepted unchanged.
- An inconclusive experiment does not establish a benchmark winner. A default may still change through a separately recorded owner compatibility decision that explicitly accepts the migration, as occurred first for `cwk-task-projection/1` and then for its headerless subtraction.

### Enforcement

- Candidate C retains its status and evidence as the deterministic first stable baseline; the current headerless task projection, including flat message adjacency and the reviewed positional collections, receives active golden, semantic-answer, empty-result, hostile-output, canonical-reference, order, optional-suffix, and subtractive-field tests.
- A future presentation-competition work packet defines candidates and measurement before experimental implementations begin.
- Comparison reports identify each worktree/commit and record raw results, benchmark defects, and audit findings, not only a winner summary. Later compatibility decisions must not relabel inconclusive evidence as a win.

## Axiom 6: Discovery owns ambiguity; actions bind one exact target

Room, account, and message discovery may return candidates. A read or mutation
acts on declared opaque references passed unchanged from an invocable producer.
When the product deliberately owns exactly one local instance and exposes no
selection, the exact command may instead bind one catalog-declared fixed
`tool_local` target. That is an explicit singleton contract, not a default
chosen from available remote or credential state.

### Consequences

- Every public command is `utility`, `discover`, or `act`.
- Display names, positions, browser URLs, `rid...` strings, kind-prefixed text,
  and presentation-derived shorthand are not authorization identities.
- An action never case-folds, decodes, trims, reconstructs, or substitutes a Chatwork identifier.
- A Chatwork room action accepts only the discovered canonical room reference:
  a leading-zero-free positive decimal value of 1 through 32 digits, passed
  unchanged through `--room <room-id>`. A room name, `rid442804132`,
  `chatwork-room:442804132`, or Chatwork browser URL must first be resolved with
  `rooms list --query` and is rejected at the action boundary.
- A fixed-target action declares a stable kind, ID, scope, and description,
  produces/consumes no target reference, and becomes invalid when the product
  admits multiple possible targets.
- Required-reference chains lead back to an invocable producer.
- A caller-supplied account reference is not proof of credential identity.
  `rooms create --account` is an exact same-PAT precondition verified through
  `/me`; it is not a room-owner assignment because the provider create
  operation exposes no owner input.

### Enforcement

- Reference kinds live on structured catalog inputs and outputs.
- Fixed singleton targets live on a separate structured catalog field and are
  mechanically excluded from provider-owned or reference-selected flows.
- Whole-catalog graph tests prove producer/consumer reachability.
- Round-trip and negative tests preserve exact identity, exercise room-name
  discovery separately, and reject alternate room forms before adapter access.

## Axiom 7: Effects and uncertain outcomes are visible before repetition

An agent must know what a command can affect and whether a previous mutation may have happened before repeating or recovering.

### Consequences

- Every public operation declares `read`, `create`, or `write`; unknown effects fail closed.
- Message creation binds its room parent and declares notification impact.
- `messages send` accepts ordered exact recipient refs and an optional exact reply ref as typed preflight intent, separate from the room create scope; no caller-side notation parsing or exploratory read is required.
- When a message send has an unknown outcome, `messages reconcile-send` reuses the same room, body source, ordered recipients, and optional reply reference to compare the complete expanded payload against one latest bounded window. It returns exact candidates authored by the currently authenticated account, with an always-present `account_ref` and each candidate's canonical reference, `sender_ref`, author, and time; each candidate sender equals `account_ref`; outcome always remains unknown because a match cannot identify the invocation, and no match cannot prove non-delivery. This intentionally changes the earlier all-author candidate set.
- A consumed stdin stream cannot be replayed implicitly; the caller must supply the same bytes again. The current file contents and current reply author can also differ from the original send inputs.
- Send reconciliation never writes or retries. It is bounded to one exact reply read when needed, one current-account read, and one latest message-list read; its body matcher remains infrastructure-owned and one-use.
- Message update/deletion binds the existing message and declares destructive/notification impact.
- Authentication, permission, validation, and policy rejection cause zero downstream calls.
- Invite-link replacement names every provider-exposed setting: an explicit
  code or explicit regeneration, approval, and a nonempty description.
  Provider omission is never interpreted as "leave unchanged".
- An unclassified post-mutation outcome is non-retryable and points to read-only reconciliation.
- Credentials and unsafe provider causes remain inside infrastructure.

### Enforcement

- Domain intent and catalog mutation validation reject incomplete targets or impact.
- Fake-adapter tests prove pre-I/O rejection and safe post-I/O classification.
- Secret-canary and public-boundary checks cover output, errors, logs, and fixtures.

## Axiom 8: Claims remain executable

Command certainty, operational closure, semantic fidelity, understanding quality, token efficiency, output safety, and reference flow require repeatable evidence.

### Consequences and enforcement

- `scripts/check.sh` remains the canonical gate and `cli.Catalog` the public-command source of truth.
- Agent evaluations record discovery calls, external-processing steps, canonical references, semantic answers, recovery, tokens, and latency.
- Public examples use synthetic Chatwork-like data.
- `task check` decides implementation completion; higher-risk changes add the named security/public/release profiles.
- Catalog tests require Japanese user-facing prose while preserving stable
  machine identifiers, and repository localization checks cover the active
  Japanese entry documents and contribution templates.

## First complete implementation

The first complete implementation is bounded by the 32 REST operations linked from the official Chatwork documentation index on 2026-07-18. It must:

1. map every operation to at least one task-oriented public workflow and mechanically reject gaps or unreviewed extras;
2. implement single-account, process-local PAT authentication behind one secret-free binding, fixed-destination bounded transport, provider faults, and safe mutation intent;
3. support room discovery followed by a bounded recent-message result with explicit relationships, canonical references, hostile text, partial coverage, access limitation, and relation uncertainty;
4. implement every remaining operation with the same catalog, reference, authentication, effect, and recovery contracts;
5. establish candidate C deterministically as the first stable presentation and prove its semantic answer, bounds, trust framing, and canonical reference flow;
6. pass full, security, public, local-provider E2E, and agent-readiness gates without live credentials.

Within that finite implementation, every provider operation has one attempt;
metadata/read and non-upload requests use a 20-second timeout and uploads 60
seconds. The checked contract also caps success/error bodies at 8 MiB/64 KiB,
complete output at 16 MiB, aggregate lists at 10,000 items, documented endpoint
lists at 100 items, and upload at 5 MiB. Exact typed invocation suffices for
ordinary creates/updates; the reviewed access-changing and destructive sets
add exact `--confirm=access-change` and `--confirm=destructive`, respectively.
Uncertain mutation outcomes reconcile only through read-only tasks.

For read transport failures, a deadline is described as a timeout only when
`errors.Is(ctx.Err(), context.DeadlineExceeded)` or
`errors.Is(err, context.DeadlineExceeded)` proves that condition. The stable
`operation_canceled` kind, code, retryability, and exit status remain unchanged;
caller cancellation keeps cancellation wording, and ordinary I/O failures stay
`chatwork_unavailable`. A 403 may mention that the caller might not be a member
only for a task with a selected room target or scope, using fixed Japanese text
and never provider text. A known access-change or destructive confirmation
refusal states the exact required `--confirm` value without echoing argv or a
private cause.

Authentication has no selection step. `CWK_API_TOKEN` is the sole Chatwork
credential input and is read only from the command process. The CLI does not
accept it in argv, persist it in project or user configuration, expose an
authentication lifecycle command, or probe another credential source. Missing
or invalid token input fails before a provider task request. The fixed
production destination and secret-free ephemeral binding remain unchanged.
The required `rooms create --account` reference does not choose a credential:
after confirmation, infrastructure verifies that the already configured PAT's
`GET /me` account is an exact match before issuing one room-create request.
Mismatch or an uncompleted identity check makes zero room-create calls.

Future provider additions, OAuth grants and lifecycle commands, token
persistence, multiple accounts/profiles, GUI work, release publication,
alternative presentations, and further token optimization are outside this
completion boundary.

After that boundary closed, the owner accepted the breaking default-text
migration from `cwk-context-capsule/1` to `cwk-task-projection/1`, followed by a
second reviewed subtraction that removed the repeated schema/task preamble and
standalone provider-oriented coverage record. The semantic task result and
exact canonical references remain authoritative; historical headers,
dictionaries, aliases, ordering, and grammar are not compatibility promises of
the current default.

## Explicit non-goals

- Mirroring Chatwork endpoints mechanically or exposing transport vocabulary, even though the fixed public-operation snapshot must be covered by user-task workflows.
- Raw routes, arbitrary headers/bodies, or transport passthrough.
- Treating either the historical context-capsule grammar or the current task-projection grammar, field selection, or ordering as an axiom or domain model.
- Hidden fuzzy target selection.
- Silent truncation or fabricated relationships.
- Default lossy or model-generated summaries.
- Claiming structural escaping prevents semantic prompt injection.
- Enabling writes before authentication, impact, retry, and reconciliation policies are concrete.

## Thesis lifecycle

Record agent confusion, repeated pipelines, extra discovery calls, answer errors, token regressions, unsafe identifier conversions, and comparison bias in the active work packet. Revise an axiom before normalizing a workaround. Presentation evidence updates the selected contract; it does not retroactively turn one implementation pattern into an axiom.
