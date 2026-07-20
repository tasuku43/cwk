# Public Repository Boundary

Public publication is irreversible in practice. Removing a secret, private URL, personal record, or proprietary file in a later commit does not remove it from clones, caches, logs, or forks. This guide treats repository creation, derivation, fixtures, history, licensing, and release metadata as one public boundary.

## Clean-room derivation

Create a derived public project from this public template or from an explicit allowlist of reviewed files. Never copy a private repository's `.git` directory. Do not preserve private commit messages, branches, tags, pull-request artifacts, or generated caches for convenience.

If code or documentation is inspired by private work, rewrite it from approved requirements and confirm the rights to publish it. String replacement is not a legal or confidentiality review.

## Material that must not cross the boundary

- Credentials, tokens, keys, certificates, cookies, or authenticated URLs.
- Private domains, tenant names, repository names, organization identifiers, or internal documentation links.
- Real customer, employee, account, calendar, message, file, or operational data.
- Private incident details, vulnerability information, or security assumptions useful to an attacker.
- Proprietary source, generated output, schemas, examples, or screenshots without publication rights.
- Internal deployment steps, access groups, approval routes, and contact lists.
- Local absolute paths, usernames, shell history, editor state, build caches, or debug logs.

Use `example.com`, synthetic identifiers, fixed timestamps, and invented content in fixtures and documentation.

## Executable public guard

`task public:check` scans publishable regular files and fails before reading repository-controlled content when a symbolic link or special file is present. Under the retained `ready` profile it also rejects runnable template identity anywhere except `tools/internal/projectconfig/defaults.go`, which remains as the derivation provenance record.

Repository shape checks also reject Claude-specific policy paths, interrupted bootstrap residue, and root-level binary build artifacts. The template has one canonical `AGENTS.md` policy and a Codex harness; a parallel `CLAUDE.md` or `.claude/` tree is a failed hygiene check. The full-tree shape walk does not treat deliberately ignored local files such as `.env` as publishable content, but symbolic links and special files still fail closed.

Every local Markdown link must use a canonical repository-relative path that stays inside the repository and resolves to a publishable regular file without crossing a symbolic link. External URLs, `mailto:` links, same-document fragments, and examples inside fenced code blocks are outside this local-file check. External link availability still requires review because network state is not reproducible inside the repository gate.

The security scan recognizes common token formats, credential-bearing URLs, authorization headers, and secret assignments, including quoted JSON keys and values. Example values are exempt only when they use an explicit whole-value convention such as `dummy-value`, `example-token`, `${ACCESS_TOKEN}`, `env.ACCESS_TOKEN`, exact GitHub Actions `${{ secrets.ACCESS_TOKEN }}` syntax, `null`, or `[redacted]`. A marker embedded in a plausible real value, such as `production-dummy`, `contest-token`, or a secret expression with a suffix, is not an exemption.

These checks are a repository-specific backstop, not a claim that regular expressions can prove the absence of secrets. Public history and artifacts still require the approved full-history secret scanner and human confidentiality review.

## History and secret review

For a new public repository, review the complete history, not only `HEAD`.

- Confirm the first commit contains only reviewed public content.
- Scan all refs and generated artifacts with the approved secret scanner.
- Search for forbidden identifiers defined in repository policy.
- Inspect unusually large or binary objects.
- Verify ignored local files were never committed.
- Review workflow logs and release artifacts before making them public.

If sensitive material entered history, stop publication. Coordinate revocation and history remediation before any push; deleting the working-tree file is insufficient.

## Rights and license review

Before publication:

- Confirm who owns every copied or generated component.
- Choose and commit an explicit project license.
- Review dependency and bundled-asset licenses.
- Decide the inbound contribution policy, such as MIT inbound licensing, DCO, or CLA.
- Add required notices and attribution. The checked-in project `LICENSE` and `THIRD_PARTY_NOTICES` are reviewed release inputs and accompany every public binary archive; the notice file preserves the pinned Go runtime/standard-library license and patent grant as well as every linked third-party module license.
- Confirm names, logos, and examples do not imply unauthorized endorsement.

