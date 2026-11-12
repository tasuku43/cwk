# Plan: Measure message-resource agent readiness

## Chosen approach

Reuse the historical presentation runner only as a reference for bounded Codex execution, strict JSONL event parsing, isolated workspace/PATH, tool restrictions, and transcript recording. Do not restore its task simulator as the execution layer because it bypasses the production CLI for task commands.

Add an evaluator-only child executable in the existing `tools/presentationeval` package. Build it with a dedicated `presentationeval` Go build tag and provide a tag-scoped constructor inside `internal/cli` that composes the real `CLI.RunContext` with the default catalog, the real argument parser, `chatworkcmd.Service`, authentication gate, and capsule renderer. Inject only a synthetic `chatworkcmd.Port` and synthetic `app/authn.Authenticator`; keep production `cli.New` and credential resolution untouched in ordinary builds. The fixture port should return the exact synthetic typed file/task sources and show results and append only task names, exact request refs, context/binding correlations, and write counts to a bounded private run journal.

The Codex process should run with a fresh workspace, PATH containing only the fixture executable, no credential-bearing environment, network disabled, unrelated tools/apps/plugins and multi-agent calls disabled, and a strict event parser that fails closed on unknown or non-`cwk` tool calls. The prompt stays natural and does not list command paths or expected resource refs. The answer key holds semantic facts; the runner independently checks the command graph, exact argv/ref reuse, result hashes, read/write journal, external-tool count, answer, budgets, and run identity.

Before any scored invocation, create a private source copy outside the target
tree, compare its digest with the source after copying, and build the fixture
from that copy. Keep the copy isolated throughout measurement and compare it
again after build and run. The runner must remove its private copy on all exits.
These comparisons catch observed drift but do not establish immutability or
atomic capture: enforce and independently verify a read-only snapshot boundary
before enabling the runner, and continue treating the result as inconclusive
until the other isolation, provenance, journal, and semantic checks are solved.

Budget the task separately from provider traffic. Permit a small fixed number of root/namespace/exact-help calls to discover the relevant CLI paths and at most three task invocations. The resource task should account for two read ports in files-before-room-tasks order; each exact show task accounts for one more read; total provider-port calls are four, write calls zero, and non-`cwk` external calls zero. Final budgets should be frozen before invoking the agent.

## Alternatives considered

- **Keep only the fixed sequence contract test.** It is useful semantic and application evidence but cannot satisfy actual discovery/execution or observed zero-post-processing claims.
- **Restore the historical simulator unchanged.** It manually parses task argv and calls `capsule.Render`; it is a presentation simulator, not the production CLI execution path required here.
- **Use the installed production `cwk`.** It is stale and would attempt to use the production PAT/API path; it is neither the current command surface nor safe for a synthetic no-network scenario.
- **Make a public runtime `NewWithFixture` API.** This would expand the product composition surface solely for an evaluator. Prefer a build-tagged, non-production constructor scoped to the evaluator build.

## Risks and controls

- The build-tagged constructor is still compiled code; add a negative test or release guard proving ordinary `cmd/cwk` builds do not include the fixture route.
- Use only synthetic IDs/text (`4201`, `9101`, `7711`, `8811`, `example.com` if a hostname is needed) and no credentials or external URLs.
- Ensure the runner records observations independently of the agent's final answer. Never accept model self-report as proof of tool or write counts.
- Do not treat historical runner source or the current Codex version as a compatible pin without updating and reviewing the protocol.
- Keep raw Codex JSONL and model output out of public docs unless reviewed for secrets and prompt-like fixture data; retain only the sanitized transcript required by the harness policy.

## Verification

1. Test the fixture executable against help and every selected command; compare output with current CLI/capsule contracts.
2. Test the runner parser with fake Codex JSONL for discovery, exact command success, wrong refs, extra tools, operators, writes, malformed/unknown events, cancellation, and bounds.
3. Run one fixed synthetic scenario using the reviewed Codex/model pin. Save the raw run in the bounded private run packet, sanitize the publishable transcript, then run the deterministic audit against the answer key.
4. Run focused `go test ./internal/cli ./tools/presentationeval` plus the relevant transport integration package; run repository gates only through the parent task owner.

