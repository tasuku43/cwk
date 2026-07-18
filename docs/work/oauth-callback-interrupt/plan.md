# Plan: Interruptible OAuth callback handoff

1. Race one bounded callback-line read against command-context cancellation
   using a buffered result channel. Return cancellation without waiting for a
   terminal implementation to release its blocked read.
2. Keep generic CLI readers caller-owned; the callback reader retains its
   existing bound and one-line contract.
3. Remove the `browser_opened` boolean from both prompt branches. Successful
   handoff prints only `callback_url: `; failure prints `authorization_url`
   followed by the same prompt.
4. Add unit coverage for context-driven input closure and exact prompt output,
   then reproduce SIGINT behavior through a PTY.
5. Update operating documentation, run the full gate, capture evidence, and
   commit.

## Risks and controls

- A generic reader cannot be force-canceled portably: the executable exits
  after returning the typed cancellation; tests release the blocking reader and
  verify the buffered worker terminates.
- A read error could mask cancellation: the OAuth manager checks the shared
  context and maps it to the existing cancellation fault.
