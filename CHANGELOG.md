# Changelog

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
