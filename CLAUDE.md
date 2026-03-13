# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build ./...           # Build all packages
go test ./...            # Run all tests
go test -v ./...         # Verbose test output
go test -run TestName    # Run specific test
go install ./cmd/gokart  # Install gokart CLI
```

## Architecture

GoKart is an opinionated Go service toolkit providing thin wrappers around battle-tested packages. Factory functions return the underlying types directly (e.g., `*pgxpool.Pool`, `chi.Router`, `*redis.Client`), not custom abstractions.

### CLI Generator (`cmd/gokart`)

Scaffolds new CLI projects:
```bash
gokart new mycli                    # Structured, global (default)
gokart new mycli --local            # Structured, no global config
gokart new mycli --flat             # Single main.go, local
gokart new mycli --flat --global    # Single main.go, global config
gokart new mycli --sqlite           # With SQLite wiring
gokart new mycli --postgres         # With PostgreSQL wiring
gokart new mycli --ai               # With OpenAI client (v3)
```

### Add Integrations (`gokart add`)

Surgically adds integrations to an existing structured project without re-scaffolding:
```bash
gokart add sqlite              # Add SQLite wiring
gokart add ai                  # Add OpenAI client
gokart add postgres --dry-run  # Preview changes
gokart add ai --force          # Overwrite modified files
```

Only re-renders integration-affected files (`internal/app/context.go`, `internal/commands/root.go`), runs `go get` for dependencies, and updates the manifest. Does not touch `go.mod` directly — uses `go get` + `go mod tidy`.

**Global mode** (default for structured, opt-in for flat):
- Creates `~/.config/<app>/config.yaml` on first run
- Generates CLAUDE.md documenting paths for AI assistants
- Generates README.md with build commands

**Version embedding** (all scaffolded projects):
```bash
# Dev build
go build -o mycli ./cmd

# Release build with git tag
go build -ldflags "-X main.version=$(git describe --tags)" -o mycli ./cmd
```

### Main Package (`gokart`)

Root package provides logger, config, and state persistence.

| File | Component | Wraps |
|------|-----------|-------|
| `log.go` | Logger | `log/slog` |
| `config.go` | Config | `spf13/viper` |
| `state.go` | State Persistence | `encoding/json` (stdlib) |

### Subpackages

| Package | Purpose | Key dependencies |
|---------|---------|-----------------|
| `gokart/web` | HTTP toolkit: router, server, response, templ, negotiate, flash, CSRF, pagination, assets (~980 lines across 11 files, each ≤144 lines — small focused helpers, not a framework) | `chi/v5`, `a-h/templ`, `validator/v10` |
| `gokart/postgres` | PostgreSQL connection pool, transactions | `jackc/pgx/v5` |
| `gokart/sqlite` | SQLite (zero CGO), transactions | `modernc.org/sqlite` |
| `gokart/migrate` | Database migrations | `pressly/goose/v3` |
| `gokart/ai` | OpenAI client | `openai/openai-go/v3` |
| `gokart/cache` | Redis cache with Remember pattern | `redis/go-redis/v9` |
| `gokart/cli` | CLI applications with styled output | `cobra`, `lipgloss` |
| `gokart/logger` | Structured logging | `log/slog` |

### CLI Subpackage (`gokart/cli`)

Wraps `spf13/cobra` + `charmbracelet/lipgloss` for CLI applications with styled output, tables, spinners, and editor input.

## Design Principles

- **Thin wrappers**: No business logic, just factory functions with sensible defaults
- **Direct access**: Return underlying types, don't hide them — `*pgxpool.Pool`, `*sql.DB`, `chi.Router`, `*redis.Client`, `*validator.Validate`
- **Fight for inclusion**: stdlib-sufficient things stay in stdlib (no error helpers, file utilities, string utilities)
- **Web is not a framework**: `gokart/web` looks large because it lists many features, but it's 11 small files (~980 lines total) that each do one thing. Every function returns standard types (`chi.Router`, `http.Handler`). It's a toolkit of HTTP helpers sharing an import path, not an opinionated framework

## Key Patterns

Config uses generics with automatic env binding:
```go
cfg, err := gokart.LoadConfig[AppConfig]("config.yaml")
// Reads config file + env vars (db.host → DB_HOST)
```

Transaction helpers auto-rollback on error/panic:
```go
postgres.Transaction(ctx, pool, func(tx pgx.Tx) error { ... })
sqlite.Transaction(ctx, db, func(tx *sql.Tx) error { ... })
```

Content negotiation serves JSON or HTML from the same handler:
```go
web.Negotiate(w, r, jsonData, views.Page(data))
// JSON for Accept: application/json, HTML otherwise
```

State persistence for CLI tools (separate from config):
```go
gokart.SaveState("myapp", "state.json", myState)
state, _ := gokart.LoadState[MyState]("myapp", "state.json")
// Uses os.UserConfigDir(): macOS ~/Library/Application Support/, Linux ~/.config/
```

Scaffolded global CLIs use `~/.config/<app>/` on all platforms for consistency.

File logger keeps stdout clean for UI:
```go
logger, cleanup, _ := gokart.NewFileLogger("myapp")
defer cleanup()
// Writes to /tmp/myapp.log
```

Editor bridge for long-form input:
```go
text, _ := cli.CaptureInput("# Enter description", "md")
// Opens $EDITOR, returns edited content
```

Migrations support embedded files via `embed.FS`.
