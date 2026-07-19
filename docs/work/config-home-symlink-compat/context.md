# Work Context: Config-home symlink compatibility

## Current behavior

- `FileStore.baseDirectory` returns the configured absolute path unchanged.
- `Load` and `Save` pass that path to `inspectDirectory`, whose `Lstat` rejects
  the final path when it is a symbolic link.
- The check does not reject symbolic links in earlier path components, so the
  final-component rejection is not a complete no-symlink traversal boundary.
- The `cwk` application directory must be an actual Unix `0700` directory and
  `command-selection.json` an actual Unix `0600` regular file.
- A missing configuration home is supported and is created on first save.

## Relevant structure

- Infrastructure boundary: `internal/infra/commandconfig/store.go`
- Unix resolver: `internal/infra/commandconfig/resolver_unix.go`
- Existing tests: `internal/infra/commandconfig/store_test.go` and
  `resolver_unix_test.go`
- Public guidance: `README.md`
- Durable claims: `docs/01_product_contract.md`, `docs/02_architecture.md`,
  `docs/03_security_model.md`, and `docs/04_harness.md`

## Constraints

- An absent configuration home must remain creatable; unconditional
  `filepath.EvalSymlinks` would incorrectly reject that state.
- Once an existing alias is resolved, later operations must use the resolved
  path rather than traverse the alias again.
- The change must not weaken validation of the `cwk`-owned directory or file.
- The stored content is non-secret cognitive-surface state, not authorization.
- The repository currently has an unrelated user-authored README change to the
  Homebrew installation commands; it must be preserved.

## External facts

None. This change relies on the Go filesystem contract already used by the
adapter and a locally reported macOS reproduction; it adds no external content
or dependency.

## Unknowns

- [x] Whether the failure can be explained by the final configuration-home
  `Lstat`: confirmed by source inspection and the existing `base symlink` test.
- [x] Whether accepting the base alias would also accept `cwk` or file links:
  no; those are inspected after base resolution and retain separate checks.

## Thesis evidence

- User friction: a clean Homebrew install was unusable in a common dotfiles
  layout before authentication or command selection.
- Repeated decision: symbolic links are useful namespace aliases but unsafe as
  mutable `cwk`-owned write targets.
- Local workaround rejected as the product answer: requiring every user to set
  `XDG_CONFIG_HOME` to a canonical target duplicates deterministic resolution
  that the adapter can perform safely.
- Thesis impact: no thesis revision is needed because Axiom 2 already requires
  executable tasks without guessing; product, architecture, security, README,
  and harness wording require propagation.

## Reproduction or observation

```sh
ln -s /absolute/dotfiles/config "$HOME/.config"
cwk doctor
```

Observed on the reported `v0.1.0` Homebrew installation: the
`command-selection` check failed with `state=unsafe` and
`code=command_selection_unsafe`. Expected for `v0.1.1`: the alias resolves to
its directory target and a missing preference reports the normal default state.

## Security and public-boundary notes

- Asset: non-secret `command-selection.json` preference.
- Side effect: same-directory temporary-file replacement under the resolved
  `cwk` directory; no new destination is introduced.
- Credentials/confidential data: none.
- Dependencies/network/generated content: none.
- Publication: README wording is Japanese and uses synthetic paths only.

## Glossary

- **configuration-home alias**: the final `XDG_CONFIG_HOME` or fallback
  `$HOME/.config` path when that path itself is a symbolic link.
- **owned target**: the `cwk` application directory or preference file that the
  adapter creates, validates, reads, or replaces.
