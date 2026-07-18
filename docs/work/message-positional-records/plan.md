# Plan: Positional message records

1. Change only the `messages list` schema and record renderer.
2. Update renderer/golden/hostile/deep-chain/canonical-round-trip tests to parse
   the fixed positions and reject the removed labels.
3. Update active semantic/readiness fixtures and the after-measurement golden;
   leave the before baseline unchanged.
4. Re-run the pinned tokenizer and record hashes, bytes, tokens, and delta.
5. Update README and governing presentation contracts, run `task check`, review,
   commit, and close this packet.

The semantic result, catalog field meanings, provider adapter, and application
relation resolver remain unchanged.
