# Token measurement: bounded message sender selection

This measurement uses the same 14-message synthetic typed source window for
the unfiltered result, exact sender anchors only, and anchors plus direct typed
reply context. The fixture includes interleaved speakers, branches, deeper
reply descendants, typed To, raw `[rp]`/`[To]` canaries, hostile text, and
canonical next-action references.

| Selection | Frozen input | SHA-256 | Bytes | Tokens |
| --- | --- | --- | ---: | ---: |
| unfiltered source window | `active-message-sender-selection.unfiltered.txt` | `7ff794be2f6364851d5387d586ce1b9731f9b17aef6888ce39cbec5839693e78` | 1,231 | 443 |
| two exact sender anchors, `context=none` | `active-message-sender-selection.none.txt` | `d08c8e8bfaa4b6753e7c67e61d4602f10ac3cc4e202acfd6b3e6303c58a79fbe` | 605 | 208 |
| the same anchors, `context=replies` | `active-message-sender-selection.replies.txt` | `15f96623820a46b44266f827b3a802c1ad4542f41aefa189dabe52b99670fccf` | 957 | 341 |

All files are under `tools/presentationeval/testdata/` and are regenerated
mechanically from one source fixture through the application service and
current renderer. Golden tests reject drift.

## Result

- `context=none` removes 626 bytes (50.9%) and 235 tokens (53.0%) from the
  representative source window.
- `context=replies` removes 274 bytes (22.3%) and 102 tokens (23.0%) while
  retaining direct typed reply parents and children.

These are descriptive fixture results, not a universal reduction guarantee or
an independent eligibility claim. Semantic tests separately prove sender OR,
anchor/context provenance, source sequence gaps, one-hop reply bounds, no raw
text inference, terminal-safe framing, and exact canonical-reference reuse.
The readiness scenario fixes one provider task call and zero external
post-processing calls.

## Reproduction

The 2026-07-19 run loaded strict UTF-8 bytes and counted all three files in one
process using `tiktoken==0.13.0` with `o200k_base`:

```sh
python - <<'PY'
from pathlib import Path
import hashlib
import tiktoken

enc = tiktoken.get_encoding("o200k_base")
base = Path("tools/presentationeval/testdata")
for name in (
    "active-message-sender-selection.unfiltered.txt",
    "active-message-sender-selection.none.txt",
    "active-message-sender-selection.replies.txt",
):
    data = (base / name).read_bytes()
    text = data.decode("utf-8", errors="strict")
    print(name, hashlib.sha256(data).hexdigest(), len(data), len(enc.encode(text)))
PY
```

The tokenizer remains a temporary measurement-only dependency and is not
imported by production code or repository tests.
