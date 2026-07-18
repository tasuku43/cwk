# Chatwork CLI

Chatwork CLI is a runnable, public-ready foundation for building task-oriented Go command-line tools with coding agents. It starts as a small `cwk` binary and gives a derived project an explicit product thesis, a four-layer architecture, a machine-readable agent contract, typed side-effect and external-API boundaries, one verification gate, and a documented path to a public release.

The default repository is intentionally real and buildable:

- Go module: `github.com/tasuku43/cwk`
- Binary: `cwk`
- Display name: `Chatwork CLI`

The bootstrap tool replaces those exact defaults with validated project values. The defaults are not placeholder syntax, so the template can be built and tested before it is customized.

## What this template optimizes for

- A project-specific thesis that lets contributors and agents resolve ambiguous design choices.
- User tasks as the public vocabulary, instead of leaking transport or vendor APIs into the CLI.
- Explicit utility, discover, and act roles with opaque IDs passed unchanged between tasks.
- Pure domain rules, application use cases, infrastructure adapters, and a thin CLI composition root.
- Explicit `read`, `create`, and `write` effects with typed intent, target, and impact information.
- Structured command prerequisites, inputs, outputs, completeness, failures, and recovery actions for agents.
- A secret-free PAT authentication boundary plus policy-neutral foundations for pagination, timeout, retry/idempotency, and mutations.
- A single command catalog as the source of truth for routing and help.
- Executable architectural, security, release, and public-repository claims.
- A clean public boundary: no inherited organization names, private URLs, credentials, or internal history.

The repository fixes reusable vocabulary and enforcement points without turning transport details into public tasks. For authentication, it uses PAT-only secret-free requirements and sessions, a fail-closed application gate, an ephemeral infrastructure-issued binding passed unchanged through task ports, and exact credential-record revalidation before I/O. OAuth is not supplied by the current core; adding it later requires a new product, domain, security, dependency, and migration decision. [Authentication](docs/07_authentication.md) and [External API Contracts](docs/08_external_api_contracts.md) define the current boundary in detail.

## Start a derived project

Create a new repository from this template, then work from the new repository. Do not copy this repository's `.git` directory into an unrelated project.

For Codex, invoke [`$bootstrap-derived-cli`](.agents/skills/bootstrap-derived-cli/SKILL.md) first. It gathers the project identity, uses the same transactional tool described below, verifies imports and gates, and hands off to project-specific thesis work. The manual equivalent is:

1. Edit [`.harness/project.json`](.harness/project.json) with the new project identity and policy.
2. Preview the exact replacements:

   ```sh
   go run ./tools/bootstrap --dry-run
   ```

3. Apply the validated bootstrap:

   ```sh
   go run ./tools/bootstrap
   ```

4. Replace the generic project reasoning with concrete decisions, in this order:

   - [theses](docs/00_theses.md)
   - [product contract](docs/01_product_contract.md)
   - [security model](docs/03_security_model.md)
   - [authentication decision](docs/07_authentication.md)
   - [external API contracts](docs/08_external_api_contracts.md)
   - [release model](docs/06_release.md)

5. Run the canonical gates:

   ```sh
   task check
   task public:check
   ```

The bootstrap changes repository identity; it does not invent the product. A derived project is not ready merely because all names were replaced. Its north star, supported tasks, trust boundaries, and release promises must be made specific before implementation expands.

## Run the default CLI

```sh
go run ./cmd/cwk --help
go run ./cmd/cwk help --format agent
go run ./cmd/cwk help rooms --format agent
go run ./cmd/cwk help messages list --format agent
go run ./cmd/cwk doctor
```

The `doctor` task is a minimal utility slice through the domain, application, infrastructure, and CLI layers. Chatwork task commands now own the public discover-to-act workflows: for example, pass the canonical `room_ref` emitted by `rooms list` unchanged to `messages list --room`. Supply the Chatwork API token from the command environment before invoking an API task. The former synthetic sample pair remains only as an offline test fixture and is not returned by public help.

### Authenticate

