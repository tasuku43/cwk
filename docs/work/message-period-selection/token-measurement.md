# Token measurement: bounded message period selection

This measurement uses one publishable 100-message typed source window. Thirty
messages precede 2026-07-17 in Tokyo, forty fall on that day, and thirty follow
it. The in-day records retain the semantic answer key for the archive-export
decision, canonical owner, and deadline. One explicit reply crosses the day
boundary so separate semantic tests can prove context behavior.

| Selection | Frozen input | SHA-256 | Bytes | Tokens |
| --- | --- | --- | ---: | ---: |
| unfiltered maximum-100 source | `active-message-period.unfiltered.txt` | `6ca26d66d684d209ab03cdf752afdca75c0f7331d2a21a0bab040ee7ae18a18b` | 12,876 | 2,940 |
| `--on 2026-07-17`, 40 anchors | `active-message-period.filtered.txt` | `43df27b03d54f7c05c6143cf41266b02b29e7853d1d8c342eab2a4e35877a883` | 5,718 | 1,381 |

## Result

The selected day removes 7,158 bytes (55.6%) and 1,559 tokens (53.0%) from
the same source result. It retains every answer-key fact, canonical references,
the provider `source-limit=100`, effective Tokyo day/bounds, and unresolved
boundary reply. The provider task-call count remains one and external
post-processing remains zero.

This is descriptive evidence for the motivating local-day workflow, not a
universal reduction guarantee. A date outside the provider-returned window is
still unavailable, and local selection does not reduce response bytes.

## Reproduction

The 2026-07-20 run loaded strict UTF-8 bytes and counted both frozen files in
one process using temporary measurement-only `tiktoken==0.13.0` with
`o200k_base`:

```sh
python - <<'PY'
from pathlib import Path
import hashlib
import tiktoken

enc = tiktoken.get_encoding("o200k_base")
base = Path("tools/presentationeval/testdata")
for name in (
    "active-message-period.unfiltered.txt",
    "active-message-period.filtered.txt",
):
    data = (base / name).read_bytes()
    text = data.decode("utf-8", errors="strict")
    print(name, hashlib.sha256(data).hexdigest(), len(data), len(enc.encode(text)))
PY
```

The tokenizer was installed only under `/private/tmp` for this measurement. It
is not a project dependency or runtime/test import.
