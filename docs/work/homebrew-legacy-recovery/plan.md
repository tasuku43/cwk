# Plan

Review a narrowly bounded manual recovery option against the existing architecture, security, and release contracts. Build from the exact published tag, verify the entire immutable release asset set, and allow only a mechanically checked removal of an exactly matching redundant version line. Strictly audit the resulting Formula before any tap credentials are used, pin its digest, then verify the same digest at the Formula-only proposal boundary. Do not grant a script arbitrary transformations, token-bearing source execution, release write access, or tag replacement. The evidence must retain both tagged-original and audited-result hashes and the exact one-line difference.

Alternative: publish a new stable tag containing the future renderer fix. That preserves the current workflow untouched, but it requires a fully reviewed new commit and all release gates; it does not repair the historical release's current Formula failure. Another alternative is to alter recovery to run an arbitrary newer renderer, which broadens the trusted input and is rejected without a separate review.

First obtain an independent Luna design review. If accepted, propagate the specific decision through durable contracts and add negative executable tests before preparing or proposing a Formula. Do not treat this work packet alone as authority to perform the mutation.
