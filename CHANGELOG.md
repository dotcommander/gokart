# Changelog

## v0.12.0 (2026-07-14)

### Breaking
- Change the default no-integration scaffold from structured to flat.

### Added
- Add `--structured` to explicitly select the multi-package layout.

### Fixed
- Make generator streams race-free and align Kong documentation, templates,
  dependency binding, and effective generator-version metadata.

## v0.11.0 (2026-07-12)

### Breaking
- Remove the `ai` and `fs` modules, root map getters, Redis command mirrors,
  CLI fatal/writer overrides, and policy-heavy web helpers.
- Move config-directory setup to the root module and require direct upstream
  APIs for OpenAI, ordinary Redis commands, templ, retries, and rate limiting.

### Changed
- Pin generated GoKart and upstream dependencies for deterministic scaffolds.
- Generate PostgreSQL through `postgres.Open` and OpenAI through the official SDK.
- Establish `PHILOSOPHY.md` as the admission and deletion contract.
- Add a runnable newcomer flow covering migrations, greeting, and persisted
  counter behavior.
- Harden workspace, documentation, release-layout, and generated-scaffold
  verification across every surviving module.

## v0.10.3 (2026-07-12)

### Added
- Add scalar configuration parsing and identifier helpers.

### Fixed
- Include the workspace module manifest so fresh-clone verification covers every
  published module consistently.

## v0.10.2 (2026-07-12)

### Added
- Add typed configuration parsing helpers and explicit PostgreSQL configuration.
- Add SQLite profiles, transactions, checkpoints, health checks, retries,
  savepoints, inspection, maintenance, and backup operations.
- Add provider-scoped migrations and typed migration status results.
- Publish the `cmd/gokart` module as an independently installable component.

### Security
- Require API keys in the `X-API-Key` header; query-string credentials are no longer accepted.
- Return a generic forbidden response when credential validation fails instead of exposing callback errors.
- Expand generated `.gitignore` files to cover common environment and private-key variants.

### Changed
- Stop generating provider-specific coding-agent guidance in new projects.
- Align public package and config-path documentation with the current APIs.
- Correct installation, scaffold management, generator mutation, safety,
  modularity, contributor verification, and direct-library README guidance.

### Fixed
- Report the tagged module version from binaries installed with `go install`
  when no linker-supplied version is present.
- Make workspace verification bootstrap an isolated `go.work` in fresh clones.

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
