# Contributing to GoKart

Thank you for helping improve GoKart. Keep changes focused, preserve the public package boundaries, and include tests for behavior changes.

## Development setup

GoKart is a multi-module Go workspace. Use the repository workspace for development and the standalone gate to catch unpublished dependency or checksum mistakes.

```bash
scripts/verify-workspace.sh all
golangci-lint run ./...
(cd cmd/gokart && golangci-lint run ./...)
```

Generator changes must update the embedded templates, their golden trees, CLI help contracts, and documentation together. Regenerate and recheck goldens with:

```bash
GOKART_UPDATE_GOLDEN=1 go test ./cmd/gokart/internal/generator -run Golden
go test ./cmd/gokart/internal/generator -run Golden
```

Before opening a pull request, run the command-module race suite and ensure `git diff --check` is clean:

```bash
(cd cmd/gokart && go test -race ./...)
git diff --check
```

Do not commit generated binaries, credentials, local configuration, or unrelated cleanup.

## Pull requests

- Explain the user-visible behavior and compatibility impact.
- Call out generated-code or module-version changes explicitly.
- Include the exact verification commands you ran.
- Keep commits reviewable and use conventional commit subjects.

Security reports do not belong in public issues. Follow [SECURITY.md](SECURITY.md).
