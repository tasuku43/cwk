# Plan

1. Review this decision and propagate a narrowly bounded local link-to-reference discover contract before implementing.
2. Add one catalog-defined command `messages resolve-link --url <message-link>` with read effect, discover role, room/message reference producers, exact help and Japanese faults. It must participate in selection like other Chatwork task leaves, but is local and needs no PAT or external I/O.
3. Parse only the exact documented `https://www.chatwork.com/#!rid<room>-<message>` shape; reject userinfo, ports, alternate host/scheme/path, percent escapes, extra suffixes, whitespace, controls, and noncanonical or overlong reference values. Preserve both strings unchanged and never fetch the URL. It is syntactic extraction only, not target verification.
4. Keep the pure interpretation in a small domain/application boundary, CLI for parsing/wiring and safe output. Do not add a second registry or bypass catalog validation. `messages show` consumes the emitted refs unchanged.
5. Add reference-flow, no-auth/no-network, disabled-command, malformed/hostile inputs, exact output, help size and end-to-end tests. Obtain a fresh Luna implementation review and resolve findings before full gates.

## Reviewed boundary details

- The only accepted raw ASCII form is `https://www.chatwork.com/#!rid<R>-<M>` where each placeholder is 1..32 positive ASCII decimal digits without leading zero; the maximum accepted input is 95 bytes. No URL parsing normalization, percent decoding, extra fragment/query/path/port/userinfo, whitespace, controls or case variants are accepted.
- The local `EffectRead` discover leaf is configurable and has no PAT requirement, provider adapter, network call, API operation ID, or claimed change to the 32 external operations. An unsaved selection rejects dispatch; a saved disabled leaf returns `command_disabled`. No selection automatically enables either resolver or show.
- Catalog output declares `room_ref: chatwork-room`, `message_ref: chatwork-message` and a verification-not-performed state. The public output format remains text. Exact bytes round-trip to `messages show --room`/`--message`; the act command still rejects URLs. Help/recovery use only catalog paths and never interpolate the input.
- Root help directs link discovery to resolver help only when resolver is active; a separate show route is advertised only if show is active. Saved disabled leaves contribute no routes and unconfigured help remains control-only. Marked static fallback projects the full catalog with its existing notice and does not assert which leaves are enabled.
- Tests cover all selection combinations, bounded grammar and adversarial controls, no auth/network, catalog-derived reference flow and unchanged bytes, act URL rejection, Japanese failures, and actual root/namespace/exact help navigation. The fixed synthetic scenario is contract evidence until a qualifying real-agent run occurs.
