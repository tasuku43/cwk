# Context: Chatwork User-Agent

## Verified facts

- `cmd/cwk/main.go` owns the injected `version` build variable and copies it to
  `CLI.Version` before command execution.
- `cli.New` constructs Chatwork lazily, so local help and control commands do
  not need to inspect `CWK_API_TOKEN` or initialize the HTTP adapter.
- `Client.Authenticate` performs the optional `/me` preflight through
  `Client.Execute`; `Execute` is the shared boundary for all task HTTP calls,
  including authenticated mutations and uploads.
- `ProductionBaseURL`, request mapping, `productionHTTPClient`, PAT binding,
  and redirect handling are independent of build version.
- The repository release parser accepts the SemVer value emitted by the
  packager after removal of the release tag's leading `v`; its release policy
  excludes build metadata and enforces SemVer numeric leading-zero rules.
- A user-agent version is bounded to 128 ASCII bytes before being joined to
  the fixed `cwk/` product token.

## Constraints

- Infrastructure validates the scalar again, even though `CLI.Version` comes
  from build injection.
- Unsupported values use the safe `cwk/dev` fallback without logging or
  echoing the original.
- PAT and session binding remain infrastructure-only; application receives no
  new version-bearing type.
- Tests use synthetic credentials and local HTTP servers only.

## Unknowns resolved

- Version lengths above 128 bytes are treated as invalid and fall back to
  `cwk/dev`.
- The header is set on each request made by the shared `Execute` path, including
  requests triggered from authentication preflight.
