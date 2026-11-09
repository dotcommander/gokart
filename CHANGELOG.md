# Changelog

## v0.18.0 (2026-11-09)

### Added
- `web.RetryTransport` / `web.NewRetryClient()` / `web.ShouldRetryHTTP()` / `web.RetryPolicy` — explicit, operation-owned retry `RoundTripper` with jittered exponential backoff (`delay = base × (0.5 + uniform(0, 0.25))`, `base = min(Initial * 2^attempt, MaxDelay)`; `Initial` falls back to 100ms when unset, `Attempts` must be positive — there is no implicit 3-attempt default). Retries network errors, 408, 429, and 5xx; other 4xx are permanent caller errors. The caller must supply a replayable request; the wrapper never buffers bodies. See the new "Retries (operation-owned)" section in `docs/components/web.md`.

### Fixed
- `web.ValidationErrors` now uses `errors.As` instead of a direct type assertion, so wrapped validator errors are recognized instead of returning nil.

### Changed
- README: install snippet uses `go install ...@latest` so it never goes stale between releases; the Next section links the runnable example collections (`examples/` and `docs/examples/`).
- `docs/components/web.md`: retries row now points at `web.RetryTransport` (was `go-retryablehttp`).
- `cmd/gokart`: dev-version marker constantized (`devGokartVersion`), clearing a `goconst` finding CI flagged; dependency bumps in `go.mod`/`go.sum`.

## v0.17.0 (2026-11-09)

### Added
- `logger.SetTUIHandler(appName)` — installs the `NewFile` logger as the `slog` default via `slog.SetDefault` for terminal-owned UIs (bubbletea/tview/termui), where the default stderr handler corrupts rendered frames. Falls back to `io.Discard` when the file can't be opened; call before `tea.NewProgram(...).Run()`.
- `web.Transport()` / `web.NewClient()` — production-tuned outbound HTTP client: `http.DefaultTransport` cloned (never `&http.Transport{}` from scratch, which drops TLS timeouts/HTTP2/proxy) with `MaxIdleConnsPerHost` 2→100 (~7x throughput on a saturated upstream), `MaxIdleConns` 100, and `IdleConnTimeout` 90s.
- Test coverage: `cli` (25.9% → 87.7%), `cache` (49.4% → 73.0%), `migrate` (64.3% → 75.7%), `postgres` (59.3% → 91.2% via failure-path tests against a closed port), `cmd/gokart` main (0% → 81.8%) and `commands` (61.3% → 73.6%), plus root `LoadConfig`/`ParseConfig`, `web` response helpers, `sqlite` constraint/readonly, and generic transaction-lifecycle tests (begin/rollback/commit/panic) — all without new dependencies.

### Fixed
- `docs/components/fslock.md` and `docs/api/gokart.md`: replace stale "unreleased" caveats with the published `v0.16.0` install (`go get github.com/dotcommander/gokart/fslock@v0.16.0`); both `kongapp` and `fslock` have been tagged and published since v0.14.0.
- `postgres.FromEnv`: read `DATABASE_URL` from the environment and fail when it is unset or invalid. The implementation previously passed an empty string to `pgxpool.ParseConfig`, silently ignoring `DATABASE_URL` and falling back to libpq/PG* defaults (wrong database). The stale `docs/components/postgres.md` `FromEnv` row claiming pgx reads its normal environment variables is corrected to match.
- `docs/components/postgres.md`: add a "Budget your pool" section — the 25/5/1h defaults are a convenience starting point, not a production budget; `MaxConns` must be derived from `max_connections` minus reserved slots divided by simultaneously deployed instances.
- `docs/components/sqlite.md`: add a "Durability" section — WAL + `NORMAL` is the silent default but can roll back recent accepted transactions on crash; authoritative stores should set `SynchronousFull`, and `OFF` is never for authoritative data. Add a "single-connection pools" section documenting the self-deadlock when querying `*sql.DB` while a `*sql.Tx` holds the sole connection, with the reads-before-writes fix.
- `docs/api/gokart.md`: add a "Choosing your config path" section — `os.UserConfigDir()` is platform-native (right for GUI/services); hardcode `~/.config/<app>/` via `os.UserHomeDir()` when a CLI needs one consistent dotfile path across Linux/macOS.
- `web` response helpers: encode JSON before committing headers so unencodable values return `500 {"error":"json encode failed"}` (and `JSONStatusE` returns the marshal error) instead of a silent `200` with an empty/truncated body.
- `logger.NewFile`: create log files `0600` and tighten pre-existing files to `0600` (logs can carry tokens/user data; the shared `os.TempDir()` is world-readable), and reject app names containing path separators, `..`, or NUL so the log path cannot escape the temp dir.
- `SaveState`/`LoadState`/`StatePath`: validate app and file names with the same local-name check as `ConfigDir`, rejecting absolute paths, empty values, and `..` segments so state cannot be written outside the app config directory (previously a crafted name like `../../x` was joined straight onto the config dir).

## v0.16.0 (2026-11-08)

### Breaking
- None. This is a look-and-feel and documentation release.

### Added
- Add `MIGRATION.md` at the repo root (moved from the README).
- Rework the README as a newcomer funnel: prerequisites, a 60-second start with real output, generated-file trees, pick-your-shape, and add-things-later.