Inject `CWK_API_TOKEN` into the command process through your shell or secret
manager, without putting the token in argv, a command literal, or a project
file. It is the sole Chatwork credential input; there is no authentication
method selector or login command. For example, a non-echoing shell prompt can
populate a shell value and export only that variable to child commands:

```sh
read -r -s CWK_API_TOKEN
export CWK_API_TOKEN
cwk rooms list
unset CWK_API_TOKEN
```

The token remains process-local: `cwk` does not persist it in XDG/AppData
configuration, an operating-system credential store, or a project file. Unset
the variable when the shell no longer needs it. Missing or malformed token
input fails before a Chatwork request and scoped help identifies
`CWK_API_TOKEN` as the required environment input.

### Read agent output

Chatwork success output uses the versioned `cwk-task-projection/1` text
contract. It prints canonical references directly and keeps only the fields
declared for that task, plus coverage/completeness and explicit trust framing.
For example, a synthetic room collection is shaped as:

```text
cwk-task-projection/1 task=rooms.list
coverage kind="provider_collection" complete=true
rooms count=2
  room-ref=4101 name=untrusted:"Synthetic Lab" type="group" role="admin" unread=0 mentions=1 tasks=0
  room-ref=4102 name=untrusted:"Synthetic Archive" type="group" role="member" unread=0 mentions=0 tasks=0
```

Pass a value such as `4101` unchanged to a declared `--room` input. Display
aliases, icon URLs, empty descriptions, zero coverage limits, duplicated
coverage prose, and other non-contract fields are not emitted. A bounded
message window still declares `complete=false`, its positive limit, unresolved
relationship count, typed To/reply/quote facts, and message bodies as
`untrusted` external text.

Success data is written to stdout only after the complete bounded result has been rendered. Failures go to stderr as stable text or schema-versioned JSON and distinguish invalid input, authentication, permission, missing or ambiguous targets, rate limits, temporary failures, policy rejection, cancellation, unsupported work, contract violations, and internal faults with dedicated exit statuses. Schema-v3 root agent help is a compact outcome/capability index whose machine-readable `scope_request` points to exact-command or namespace help. Only that scoped response returns the complete I/O, output, error, role, prerequisite, authentication, mutation, and reference-flow contracts, so catalog growth does not duplicate them at the root.

## Repository map

```text
cmd/cwk/                 thin executable entry point
internal/domain/             pure types, faults, effects, API envelopes
internal/app/                task use cases, auth/pagination/execution gates
internal/infra/              concrete adapters for external systems
internal/cli/                catalog, routing, rendering, composition root

docs/                        durable product and engineering reasoning
docs/decisions/              accepted and superseded architecture decisions
docs/work/                   bounded work packets for active changes
tools/                       repository-aware linters and bootstrap tooling
scripts/                     canonical checks and release helpers
.harness/project.json        project identity and machine-readable policy
.agents/skills/              first-run bootstrap and capability workflows
```

Read [the documentation map](docs/README.md) for the intended order and ownership of each document. Contributors and coding agents must also read [AGENTS.md](AGENTS.md).

For community participation and help, see the [Code of Conduct](CODE_OF_CONDUCT.md), [Contributing Guide](CONTRIBUTING.md), [Support Policy](SUPPORT.md), and [Security Policy](SECURITY.md).

## Verification profiles

All entry points delegate to `./scripts/check.sh`:

| Command | Purpose |
|---|---|
| `task check:fast` | Formatting, architecture, and focused tests for short feedback loops |
| `task check` | The full pre-merge gate |
| `task security` | Credential, dependency, egress, and public-boundary checks |
| `task release:check` | Packaging and release-contract checks |
| `task public:check` | Public-readiness and template-sanitization checks |

CI is the authority. Local hooks may run a faster profile, but they must call the same script rather than reimplementing policy.

## Public template policy

This repository uses public-safe runnable defaults and synthetic examples. A derived project must keep confidential material out of source, fixtures, documentation, generated files, build logs, and Git history. Review [the public repository guide](docs/05_public_repository.md) before the first push to a public remote.

## License

Chatwork CLI is available under the [MIT License](LICENSE). Derived projects must make an explicit license choice; keeping MIT is allowed, but it must not happen accidentally.
