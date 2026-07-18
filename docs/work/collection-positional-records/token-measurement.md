# Token measurement: positional file collection

The representative synthetic fixture contains six files in one bounded room
window. It preserves non-numeric provider order, multiple uploaders, canonical
message references and `absent`, zero and positive sizes, spaces, backslashes,
and a structurally hostile newline in a file name. Both projections are
generated from this same typed fixture.

| Projection | Frozen input | SHA-256 | Bytes | Tokens |
| --- | --- | --- | ---: | ---: |
| immediately preceding labeled list | `active-file-collection.labeled-before.txt` | `0f9a7cf788a23eef63f8ed92e10f4e00049b52ac5942e85af02a3c189c3c5a03` | 675 | 218 |
| fixed positional list | `active-file-collection.after.txt` | `6fdc69930165f5ee9e2f7a4a455f1fb66c0f4a629d7dc4b4f635794389929cd1` | 385 | 146 |

The fixed positional form removes 290 bytes (42.96%) and 72 tokens (33.03%).
This is evidence of reduced repetition, not an independent eligibility claim;
the semantic answer, canonical-reference reuse, order, absence, and hostile-text
tests remain mandatory.

## Reproduction

Both UTF-8 files were measured without normalization in one process using
`tiktoken==0.13.0` with `o200k_base`:

```sh
python - <<'PY'
from pathlib import Path
import hashlib
import tiktoken

enc = tiktoken.get_encoding("o200k_base")
for name in (
    "active-file-collection.labeled-before.txt",
    "active-file-collection.after.txt",
):
    path = Path("tools/presentationeval/testdata") / name
    data = path.read_bytes()
    text = data.decode("utf-8", errors="strict")
    print(name, hashlib.sha256(data).hexdigest(), len(data), len(enc.encode(text)))
PY
```

The tokenizer is a temporary measurement dependency and is not imported by
production code or repository tests.