This template uses MIT. A derived project may keep MIT, but must record that as a deliberate decision rather than inheriting it silently.

## Security disclosure readiness

- Enable GitHub private vulnerability reporting or publish another private channel.
- State supported versions and response expectations in `SECURITY.md`.
- Do not ask reporters to disclose sensitive details in public issues.
- Document project-specific assets, trust boundaries, and limitations.
- Ensure maintainers can revoke credentials and pull or replace a release if needed.

## Community health

A public repository should provide, as appropriate:

- README with supported use and maturity status;
- LICENSE;
- CONTRIBUTING;
- CODE_OF_CONDUCT;
- SECURITY;
- support expectations;
- issue and pull-request templates;
- ownership and review rules;
- versioning and deprecation policy.

The base template provides the core technical documents. A derived project must fill real contacts, ownership, and support promises before inviting external users.

## Dependency and automation review

- Pin third-party workflow actions to immutable revisions.
- Pin security and generation tools to reviewed versions.
- Grant workflow tokens the minimum required permissions.
- Separate untrusted pull-request execution from privileged release jobs.
- Do not expose secrets to forked pull requests.
- Verify dependency integrity, licenses, and known vulnerabilities.
- Treat automated dependency or schema pull requests as untrusted changes that must pass the same checks.

## Public release review

Before each public release, verify:

- the tag points to reviewed source;
- all required profiles pass;
- version and commit metadata are correct;
- supported-platform artifacts are complete;
- checksums and any provenance or signatures are present and verified;
- archives contain only the intended executable, project `LICENSE`, and reviewed `THIRD_PARTY_NOTICES` artifact;
- installation instructions that do not depend on post-publication
  package-manager metadata work in a clean environment;
- release notes disclose compatibility, security, and migration impact;
- the annotated tag carries those reviewed notes and the release workflow
  publishes them unchanged rather than deriving incomplete notes from pull
  request history;
- any post-publication Formula recovery uses only an existing stable tag,
  verifies the exact public asset set and checksums without replacing it, and
  resumes the same audit plus App-scoped fresh-runner boundary;
- for a stable release, the candidate Formula was rendered and syntax-checked
  from the exact release revision and the post-publication audit/PR contract
  was reviewed;
- the Homebrew GitHub App remains installed only on
  `tasuku43/homebrew-tap` with the reviewed Contents and Pull requests
  permissions, without copying either secret value into review evidence;
- no artifact, Formula, URL, log, or metadata contains a forbidden identifier.

For a stable release, Homebrew availability is a post-publication rollout
check rather than a pre-tag claim. After GitHub Release publication, verify
that the macOS job strictly audited the exact Formula and that its shared-tap
pull request changes only `Formula/cwk.rb`. After that pull request merges,
perform a clean `brew install tasuku43/tap/cwk`, record the result in that
release's evidence record, and only then announce the Homebrew path as available.
The GitHub Release and its public checksums must exist before the new Formula
can be installed from the shared tap.

See [Release](06_release.md) for the artifact workflow.

## Automated and manual gates

`task public:check` is required, but it cannot decide ownership, confidentiality context, trademark use, or whether an example reveals an internal process. The release owner records manual review evidence in the release evidence record.

Minimum first-public-push checklist:

- [ ] Repository was created with clean public history.
- [ ] Theses and product contract are concrete.
- [ ] Security model covers every real side effect and credential.
- [ ] License and contribution terms were approved.
- [ ] Private reporting and maintainer contacts exist.
- [ ] Fixtures and docs contain only synthetic data.
- [ ] Full history and artifacts passed secret and identifier review.
- [ ] `task check`, `task security`, and `task public:check` passed.
- [ ] A human reviewer approved publication.
