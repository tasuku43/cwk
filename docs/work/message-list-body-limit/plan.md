# Plan: `messages list --max-body-bytes`

1. Add the optional flag to the catalog and exact help.
2. Extract and validate it in the existing CLI presentation argument boundary before authentication; keep it out of the application request.
3. Apply raw-byte UTF-8-prefix projection inside the `messages list` capsule branch after the assembled result is available.
4. Add explicit per-record truncation metadata and conditional schema declaration, without changing omitted-flag output.
5. Test Japanese/emoji boundaries, zero, escaped controls/backslashes/bidi, original lengths, existing selection/reply/context semantics, invalid pre-auth input, relation-context coverage, combined overall budget, and default byte identity.
6. Update thesis, product, architecture, security, harness, capability skill, readiness contract, and a semantic fixture where required.

## Risks

- A byte cap applied after escaping would not match the requested meaning; tests must assert raw-byte boundaries and escaped final output.
- Applying the cap before typed filtering or relation inference could hide search matches or create fabricated relationship/completeness changes; it must remain presentation-only.
- A change to schema help or output metadata can exceed the exact-help ceiling; catalog validation and help tests cover this.
