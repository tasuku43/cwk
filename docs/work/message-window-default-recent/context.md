# Work Context: Recent message window by default

## Verified current behavior

- `messages list` declares optional `--window changes|recent`; its description
  names differential `changes` as the default.
- CLI request assembly starts from the zero value of `Request.ForceRecent`.
  Omission therefore leaves it false; explicit `recent` sets it true.
- Infrastructure maps `ForceRecent=true` to the documented `force=1` query and
  `latest_window` coverage. False sends no query and maps to
  `differential_window` coverage.
- Both paths make one bounded provider request with the existing 100-message
  source ceiling. The output already distinguishes `window=recent` from
  `window=changes`.
- Sender, limit, and direct-reply context are application-owned selection over
  the one provider result; they do not cross into the provider query.
- The completed `message-list-limit` packet deliberately excluded this default
  change. The owner's latest decision supersedes only that historical
  non-goal; its limit semantics and evidence remain valid.

## Constraints

- `cli.Catalog` remains the single source for allowed values and help.
- Omission must be resolved before authentication and provider I/O without a
  hidden extra request.
- Explicit `changes` must remain byte-for-byte distinguishable from omission
  at the typed request boundary.
- Presentation continues to derive `window=recent|changes` from typed coverage;
  it does not infer which flag was present.
- This is an intentional pre-1.0 default change and must be documented as
  such. No migration state is required because the input is invocation-local.
- Tests use synthetic ports and local request construction; no developer PAT or
  live Chatwork data is required.

## Thesis evidence

- The shortest invocation should represent the common supported outcome. For
  room-message reading, that outcome is current conversation understanding,
  not incremental polling state.
- Requiring agents to discover and repeat `--window recent` for ordinary reads
  adds a command-choice step and tokens without adding task information.
- Differential retrieval remains valuable for callers that deliberately ask
  for changes, so it stays explicit rather than being removed.

## Security and public-boundary notes

- Effect remains `read`; the exact room reference remains the sole target.
- PAT authentication, fixed destination, 20-second timeout, one attempt,
  response/output bounds, cancellation, and provider fault mapping do not
  change.
- No dependency, credential source, filesystem state, schema fixture, or
  external destination is added.
- The provider query remains limited to its already reviewed `force` field.

## Verification evidence

- The focused domain, application, infrastructure, CLI, and presentation
  packages pass together with synthetic fixtures.
- Relevant application, CLI, and active-presentation packages pass under the
  race detector; the changed packages also pass `go vet`.
- Exact human help renders `--window recent|changes` and names `recent` as the
  default. Scoped agent help projects the same catalog-owned values and
  description.
- Runtime tests prove omission and explicit `recent` produce
  `ForceRecent=true`, explicit `changes` produces false, and output preserves
  the corresponding typed window.
- Adapter request tests prove true emits only `force=1` and false emits no
  query. Coverage tests prove the respective `latest_window` and
  `differential_window` semantics with the unchanged 100-record incomplete
  bound.
- Active adjacency, sender-selection, and limit readiness scenarios now use
  the flag-free common invocation and retain one-command, one-provider-call,
  zero-post-processing budgets.
- Three independent reviews found no unresolved P1/P2 issue after restoring a
  completed historical readiness transcript to its original explicit command.
- `git diff --check` passes.

## Completion gate

The exact sandbox-exempt `task check` passed on 2026-07-19 with Go 1.26.5. It
included repository, architecture, contract, security, vulnerability, release,
and public-boundary checks plus the normal and race-enabled Go package suites.
The Go vulnerability scan reported zero called vulnerabilities. This approved
run supplied only the loopback-listener and official vulnerability-database
access that the managed sandbox had denied; it did not use live Chatwork data
or credentials.
