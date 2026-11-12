# Plan

Derive body segments during the same scan that yields relation facts. Preserve ordinary and code text verbatim, retain quoted text as a quote segment with its associated metadata, and remove only recognized addressing markers that are represented by typed facts. Preserve the exact source body separately. A malformed reviewed marker yields unknown body segments alongside the existing unknown relation state, never a partial result. Treat unsupported syntax as text rather than inventing facts. Do not render the candidate publicly until semantic fixtures, comparable measurement and an explicit decision justify it.

Verify exact source retention, order, multiple adjacent markers, quote/code containment, malformed tags, empty strings, hostile Unicode and deterministic behavior. Review implementation independently, run relevant focused tests and `task check`.
