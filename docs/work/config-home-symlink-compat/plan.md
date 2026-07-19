# Work Plan: Config-home symlink compatibility

- Status: Complete
- Goal: [goal.md](goal.md)
- Context: [context.md](context.md)

## Chosen approach

Canonicalize only an existing configuration-home symbolic link inside
`FileStore.baseDirectory`. Resolve the complete alias chain once, require an
absolute directory target, and return that real path to both load and save.
Leave an absent non-link path unchanged so first-save creation still works.
Keep the later `Lstat`, exact Unix mode, opened-root identity, target-shape,
same-directory temporary-file, rename, and directory-sync checks unchanged.

## Alternatives considered

### Require users to export a canonical `XDG_CONFIG_HOME`

Rejected as a permanent answer because it makes a common supported dotfiles
layout require a product-specific shell wrapper and causes installation-time
failure before the user can run `config`.

### Allow every symbolic link below the configuration home

Rejected because the `cwk` directory and preference file are mutation targets.
Retaining their no-link contracts prevents a replace operation from following
an independently redirected leaf.

### Resolve the path again before every filesystem operation

Rejected because repeated traversal lets a concurrently changed alias select a
different destination. The store instead resolves once per load/save call and
uses the resulting path thereafter.

## Design

### Public contract

No command, capability ID, role, reference flow, output schema, exit status, or
fault code changes. The supported local prerequisite expands: an existing
configuration-home link to a directory is accepted. Broken links and unsafe
owned targets retain `command_selection_unsafe` and `cwk doctor` recovery.

### Layer changes

- Domain: none.
- Application: none.
- Infrastructure: canonical configuration-home resolution and boundary tests.
- CLI and catalog: none.
- Documentation/harness: clarify allowed alias and rejected owned targets.

### Data and control flow

```text
XDG_CONFIG_HOME or $HOME/.config
  -> absolute path validation
  -> if final path is a link, one complete EvalSymlinks resolution
  -> resolved directory inspection
  -> real-path/cwk strict inspection and opened root
  -> real-path/cwk/command-selection.json strict read or replacement
```

### Error and cancellation behavior

An absent normal path remains default/unconfigured. An unreadable metadata
lookup remains unavailable. A present configuration-home link that cannot
resolve to an absolute directory is unsafe. Existing cancellation and
uncertain-save behavior is unchanged.

### Security and public boundary

The alias is resolved before joining `cwk`; later filesystem operations do not
reuse it. The owned directory and file stay non-link, exact-mode targets. No
credential, provider data, dependency, network destination, or license changes.

## Implementation slices

1. Add work-packet decision and failing allow/reject boundary tests.
2. Add bounded base-alias resolution in the store.
3. Update README and durable product/architecture/security/harness claims.
4. Run focused, full, security, and public checks.

## Verification

- Unit tests: symlinked base missing-state and save/load round trip; broken base
  link rejection; retained application/file link rejection.
- Manual observation: deterministic temporary-directory test substitutes for a
  developer home mutation.
- Agent-readiness: `doctor` requires one invocation and no external processing;
  the fixed layout reports default state rather than recovery.
- Required profiles: `task check`, `task security`, `task public:check`.

## Rollout and rollback

The change is backward compatible for normal directories and existing stored
profiles. It is targeted for `v0.1.1`; tag creation and publication are outside
this packet. Rolling back reintroduces the false unsafe failure but does not
migrate or corrupt stored state.

## Documentation promotion

- Product contract: configuration-home alias support.
- Architecture: resolve-once boundary and retained owned-target checks.
- Security model: why the alias is allowed and owned targets remain strict.
- Harness: positive alias test plus negative leaf-target tests.
- README: dotfiles compatibility and actionable unsafe-path troubleshooting.
