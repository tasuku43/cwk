# Plan: Positional collection records

1. Add a shared collection prelude that emits the existing header, one trust
   declaration, and the collection's fixed schema.
2. Convert only the seven in-scope list item renderers to positional records;
   keep their single-record counterparts unchanged.
3. Add per-schema, empty/multi-item, hostile-text, provider-order, canonical
   round-trip, absent-file-message, and unchanged-show tests.
4. Freeze representative labeled and positional collection outputs and measure
   both with the pinned tokenizer used for message evidence.
5. Update catalog descriptions, README, theses/product/architecture/harness/
   readiness, and `$add-capability`.
6. Run `task check`, review the boundary, commit, and close this packet.
