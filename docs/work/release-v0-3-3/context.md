# Context

The latest published stable release is v0.3.2, while the shared Homebrew Formula remains v0.2.4 because strict audit rejected its redundant explicit version declaration. Tag-specific recovery renders from the old tag and therefore cannot incorporate the later Formula fix without a policy exception. An independently reviewed alternative is a new stable release from the default branch with the fix, preserving exact-revision recovery and immutable existing assets.

PR #12 merged as commit `65c84d8c482d5f65cba5a8fcc05303a74159cac0`; it updates Go to 1.26.10 and makes release-input fingerprinting fail closed. The exact merged commit passed `task check`, `task security`, `task release:check`, and `task public:check`. The GitHub App secret names exist, but its actual installation scope and permissions have not been confirmed. CI on that merge commit succeeded.

A pre-publication synthetic isolated macOS arm64 archive built from the exact commit with version v0.3.3 extracted with the three expected members, reported the exact version/revision, and rejected `help messages list --format agent` with `--help` guidance. This does not establish a post-publication Homebrew installation. The currently observed installed CLI reports v0.1.1; the older feedback described v0.2.4. The owner has been asked where the existing CLAUDE.md instruction is located so migration can be checked without altering unrelated configuration.

No v0.3.3 tag, GitHub Release, tap Formula PR, or new Homebrew installation has been created. The working tree containing the broader feedback implementation is separate and remains unfinished.
