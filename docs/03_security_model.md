# Security Model

This document defines the minimum security reasoning expected from Chatwork CLI and every derived project. It does not claim that the base scaffold makes project-specific integrations secure. Its purpose is to make assets, effects, trust boundaries, and enforcement visible before implementation spreads them across commands.

## Security objective

The CLI must not perform a side effect broader than the validated user task and declared `operation.Intent` and `operation.Impact`. Invalid, unknown, or inconsistent operation state fails before external I/O. Credentials, confidential identifiers, and private repository history must not enter public source or artifacts.

## Assets in the base template

- Integrity of the user's local system and files.
- Confidentiality of arguments, environment values, and future credentials.
- Integrity of command selection, effect classification, and target binding.
- Integrity of source, dependencies, generated files, and release artifacts.
- Confidentiality of information excluded by the public-repository policy.
- Availability of a deterministic local verification path.

A derived project must add every remote account, object type, local state file, credential, cache, log, subprocess, network service, and published artifact it introduces.

## Actors and assumptions

The model distinguishes:

- **User:** invokes the CLI and may provide malformed input accidentally.
- **Coding agent:** can edit files and execute commands within its granted environment; it is not proof of human authorization.
- **Contributor:** can propose source changes but cannot redefine policy without review.
- **External system:** returns untrusted data and may fail, drift, rate-limit, or be compromised.
- **Attacker:** may control input, remote content, a dependency, a copied file, or a proposed patch.
- **Release owner:** is trusted to follow the reviewed release procedure, not to bypass it manually.

The base model does not assume that a TTY, parent process, environment variable, or successful process launch proves human intent.

## Trust boundaries

```text
untrusted argv / environment / files
                |
                v
       CLI parsing and validation
                |
                v
      application task boundary
                |
                v
 Effect + Intent + TargetRef + Impact validation
                |
                v
     controlled side-effect boundary
                |
      +---------+----------+
      |         |          |
 filesystem  process    network / platform service
```

Remote responses cross the boundary in the opposite direction. They remain untrusted data after parsing, bounding, and safe structural rendering; those steps do not turn remote prose into trusted instructions.

## Effect and intent policy

| Effect | Required declaration | Base policy |
|---|---|---|
| `Read` | Stable task and bounded source | May execute without mutation authorization; still validates input, destination, and output bounds |
| `Create` | Parent or creation scope in `TargetRef`; complete base `Impact` | Fails closed without complete intent and impact; derived project decides confirmation or human authorization |
| `Write` | Existing target in `TargetRef`; complete base `Impact` | Fails closed without complete intent and impact; product-specific consequences require additional typed detail |
| `Unknown` | None | Never executable |

HTTP methods, SDK names, or command verbs do not determine the effect. A query sent with `POST` may be a read; writing a local output file is a side effect even when no network request occurs.

The base `Impact` always declares cardinality and whether the operation sends notifications, changes access, or is destructive. When those dimensions cannot explain policy, extend intent with a typed domain model before exposing the command. Examples include:

- multiple targets;
- irreversible deletion;
- access or ownership changes;
- messages or notifications sent to other people;
- execution of user-provided code;
- publication outside the user's account or machine.

Do not encode these distinctions only in a human-readable confirmation string.

## Opaque reference boundary

Discovery output may contain labels controlled by an external system, but action identity comes from the declared opaque reference field. Pass the emitted ID unchanged into the consuming argument.

- Validate the documented shape and length before adapter execution.
- Do not authorize against a display name, row position, copied URL, or reconstructed resource path.
- Do not case-fold, decode, unescape, trim, or otherwise normalize an opaque ID unless its domain contract explicitly defines that operation.
- Bind mutation `TargetRef` to the same validated ID accepted by the action command.
- Keep remote labels out of authorization identity and sanitize them separately for display.
- Reject control/format runes and Unicode line or paragraph separators at opaque transport boundaries where they have no valid protocol role. Validation rejects; it never silently rewrites an ID, cursor, target part, or idempotency key.

The only reference-free action target is a catalog-declared fixed
`tool_local` singleton with a stable kind, ID, and description. It cannot be a
provider object, cannot be selected from credential presence, and cannot be
used once multiple possible instances exist. Catalog tests keep this path
disjoint from opaque-reference flows and bind its mutation target explicitly.

The internal sample test fixture accepts only `smp_` followed by twelve lowercase hexadecimal characters. Its negative tests remain as generic boundary evidence, but the fixture is not a public capability. Public Chatwork commands apply the same exact-byte rule to their typed canonical references.

