# Work Goal: Interruptible OAuth callback handoff

- Status: Complete
- Owner: Codex
- Target: Current implementation cycle
- Related ADRs: ADR 0002

## Outcome

While `cwk auth login` waits for the pasted callback URL, one Ctrl-C terminates
the command promptly with the existing structured cancellation result. A
successful browser handoff prints only the actionable callback prompt; browser
opener implementation state is not exposed to the user.

## Non-goals

- Changing the manual callback-paste model or registering an OS URI handler.
- Changing OAuth state, PKCE, exchange, credential storage, or API behavior.
- Adding a general interactive prompt framework.

## Acceptance criteria

- [x] One SIGINT during the production callback wait unblocks stdin and exits.
- [x] Cancellation still crosses the existing typed fault/exit boundary and
      never renders callback, code, state, or token material.
- [x] Successful automatic browser opening emits `callback_url: ` without a
      `browser_opened` diagnostic.
- [x] Browser-opening failure emits the bounded fallback authorization URL and
      the same callback prompt without a boolean diagnostic.
- [x] Unit and PTY regression tests cover the observed behavior.
- [x] `task check` passes.

## Completion definition

The executable behavior, CLI tests, catalog wording, README, and work-packet
evidence agree; the full repository gate passes; and the change is committed.
