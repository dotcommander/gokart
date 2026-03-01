# GoKart Documentation

Opinionated Go service toolkit. Thin wrappers around best-in-class packages with sensible defaults.

## Start Here

- [Getting Started](getting-started.md) — Install, scaffold a project with SQLite and an example command, then add integrations in under 10 minutes.

## CLI

- [CLI Package](api/cli.md) — App builder, styled output (success/error/warn/info), tables, spinners, progress, and editor bridge.
- [Generator](components/generator.md) — `gokart new` and `gokart add`: flags, presets, manifest format, conflict handling, and JSON output.

## Core

- [Root Package (Config, State, Logger)](api/gokart.md) — `LoadConfig`, `LoadConfigWithDefaults`, `SaveState`, `LoadState`, and deprecated logger aliases.
- [Logger](components/logger.md) — `log/slog` wrapper with JSON/text formats, zero-config defaults, and a file logger for TUI tools.
- [State Persistence](components/state.md) — Save and load typed structs as JSON in the platform config directory across CLI invocations.

## Web

- [Web Toolkit](components/web.md) — chi router, graceful server, retryable HTTP client, response helpers, templ rendering, content negotiation, flash, CSRF, pagination, and static assets.
- [Response Helpers](components/response.md) — `JSON`, `JSONStatus`, `Error`, and `NoContent` for writing consistent HTTP API responses.
- [Templ Rendering](components/templ.md) — `Render`, `RenderWithStatus`, and handler adapters (`TemplHandler`, `TemplHandlerFunc`, `TemplHandlerFuncE`) for templ components.
- [Validation](components/validate.md) — Struct validation with `go-playground/validator`, JSON field names in errors, and a custom `notblank` tag.

## Data

- [PostgreSQL](components/postgres.md) — pgx/v5 connection pool with `Open`, `FromEnv`, configurable pool settings, and a transaction helper with auto-rollback.
- [SQLite](components/sqlite.md) — Zero-CGO SQLite via `modernc.org/sqlite` with WAL mode, sensible pragmas, in-memory support, and a transaction helper.
- [Migrations](components/migrate.md) — Schema versioning with goose/v3: `Up`, `Down`, `DownTo`, `Reset`, `Status`, embedded migrations, and database-specific helpers.
- [Cache](components/cache.md) — Redis client (go-redis/v9) with `Get`/`Set`, JSON helpers, Remember pattern, counters, distributed locks, and key prefixing.
- [OpenAI](components/openai.md) — openai-go v3 client factory reading `OPENAI_API_KEY`, with examples for chat completions, streaming, and conversation history.

## Examples

- [Examples](examples/README.md) — Runnable examples for logger, config, HTTP server, PostgreSQL, SQLite, Redis cache, OpenAI, and a combined full-service app.
