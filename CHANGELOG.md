# Changelog

## v0.10.1 (2026-06-05)

### Changed
- Make plain CLI scaffolds local-only and unmanaged by default; write manifests only for global config or integration scaffolds.
- Restore the README logo as a tracked asset and update generator docs for the lighter default scaffold.

### Fixed
- Align scaffold onboarding docs with the live generated file trees and cache-path behavior.
- Route JSON test output through the CLI output writer so `go test -race ./cmd/gokart` passes without global stdout races.

## v0.10.0 (2026-05-29)

### Added
- Add configurable CLI package output writers for stdout/stderr helpers.
- Add workspace-wide verification that builds, tests, vets modules, and compile-checks ignored examples.

### Changed
- Cap SQLite defaults at one open connection to preserve immediate transaction locking.
- Bound JSON request body decoding in `web.BindJSON` and expose `BindJSONWithLimit` for custom caps.
- Split scaffold lock process checks by platform so Windows builds stay portable.
- Propagate command contexts into scaffold dependency resolution.

### Fixed
- Harden state file writes to use `0600` permissions.
- Synchronize public API docs with the current CLI, SQLite, and web binding behavior.

### Maintenance
- Update dependencies and Go toolchain directives.
- Expand tests around startup, workspace verification, JSON binding, and HTTP server cancellation.