## Candidate parent-owned fixture broker (not implemented)

The current draft places the invocation journal in the agent workspace, so it
cannot establish provenance. A possible replacement is a parent-owned local
broker. The agent-visible `cwk` wrapper sends bounded argv over one explicitly
allowed Unix socket; it never receives the journal path or a signing secret.
The broker accepts a bounded request, reserves the invocation budget, invokes
only the pinned synthetic CLI helper as a subprocess with those argv, and
records the actual execution, exit code, output hashes, provider journal and
failure before returning bounded bytes. It cannot execute an arbitrary shell
command or use a real provider. The broker's private artifacts and answer key
must be outside paths readable by the actual agent sandbox; `/tmp` alone is not
a privacy boundary in the observed macOS profile.

The socket and its protocol are untrusted: an agent or same-user process may
speak to it without the wrapper. This must not permit forged execution records,
new capabilities, silent requests, unbounded calls or unrecorded failures.
All accepted attempts count against a fixed broker budget and must be matched
against the agent trace and exact output bytes; unexpected socket requests or
trace/tool mismatch make the run inconclusive. Denials and incomplete requests
must likewise be observable rather than silently discarded. Client identity is
not inferred from socket access. Only after independent tests prove broker
lifecycle, locked and bounded accounting, concurrent and partial input,
malicious extra clients, timeouts, killed subprocesses and cleanup should it be
considered for wiring into the disabled runner.

This broker alone cannot prevent an agent from using other tools, reading host
files, escaping child processes, changing the target, or writing elsewhere.
The actual `codex exec` profile, all non-shell tools, immutable target identity,
private path access, trace provenance, process cleanup and semantic assessment
still require separate controls and tests. The public `run` remains disabled;
no claim of a scored run follows from implementing a broker prototype.

### Broker boundaries to validate before integration

- Reserve the budget at `Accept`, before reading input; use one request per
  accepted connection, at most seven accepted attempts, one active attempt, and
  at most one recorded over-budget rejection before closing the listener.
  Each request is a length-prefixed JSON object of at most 64 KiB with at most
  64 argv elements of at most 8 KiB each. Reject unknown/duplicate fields,
  malformed or trailing bytes and commands outside the fixed reviewed
  `readinessAllowedCommands` allowlist. Each read/write has a five-second
  deadline; helper execution has a separate 20-second deadline. These are
  candidate limits to be frozen before a run, not measured results.
- The parent records accepted connection attempts before reading, including
  malformed, rejected, disconnected, timed-out and incomplete attempts, and
  records response byte counts and write failures. A reservation without a
  completed outcome, any unexpected attempt, partial response, over-budget
  attempt or process cleanup failure makes the run inconclusive. Never retry a
  failed attempt automatically. The bounded record is parent-owned and must not
  silently drop events when full; failure to record also invalidates the run.
- Resolve the helper executable, working directory, environment and private
  instrumentation channel only from the reviewed parent configuration; never
  accept them from a client. Execute without a shell, with bounded stdout,
  stderr and instrumentation reads. The fixture's own provider/auth call data
  is child instrumentation and is not independent observation; the parent can
  validate and preserve it, while OS-level writes and process descendants need
  separate observation/containment. Do not claim otherwise.
- Socket access is not wrapper or agent authentication. Match every accepted
  attempt one-to-one and in order with exact argv and output bytes in the Codex
  trace, and fail closed on any mismatch. Even a match does not prove agent
  identity until tool isolation and trace provenance are independently verified.
  An exact command allowlist narrows discovery and can bias the evaluation; its
  effect on the scenario must be reviewed before it is treated as evidence of
  ordinary usage.