### Presentation-derived identity

A presentation candidate may introduce shorthand, positions, labels, grouping, or other derived display values. None is authorization identity by default. Public action inputs, policy, logs, and infrastructure use validated canonical references. If a later presentation proposes reusable shorthand, it requires a separate typed contract defining scope, lifetime, collision handling, and exact resolution; the presentation experiment itself cannot grant that meaning.

## Chatwork notation and relationship trust

Chatwork notation is untrusted provider data with a documented syntax, not executable instruction. Infrastructure parses only reviewed bounded forms into typed facts.

- To establishes recipient identity but not a reply edge.
- Reply notation establishes a provider-declared room/message relation only after identifier validation.
- Quote metadata remains a quote relation; missing message identity is not reconstructed from author, timestamp, or text.
- Malformed, nested, contradictory, or unsupported recognized notation keeps the bounded external body, drops every partial relation fact for that message, and marks the whole relation set unknown; malformed wire identity, UTF-8, JSON, or response bounds still fail closed. One malformed body never discards otherwise valid list records.
- Message access-limitation headers are untrusted protocol evidence. Only the official `true` value on its documented list/exact-message status is promoted; malformed or contradictory combinations fail closed, and the provider summary is neither exposed nor parsed into policy.
- Display names embedded beside tags cannot override account identifiers.
- Proximity, layout-looking text, prompt-like prose, and copied tags inside quoted/code-like content do not create authorization or mutation targets.

The typed task result distinguishes explicit resolved, explicit unresolved, and absent relations before presentation. Every candidate must preserve that answer; a clearer or smaller display is not allowed to strengthen a relation.

## Controlled execution boundary

All filesystem writes, subprocess execution, credential access, network calls, and platform services must have a bounded construction path. A command or use case must not receive an unrestricted client merely because it is convenient.

For a mutation, the boundary performs this order:

1. Snapshot and validate the operation input.
2. Validate `Effect`, `Intent`, `TargetRef`, `Impact`, and concrete destination consistency.
3. Apply project-specific policy such as dry-run, human authorization, or confirmation.
4. Execute the side effect once.
5. Return a bounded result and audit event if the product requires one.

A rejection at steps 1–3 must result in zero mutation attempts. Tests use fakes or counters to prove this order.

Cancellation is interpreted relative to that boundary. Before step 4, the invoker can prove zero attempts and may expose retryable `operation_canceled`. After step 4 begins, a valid structured adapter fault remains authoritative and is copied without its private cause. A raw error or cancellation cannot prove rollback; it becomes non-retryable `unclassified_mutation_outcome`, whose catalog recovery must be a read-only reconciliation command. A confirmed nil result is not replaced by a later context cancellation.

Application ports treat both a nil interface and an interface containing a typed nil pointer as missing configuration. The shared port check prevents a dependency-wiring mistake from becoming a panic after validation or policy approval.

### Chatwork mutation policy

The first Chatwork implementation treats a fully specified exact invocation as sufficient confirmation for ordinary creates and updates. It does not add a generic “confirm every write” prompt that an agent could satisfy mechanically without understanding the operation.

Two higher-impact classes fail before provider I/O unless argv contains the exact typed confirmation:

- `--confirm=access-change` for room creation, room-member replacement, invite-link creation/update, and incoming-contact-request acceptance;
- `--confirm=destructive` for room leave/delete, message deletion, invite-link deletion, and incoming-contact-request rejection.

The catalog binds each confirmation to the corresponding operation impact; a confirmation for one class does not satisfy the other, persist across invocations, or prove human approval. Missing, duplicated, misspelled, or inapplicable confirmation is invalid input or policy rejection with zero mutation attempts.

Room creation never treats caller-provided `--account` as authentication
evidence or room ownership. The same private PAT binding must first return that
exact account from `/me`; mismatch and identity-probe failure remove the
provisional binding and make zero room-create calls. Neither the requested nor
actual identity is copied into a public mismatch detail. The provider room
form contains no owner/account field. Infrastructure revalidates the stored
verified account immediately before request construction, so a generic or
mismatched binding cannot become an alternate authorization path. Official
room-name and icon-preset validation also completes before authentication.

Invite-link update never sends an empty or partial form. Exact code versus
explicit regeneration, approval, and nonempty description are validated before
authentication and again at form construction. This prevents omission from
silently regenerating a code, applying the documented approval default, or
depending on the undocumented description-omission behavior. No URL parsing,
prior-value merge, or empty-description clear is used to invent provider
semantics.

