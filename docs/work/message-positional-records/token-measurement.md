# Positional message token measurement

This bounded measurement uses the same presentation-independent fixture,
baseline, tokenizer package, encoding, and counting procedure as the accepted
flat-adjacency change.

## Inputs

| Input | File | SHA-256 | Bytes | Tokens |
| --- | --- | --- | ---: | ---: |
| repeated pre-adjacency baseline | `active-message-adjacency.before.txt` | `58f181b933c9d25b27d6304c961dc1afaec4e3454e9206344430aafa6dc473d3` | 1,797 | 541 |
| labeled flat adjacency | `active-message-adjacency.labeled-after.txt` | `48c08d758d2c83e47c646b9090c4fa75ba268b3d6f934a81e29bc92758c43845` | 1,105 | 365 |
| positional flat adjacency | `active-message-adjacency.after.txt` | `0d4273c54d4361c177e419b939dbbed4d3d543aa285aeba6a1e31fe0bfef1c4e` | 909 | 330 |

All files are under `tools/presentationeval/testdata/`. The current renderer and
the active semantic fixture mechanically reproduce the positional after file;
the labeled file retains the superseded intermediate public contract as
measurement evidence.

## Token source and result

The 2026-07-19 run loaded strict UTF-8 bytes and counted both compared texts in
one process using `tiktoken==0.13.0` with `o200k_base`.

- Versus the repeated baseline: 541 to 330 tokens, 211 fewer (39.0%); 1,797 to
  909 bytes, 888 fewer (49.4%).
- Versus the labeled flat adjacency: 365 to 330 tokens, 35 fewer (9.6%); 1,105
  to 909 bytes, 196 fewer (17.7%).

The package remained a temporary measurement-only dependency. These counts are
encoding-specific evidence, not an understanding-quality threshold. Semantic,
escape, and canonical-reference tests remain eligibility requirements.
