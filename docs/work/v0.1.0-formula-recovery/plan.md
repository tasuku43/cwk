# Work Plan: Recover v0.1.0 Homebrew Formula publication

- Status: Accepted
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Normalize only the isolated tap copy to `0644` immediately after copying it.
Make the fake Homebrew boundary inspect that exact mode from a `0600` source
fixture. After local and repository verification, render from the published
`v0.1.0` checksum asset, run the real strict audit, and open a Formula-only
shared-tap pull request.

## Recovery boundary

Do not move the tag, recreate the Release, replace assets, or claim that a
rerun of the old tagged workflow contains the later fix. The manual tap PR is
the documented post-publication recovery path for identical artifact identity.