No Chatwork operation is retried automatically in this implementation. After an uncertain mutation result, the only recovery is the exact read-only reconciliation command declared by that mutation's catalog contract. A reconciliation may report absence or ambiguity; it never converts uncertainty into permission to repeat the write.

## Credentials and secrets

The core contains a secret-free authentication contract, an ephemeral
non-serialized session binding, and an application gate. The current Chatwork
adapter makes the PAT source concrete below. Any future credential method or
storage mechanism must document:

- credential issuer and scope;
- acquisition and any refresh flow;
- storage location and operating-system protection;
- how secrets are kept out of process arguments, logs, errors, history, and generated files;
- revocation and expiration behavior;
- tests and scans that fail on unsafe handling.

Secrets must not cross from infrastructure into domain or application values and must not be accepted through command-line arguments when a safer channel is available. Do not persist tokens in plaintext configuration or test real credentials in CI. Read [Authentication](07_authentication.md) before changing the current PAT boundary; a future OAuth proposal must also start from [ADR 0001](decisions/0001-oauth-library-boundary.md) and a new accepted product/security decision.

The first Chatwork implementation supports one account per command process
through the API token in `CWK_API_TOKEN`. There is no method selector or second
credential source. Missing or invalid token input fails before a provider task
request.

Environment delivery is an explicit non-persistent automation trade-off, not a
claim that environment variables are a secure credential store. The CLI does
not accept the token in argv, configuration, stdin data payloads, or output;
infrastructure reads it once, keeps it in memory behind an ephemeral binding,
and redacts it from every error and test diagnostic. The tool exposes no
credential login, status, logout, callback, profile, or persistence surface.

The token may be sent only to the exact HTTPS origin
`https://api.chatwork.com` under the `/v2` base path as `x-chatworktoken`.
Redirect following is disabled for credential-bearing requests. A different
base URL can be injected only through internal test construction with synthetic
tokens and a local server; there is no public flag or environment override for
the production destination.

## Filesystem, process, and network policy

### Filesystem

- Validate output paths and overwrite behavior before writing.
- Use restrictive permissions for confidential data.
- Define atomicity, symbolic-link, and partial-write behavior where relevant.
- Keep caches, state, configuration, and audit data separate.

#### Command-selection preference

The persisted command allowlist reduces the agent's attention surface; it is
not an authorization, access-control, sandbox, or security boundary. The same
local principal that runs `cwk` may edit or remove the file, invoke `config`,
or install another binary. Removing the file intentionally restores the
documented missing-state behavior in which all current configurable Chatwork
commands are visible. Hidden commands must therefore never be used as evidence
that an operation is forbidden or inaccessible.

`help`, `doctor`, `version`, and the single exact `config` write remain
catalog-declared always-on so the local view is diagnosable and reversible.
Disabled paths fail as
`unknown_command` before PAT resolution or provider I/O, but re-enabling a path
grants no Chatwork authority:
PAT authentication, provider permissions, canonical-reference validation,
typed intent and impact, and access-change/destructive confirmation still run
unchanged. An attached TTY, raw mode, cursor movement, Space, and Enter prove no
human identity or authorization. Enter confirms only the exact local preference
replacement admitted by the fixed-target mutation policy; it cannot approve a
Chatwork action. The non-security contract remains explicit in scoped help and
documentation without a repeated schema/purpose header in the selector.

The interactive screen retains textual `[read]`, `[create]`, and `[write]`
badges. Cyan, yellow, and magenta are redundant visual cues, never the source
of effect truth; red is not used because effect is not a destructive-severity
classification. ANSI sequences are fixed CLI structure excluded from display
width, while command text is visibly escaped before it can enter a row. A
catalog string cannot introduce a terminal control sequence or replace an
effect badge.
If the terminal cannot display the complete current command path together with
its checkbox and effect, movement, Space, and Enter are ignored and the screen
requests a resize. Only q/Ctrl-C/closure remains effective, so clipped or absent
identity cannot authorize an unseen preference replacement.

