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

| File | Component | Wraps |
|------|-----------|-------|
| `log.go` | Logger | `log/slog` |
| `config.go` | Config | `spf13/viper` |
| `httpserver.go` | Router | `go-chi/chi/v5` |
| `server.go` | Server | stdlib `net/http` (graceful shutdown) |
| `httpclient.go` | HTTP Client | `hashicorp/go-retryablehttp` |
| `validate.go` | Validator | `go-playground/validator/v10` |
| `postgres.go` | PostgreSQL | `jackc/pgx/v5` |
| `sqlite.go` | SQLite | `modernc.org/sqlite` (zero CGO) |
| `templ.go` | Templates | `a-h/templ` |
| `cache.go` | Cache | `redis/go-redis/v9` |
| `migrate.go` | Migrations | `pressly/goose/v3` |
| `state.go` | State Persistence | `encoding/json` (stdlib) |
| `openai.go` | OpenAI | `openai/openai-go/v3` |
| `response.go` | Response Helpers | stdlib `net/http` + `encoding/json` |

### CLI Subpackage (`gokart/cli`)

Wraps `spf13/cobra` + `charmbracelet/lipgloss` for CLI applications with styled output, tables, spinners, and editor input.

## Design Principles

- **Thin wrappers**: No business logic, just factory functions with sensible defaults
- **Direct access**: Return underlying types, don't hide them
- **Fight for inclusion**: stdlib-sufficient things stay in stdlib (no error helpers, file utilities, string utilities)

## Key Patterns

Config uses generics with automatic env binding:
```go
cfg, err := gokart.LoadConfig[AppConfig]("config.yaml")
// Reads config file + env vars (db.host → DB_HOST)
```

Transaction helpers auto-rollback on error/panic:
```go
gokart.WithTransaction(ctx, pool, func(tx pgx.Tx) error { ... })
gokart.SQLiteTransaction(ctx, db, func(tx *sql.Tx) error { ... })
```

Cache has Remember pattern (get-or-compute):
```go
cache.Remember(ctx, "key", time.Hour, func() (interface{}, error) { ... })  // Returns string
cache.RememberJSON(ctx, "key", time.Hour, &dest, func() (interface{}, error) { ... })  // Typed
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
