# Context: Interruptible OAuth callback handoff

## Observed evidence

- A live `go run ./cmd/cwk auth login --client-id ...` opened the browser and
  printed `browser_opened: true` followed by `callback_url: `.
- Repeated Ctrl-C did not return control to the shell.
- `cmd/cwk` converts SIGINT/SIGTERM to context cancellation with
  `signal.NotifyContext`.
- The callback receiver then performs a synchronous byte-at-a-time read from
  `os.Stdin`; context cancellation alone cannot interrupt that blocked read.
- Because the signal notification remains installed until `run` returns,
  subsequent SIGINT signals are also consumed while the read remains blocked.

## Constraints

- Callback input remains stdin-only and bounded to one line.
- Generic CLI readers remain caller-owned and must not be closed implicitly.
- Closing a file does not portably interrupt every terminal read; Windows
  console `ReadConsole` is the critical counterexample.
- Normal completion must not close stdin before all command work is done.
- A canceled executable may leave one stdin read goroutine until process exit;
  tests must use a releasable reader and prove worker cleanup.
- Tests must use pipes or PTYs and synthetic callback data only.

## Completion evidence

- The callback receiver now races its existing bounded one-line read against
  the command context. Cancellation returns the stable
  `authentication_canceled` fault without closing caller-owned stdin.
- A releasable blocking-reader test proves callback waiting returns within one
  second with `ExitCanceled`, no stdout or secret material, and a worker that
  terminates after the test releases its reader. Focused race tests pass.
- Successful opener tests now require exactly the actionable `callback_url: `
  prompt. Fallback tests require `authorization_url` followed by that prompt;
  both reject any `browser_opened` diagnostic.
- A live PTY run of `go run ./cmd/cwk auth login` displayed `callback_url: `.
  One Ctrl-C immediately returned the structured canceled error with
  `code: authentication_canceled`; cwk reported exit status 11. No callback or
  authorization code was supplied or exchanged.
- On 2026-07-18 with Go 1.26.5, `task check` passed, including format,
  architecture, catalog, unit, race, vet, tidy, security, vulnerability,
  release, and public checks.