The non-secret preference is stored at
`${XDG_CONFIG_HOME:-$HOME/.config}/cwk/command-selection.json` on macOS and
Linux and `%AppData%\\cwk\\command-selection.json` on Windows. It is separate
from PAT state and from ADR 0003's retired OAuth `cwk/config.json`, and its
schema accepts only bounded exact command paths. It must not contain a token,
provider response, personal data, or copied command metadata. On macOS and
Linux, infrastructure permits the configuration home itself to be an existing
symbolic-link alias: it resolves the complete alias once to an absolute
directory and uses only that result for the invocation. It still rejects
symbolic-link or special-file `cwk` directories and preference files and
replaces from a restricted same-directory temporary file. This distinction
supports user-controlled dotfiles namespaces without letting a mutable owned
leaf redirect replacement. Unix uses rename plus opened-directory sync;
Windows replace-existing has no portable atomicity or durability guarantee.
Malformed, unsafe, or unavailable state never silently activates the complete
Chatwork surface or makes a failed load executable. Only malformed serialized
content enters the explicit `config` repair flow: it starts from a clearly
marked replacement draft and changes nothing until Enter. Unsafe modes/objects
refuse the selector and require external filesystem repair; unavailable storage
must have access restored. The always-on read-only `doctor` reports invalid,
unsafe, or unavailable command-selection state without mutating it.

Terminal control is also phase-bounded. The infrastructure-only
`golang.org/x/term` adapter requires interactive stdin and stdout, owns raw
mode, alternate-screen entry, size lookup, and idempotent restoration, and
exposes no selection or authority policy. A non-TTY invocation fails before
persistence. Up/Down and Space modify only an in-memory draft. q, Escape, EOF,
terminal closure, or cancellation before Enter restores the terminal on the
bounded graceful path and leaves the prior preference unchanged. Enter
validates the complete active view and fixed-target request, restores the
terminal, and only then invokes the save action; restoration failure makes zero
save calls. After replacement is attempted, a raw failure is uncertain and
reconciles through `doctor`; confirmed success is not replaced by late
cancellation. Process termination that cannot run cleanup is not claimed to
restore terminal modes, but it still cannot persist a draft whose Enter save
boundary was never crossed.

The uncertain write fault exposes the candidate versioned SHA-256 fingerprint,
and `doctor` exposes the actual fingerprint of the ordered canonical enabled
paths together with its source. These bounded values and counts contain command
metadata only, never PAT or provider data, and let the read-only diagnostic
distinguish the state present after an uncertain write. Confirmed success does
not repeat this recovery identifier; it reports only the human-readable display
summary and any nonzero cleanup. Profiles from the preceding selector may
contain `doctor` or `version`; those two exact legacy entries are ignored for
active-view construction and removed on the next Enter save. No other always-on
or unknown path receives that migration exception.

### Processes

- Prefer in-process APIs.
- If a subprocess is required, allowlist the executable and fixed argument structure.
- Keep secrets out of argv and inherited environment unless explicitly justified.
- Bound output, time, and cancellation behavior.

### Network

- Declare allowed destinations and redirect behavior.
- Use timeouts, cancellation, and response-size limits.
- Validate protocol and host before sending credentials or user data.
- Treat remote names and content as unsafe terminal and machine context.

For the first Chatwork implementation, metadata and ordinary provider requests have a 20-second timeout, file uploads have a 60-second timeout, and every logical operation has one transport attempt. Successful response bodies are capped at 8 MiB and provider error bodies at 64 KiB. The complete rendered result is capped at 16 MiB, aggregate lists at 10,000 items, and uploads at 5 MiB. The five provider-documented 100-item list operations retain their lower bound. Exceeding any limit fails closed without partial successful output; user configuration cannot raise a ceiling.

Message `--start-index` and `--count` are bounded local selection inputs, not
network controls. Values outside 1..100, duplicates, and malformed values fail
before authentication or I/O. Infrastructure rejects either field if it crosses
the application port, and a provider result above `source-limit=100` fails
before local selection can conceal it. A derived next start index carries no
credential, opaque provider state, or snapshot-stability claim.

Rate-limit headers and error envelopes are untrusted external data. Chatwork
reset timing is accepted only as one bounded strict-decimal
`x-ratelimit-reset` Unix timestamp in the official five-minute window;
missing, duplicate, malformed, stale, or distant values produce unknown timing
instead of a guessed delay. `Retry-After` is not authoritative for this
provider. The adapter may compare the bounded message/task posting error body
with the one official room-limit sentence to select 10 seconds, but never
publishes that body. A delay carried by a non-retryable mutation fault is
display-only evidence: it cannot authorize automatic or agent-inferred replay,
and the one-attempt mutation contract still applies.

Machine-readable policy belongs in `.harness/project.json` or a project-specific typed manifest and is checked by `tools/repoguard` or a dedicated contract test.

