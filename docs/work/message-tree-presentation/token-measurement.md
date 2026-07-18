# Message adjacency token measurement

This is a bounded before/after measurement for the active `messages list`
change. It is not a new presentation benchmark and does not alter the frozen
Competition 1 protocol or evidence.

## Measurement unit

- Semantic input: `messageAdjacencyFixture()` in
  `tools/presentationeval/active_message_adjacency.go`.
- Before text: the headerless repeated-message projection at commit `2d22298`,
  frozen as
  `tools/presentationeval/testdata/active-message-adjacency.before.txt`.
- After text: the first labeled flat provider-order adjacency projection from
  the same typed fixture, retained as historical evidence in
  `tools/presentationeval/testdata/active-message-adjacency.labeled-after.txt`.
- Included bytes: exact UTF-8 success stdout only, including the final newline.
- Excluded bytes: prompts, help, errors, shell output, and agent completion.

The fixture contains a reply chain, a branch, an interleaved root, To-only and
To-plus-reply messages, an out-of-window canonical parent, an unresolved parent
without an available reference, same-name accounts with distinct canonical
references, raw `[To]`/`[rp]` prose, and hostile structural text. Neither side
may change those semantic facts to improve its count.

## Token source

Count both frozen texts in one process with the same pinned `tiktoken` package
version and encoding. Record:

- `tiktoken` package version;
- encoding name;
- SHA-256 and byte count of each exact input;
- token count of each input;
- absolute and percentage delta.

The temporary tokenizer environment is measurement-only and is not a project
or production dependency. The recorded counts are encoding-specific input
counts, not Codex end-to-end usage and not a replacement for the semantic
quality checks.

The measurement command must load each file as bytes, require valid UTF-8,
decode once without newline normalization, call the same encoding's `encode`
method for each string, and print a machine-readable record. Re-running it over
files with the recorded hashes must reproduce the counts. Any version,
encoding, fixture, renderer, or input-hash change invalidates the comparison.

## Result record

The test-only reconstruction and the retained labeled output fix the two texts
from the same typed fixture. The frozen transport measurements are:

| Input | SHA-256 | Bytes | Tokens |
| --- | --- | ---: | ---: |
| repeated before | `58f181b933c9d25b27d6304c961dc1afaec4e3454e9206344430aafa6dc473d3` | 1,797 | 541 |
| flat adjacency after | `48c08d758d2c83e47c646b9090c4fa75ba268b3d6f934a81e29bc92758c43845` | 1,105 | 365 |

The 2026-07-19 measurement used `tiktoken==0.13.0` with the `o200k_base`
encoding for both inputs in one process. The flat adjacency projection used 176
fewer tokens, a 32.5% reduction (`365 / 541`), and 692 fewer UTF-8 bytes, a
38.5% reduction. The package was installed only into a temporary measurement
directory and is not a repository dependency.

The measurement process loaded each path with `Path.read_bytes()`, checked the
recorded SHA-256, decoded strict UTF-8 once, and evaluated
`len(tiktoken.get_encoding("o200k_base").encode(text))`. Acceptance depends on
semantic and canonical-reference accuracy; this delta is evidence rather than
a promotion threshold.
