# Work Context: Require first-run command selection

## Current behavior

- With an absent preference, `resolveActiveCatalog` enables every configurable command in `internal/cli/cli.go`.
- In a clean temporary `XDG_CONFIG_HOME`, `cwk --help` lists all 34 Chatwork commands by namespace.
- In that state, `cwk rooms list` reaches PAT resolution and fails as `chatwork_token_missing`.
- `cwk doctor` reports `state=valid source=default enabled=34 disabled=0`.
- The `config` selector already starts an absent profile with every current configurable command selected.

## Relevant structure

- Entry point: `internal/cli/cli.go` (`RunContext`, `resolveActiveCatalog`)
- Domain rule: `internal/domain/commandselection`
- Application use case: `internal/app/configcmd`
- Infrastructure boundary: `internal/infra/commandconfig`
- CLI catalog or presentation: `internal/cli/config.go`, `help.go`, catalog-derived active view
- Existing tests and harness checks: `catalog_active_view_test.go`, `config_command_test.go`, `help_test.go`, command-selection harness claims

## Constraints

- The complete `DefaultCatalog` remains the product and release ledger.
- Help, routing, recovery, and workflows must consume the same active view.
- `help`, `doctor`, `version`, and `config` remain always-on and recoverable.
- Initial selection remains terminal-native and writes nothing before Enter.
- The preference remains a cognitive-surface setting, not an authorization boundary.
- Missing state must fail before PAT resolution and provider I/O for Chatwork tasks.

## External facts

No new external facts or API behavior are involved.

## Unknowns

- [x] Whether initial `config` should begin empty or all-selected: retain the existing all-selected draft so Enter can explicitly preserve the complete surface.
- [x] Whether root help should fail or explain the reduced view: return successful control-plane help with an explicit unconfigured notice.

## Thesis evidence

- Repeated design decision or point of agent confusion: an absent profile silently looked like a deliberate all-enabled selection.
- User outcome or friction observed in the minimal slice: the first agent help request paid the full command-context cost before configuration.
- Code workaround or exception being considered: blocking execution while still advertising all commands would split help from routing.
- Current thesis that resolves it, or proposed thesis revision: Axiom 2's shared active attention view should apply from the first invocation.
- Downstream product, architecture, security, Skill, catalog, and harness impact: revise the missing-state default and its executable claims; no catalog capability or external boundary changes.

## Reproduction or observation

```sh
XDG_CONFIG_HOME=<empty-directory> cwk --help
XDG_CONFIG_HOME=<empty-directory> cwk rooms list
XDG_CONFIG_HOME=<empty-directory> cwk doctor
```

Observed on macOS/arm64 with the development build: full help, PAT failure, and valid all-enabled default respectively.

After implementation, the same clean configuration home produced control-only
root and agent help, `command_selection_required` before PAT resolution for
`rooms list`, and `state=unconfigured source=missing enabled=0 disabled=34`
from `doctor`.

## Security and public-boundary notes

- Assets and side effects involved: one existing non-secret local preference; no new side effect.
- Credentials or confidential data involved: none; missing state must stop before PAT resolution.
- New dependencies, destinations, files, processes, or generated content: none.
- External schema provenance, publication rights, and drift evidence: not applicable.
- Pagination, timeout, retry, idempotency, and cancellation facts: unchanged.
- Publication and licensing concerns: none.

## Glossary

- Unconfigured: no command-selection preference has been saved.
- Control commands: the always-on local commands `help`, `doctor`, `version`, and `config`.
- Saved empty selection: a configured profile that deliberately enables no Chatwork command; distinct from unconfigured.