## Output and terminal safety

Trusted CLI-authored human prose is Japanese. This does not relax structural
validation: Japanese catalog and TUI strings are validated and rendered as CLI
structure, while stable machine tokens remain ASCII. Localization must never
rewrite an opaque reference, credential, URL, provider response field, or
external Chatwork text. Translation is not sanitization and does not make
external text trustworthy.

Remote or file-derived text may contain terminal controls, format characters, Unicode line/paragraph separators, existing backslash escape-looking sequences, JSON-looking fragments, prompt-like prose, or excessive data. The template's visible projection escapes backslash first, then control/format runes and U+2028/U+2029. Therefore an actual newline remains distinguishable from the two input characters `\n`; JSON encoding applies its own structural escaping afterward. TSV delimiters and record newlines are added only by the renderer.

This projection protects terminal and TSV/JSON structure. It does **not** detect intent in printable text, remove prompt-like content, or prove that a language model will ignore it. Printable text such as `SYSTEM ...` or JSON-looking content remains semantically untrusted data. Agent consumers must keep data fields separate from instructions and apply their own trust policy; the CLI cannot claim semantic prompt-injection prevention.

Exact-command agent help publishes this boundary in `io_contract`: `external_text_trust` is `untrusted_data`, `external_text_projection` is `visible_escape`, and `opaque_reference_policy` is `validated_exact_bytes`. Compact root and namespace indexes contain no I/O contract or external text.

A derived project must decide:

- which output modes are stable;
- how control characters are escaped or rejected;
- maximum stdout and stderr budgets;
- whether partial output is ever allowed;
- how secrets and confidential fields are redacted;
- which stream carries errors versus data.

Do not let presentation sanitization change the identity used for authorization. Authorization uses validated domain values; display labels/content use a visible projection. Opaque references bypass that projection and retain the exact validated value.

For every presentation candidate, structure is CLI-authored and message text is external data. Its framing must prevent provider text from injecting candidate-specific records, fields, hierarchy, or completeness signals. Optimization must not remove the `external_text_trust: untrusted_data` meaning already published by exact-command agent help. Hostile-text fixtures are shared across candidates so a format cannot improve token score by weakening structural safety.

Candidate C's historical context capsule used local aliases only inside one output document. They were never accepted by command parsers, mutation policy, audit identity, or infrastructure. The current `messages list` likewise uses actor aliases only to factor repeated display data inside one document. Every actor dictionary entry exposes its validated canonical account reference, every message record exposes its validated canonical message reference, and command inputs accept only those canonical values unchanged. Aliases never cross into mutation policy, audit identity, application, or infrastructure.

The current projection is also subtractive at the trust boundary: only catalog-declared task fields, exact canonical references, task-relevant bounds/completeness/uncertainty, and external-text trust framing may cross into the text contract. Provider/wire extras, display conveniences, and non-contract defaults are not promoted merely because they are available. Raw Chatwork notation may remain inside a declared untrusted message body, but presentation neither exposes it as separate semantic structure nor reparses it to create relationships. Removing fields must never remove the framing that distinguishes external data from CLI-authored structure.

## Supply-chain boundary

- Add a dependency only with a documented purpose and license review.
- Pin CI actions and security tools according to repository policy.
- Verify module integrity and known vulnerabilities in the security profile.
- Review generated changes and prove generation is reproducible.
- Build releases from reviewed source through the documented workflow.
- Decide signing and provenance explicitly; absence of signing is also a release-model decision.

### Shared Homebrew tap publication

Stable releases propose the already rendered, checksum-pinned, and strictly
audited `Formula/cwk.rb` to the public `tasuku43/homebrew-tap` repository. The
workflow never pushes directly to the tap's default branch and never stages a
Formula wildcard. Prereleases do not cross this write boundary.

The macOS Formula audit job has no App credential and uploads the strictly
audited Formula as a workflow artifact. A fresh Ubuntu publish job checks out
no source repository and executes no checked-out tagged source or Formula
content. It downloads that one-file artifact, validates its exact regular-file
identity as data, and only then exchanges GitHub-hosted repository secrets
`HOMEBREW_APP_ID` and `HOMEBREW_APP_KEY` for a short-lived installation token.
Token creation names owner `tasuku43` and only repository
`homebrew-tap`, and explicitly requests only Contents write and Pull requests
write as write-capable token permissions. Before a stable tag, the App
installation must be manually confirmed as restricted to that repository with
matching maximum permissions and absent from `cwk` and unrelated repositories.
The audit job's source-repository `GITHUB_TOKEN` stays Contents-read-only. The
token-bearing Formula publish job has no source-repository `GITHUB_TOKEN`
permissions; the separate GitHub Release publish job retains Contents write
only for release creation.