### Changed
- `gokart new` help and docs: `--dry-run` discloses that it runs `go get`/`tidy`/`test`/`build` over the network; `--verify-only` prints `Verified <name> (mode)`; conflict errors list the files and offer `--skip-existing`; the JSON failure payload carries the dry-run and verify flags; path-form name errors echo the full input; help documents the preset form and the TTY wizard; `--force` help now reads "Overwrite existing files that the scaffold generates".
- `docs/components/generator.md`: the integration table shows upstream-direct dependencies (`modernc.org/sqlite`, `pgxpool`, `go-redis`, `openai-go/v3`); the `gokart new` flags table matches the actual `--help` (`GOKART_AUTO_VERIFY` env var, aliases, `--db` values, `--verify` default-on).
- `docs/components/kongapp.md`: replace the "unreleased" caveat with an install snippet for v0.16.0.
- README Packages table: the postgres, sqlite, and cache rows note that generated scaffolds use upstream packages directly.

### Fixed
- Split `sqlite/operations.go` into `operations.go` and `operations_backup.go` (300-line tripwire).
- `logger.NewFile` wraps the open error and reports the close error.
- Generator uses `slices.SortFunc` instead of `sort.Slice`.
- `cache`: `interface{}` replaced with `any` (8 sites); examples: `interface{}` replaced with `any` (2 sites).

## v0.15.0 (2026-11-08)

### Breaking
- Remove `web.StandardMiddleware`: middleware selection is caller policy. Pass an explicit `[]func(http.Handler) http.Handler` (for example `middleware.RequestID`, `middleware.RealIP`, `middleware.Recoverer`) to `web.NewRouter`.
- Remove `postgres.Config.ConnectionString`: use `URL`, or the discrete `Host`/`Port`/`User`/`Password`/`DBName`/`SSLMode` fields assembled by `DSN`. Config maps using `connection_string` must migrate to `url`.
- Remove the sqlite pragma passthroughs `Optimize`, `Vacuum`, and `WALCheckpointTruncate`. For `PRAGMA optimize` or `VACUUM`, call `db.ExecContext` directly; use `WALCheckpoint(ctx, db, WALCheckpointModeTruncate)` for the truncate variant.

### Changed
- `cache.RememberJSON` now marshals the computed value once and returns the stored bytes to the flight, with one unmarshal per caller into its own `dest` (previously one redundant marshal per caller). Follower callers on a double-check hit now receive a correctly decoded `dest`.

## v0.14.0 (2026-11-08)

### Breaking
- Remove the `cli` presentation layer: `Success`, `Info`, `Warning`, `Error`, `Dim`, `Bold`, `NewSpinner`, `WithSpinner`, `NewProgress`, `NewTable`, `SimpleTable`, `KeyValue`, `List`, `NumberedList`, and `SetStyledHelp`. Use lipgloss, bubbles, or lipgloss/table directly.
- Remove the `postgres` compatibility aliases `PostgresConfig` and `BuildConnectionString`. Use `Config` and `DSN`.
- Bound `LoadState` reads at 10 MiB through `ReadFileLimit`; larger files return an error satisfying `errors.Is(err, gokart.ErrTooLarge)` and a zero value.
- Generated projects no longer import any `github.com/dotcommander/gokart/*` package at runtime. Older generated projects that still do can be converted with `gokart detach`; see generated-code ownership docs.

### Added
- Add the `kongapp` module: caller-owned Kong command trees with context, streams, post-parse initialization, and teardown.
- Add the `fslock` module: advisory whole-file locks for cross-process coordination (Unix; `ErrUnsupported` elsewhere).
- Add bounded reads (`ReadAllLimit`, `ReadFileLimit`, `ErrTooLarge`) and exported `AtomicWriteFile` to the root module.
- Add `gokart detach` for converting older generated projects to upstream-direct code, and an interactive `gokart new` wizard.
- Add task-oriented documentation and runnable examples for every module, including an upstream-direct (zero-GoKart) HTTP service example.

### Changed
- Drop the root-module dependency from `sqlite` and `postgres`: the transaction helper is now module-local, so consumers no longer pull viper/mapstructure transitively.
- Generate PostgreSQL, SQLite, Redis, and AI integrations through upstream packages directly in new scaffolds.

### Fixed
- Upgrade `github.com/go-chi/chi/v5` to v5.5.0 (ClientIPFromXFF client-IP spoofing, Dependabot alerts #1–#3).
- Reject unsafe application directory names in configuration setup.

## v0.13.0 (2026-07-14)

### Added
- Add a complete TV-guide newcomer tutorial and matching runnable example.
- Report dependency, test, and build checks in human and JSON generator output.
- Generate focused command tests for flat and structured example scaffolds.

### Changed
- Make every structured scaffold managed by default while keeping flat projects
  manifest-free, including global flat projects.
- Prepare dependencies before verification and make generated projects testable
  through injected arguments and output writers.
- Initialize structured integrations only for commands that request them and
  preserve command and cleanup errors together.

### Fixed
- Return actionable partial-success output when scaffold dependency preparation
  fails, including the recovery command.
- Detect flat projects before reporting a missing manifest from `gokart add`.
- Pin the generated Go directive and expand release verification across flat,
  structured, managed, unmanaged, integration, and example scaffolds.

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
