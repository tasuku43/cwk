# Work Context: Recover v0.1.0 Homebrew Formula publication

## Observed failure

- Release run `29685708150` published all five archives and
  `checksums.txt` from tag commit `ca3c97f`.
- Formula render and `ruby -c` succeeded.
- Strict Homebrew audit rejected the isolated tap copy because its filesystem
  mode was `0600` and recommended making it readable.
- Formula PR steps were skipped; no tap mutation occurred in the failed job.

## Constraint

The tag and GitHub Release are immutable. Recovery must use the exact published
checksums and Formula content, then propose only `Formula/cwk.rb` without
moving `v0.1.0` or replacing assets.

## Cause

The runner's restrictive creation mode reached the temporary tap through
`cp`. The audit boundary owned the copied file but did not normalize its mode
before Homebrew inspected it.

## Recovery evidence

- The five published archives each match the published `checksums.txt`; the
  checksum manifest names exactly those five archives.
- A Formula rendered from that published manifest passes `ruby -c` and a real
  `brew audit --strict` after the isolated tap copy is normalized to `0644`.
- The shared tap's test workflow accepts ordinary Formula-only pull requests
  for syntax checking. Automatic merge is intentionally limited to the
  `tasuku43-homebrew-tap-writer[bot]` identity, so this manual recovery PR will
  require an owner merge after its check passes.