Post-publication Formula recovery is a separate, stable-tag-only entry to the
same workflow. Its preflight resolves the quoted input through the release-tag
validator, runs the canonical full gate at that immutable revision, and grants
the recovery job only Contents read. Recovery accepts only an existing
non-draft, non-prerelease Release whose six exact filenames and five checksums
verify. It uploads only the verified checksum file to the credential-free
Formula audit job; it has no Release write permission and does not execute
checked-out source. The later tap write still occurs only in the existing
fresh runner with the same App installation token restrictions.

Both source and tap checkouts disable persisted credentials and occur on
different runners. The installation token is supplied explicitly only to the
tap checkout and pinned pull-request
action, and it must not appear in Formula content, archives, logs, source, or
Git configuration left by checkout. The workflow and both Formula jobs admit
only their reviewed fields, so workflow- or job-level `env` and `defaults`
cannot inject runtime options into the token action. Before copying, the
publish job rejects a symbolic-link `Formula` directory, a symbolic-link
`Formula/cwk.rb`, and any existing non-regular target. The pull request targets
`main`, uses the tap automation's reviewed branch/title prefix, and changes
only `Formula/cwk.rb`. Tap merge remains a separate review/automation boundary;
a successful release job does not claim that the Formula has already merged.

Secret provisioning and App-installation review occur in GitHub settings
before the first stable tag. Secret values are never copied into a work packet
or release note. See [ADR 0004](decisions/0004-shared-homebrew-tap.md).

## Required negative tests

Every new side-effect class should include tests for:

- unknown or missing effect;
- incomplete or mismatched target;
- malformed input before adapter invocation;
- alternate, transformed, or ambiguous reference forms before adapter invocation;
- policy or authorization rejection with zero side-effect attempts;
- nil and typed-nil external ports with zero downstream calls;
- cancellation and timeout;
- structured versus unclassified post-mutation outcomes, including a deadline cause that must not erase the adapter's stable classification;
- remote error without secret leakage;
- authentication or scope mismatch with zero downstream calls;
- incomplete pagination with no partial successful output;
- unsafe mutation retry or missing idempotency declaration;
- timeout or attempt configuration above the project's declared maximum;
- keyed retry that changes key within one logical operation or reuses a key for a different target or payload;
- oversized or hostile output;
- raw ESC/newline/bidi/zero-width/U+2028/U+2029 characters, existing backslashes, JSON-looking text, and prompt-like printable data across TSV, JSON, and stderr;
- actual control characters remaining distinguishable from pre-existing visible escape-looking text;
- repeated invocation when authorization must not be reused.
- disabled-command invocation producing the ordinary unknown-command result
  before credential resolution or provider I/O;
- malformed, oversized, duplicate-key, trailing-value, symbolic-link, or
  special-file command-selection state without a silent all-enabled fallback;
- non-TTY rejection; fragmented CSI/SS3 keys; bounded terminal frames; and
  restoration across success, quit, EOF, cancellation, validation failure, and
  terminal failure;
- canceled, EOF, invalid, dependency-incomplete, or pre-Enter selection leaving
  the prior preference unchanged, and restoration failure making zero save
  calls;
- uncertain command-selection writes reconciling through the exact read-only
  `doctor` fingerprint rather than another mutation;
- legacy `doctor`/`version` selection entries normalizing without admitting
  another always-on path;
- textual effect badges remaining present without ANSI color, with hostile
  command text unable to create a control sequence;
- a re-enabled Chatwork mutation still requiring its original PAT, exact target,
  and typed confirmation policy.

## Derived-project threat-model pass

Before implementing a real integration, add a table covering:

| Task | Assets | Inputs | Effect | Target or impact | External boundary | Authorization | Stored data | Enforcement |
|---|---|---|---|---|---|---|---|---|

Then document credible abuse cases and limits. If a risk is accepted rather than eliminated, record the rationale in an ADR and state what evidence would trigger reconsideration.

## Limits

The template cannot protect a fully compromised developer machine, maintainer account, CI platform, or external service. It cannot infer whether copied source is legally publishable. It narrows accidental bypasses and makes review evidence repeatable; it does not replace independent review for high-impact systems.
