<p align="center">
  <img src="logo.png" alt="GoKart Logo" width="200">
</p>

# GoKart

Opinionated Go service toolkit. Thin wrappers around battle-tested packages that hand you the real types back — no lock-in, no hidden runtime.

> **Note:** Not affiliated with [Praetorian's GoKart](https://github.com/praetorian-inc/gokart) (static security scanner). This GoKart is a service/CLI toolkit.

## Install

```bash
go install github.com/dotcommander/gokart/cmd/gokart@latest
```

## 60 Seconds to a Working Project

```
$ gokart new myapi --postgres

  Created myapi/
  ✓ go mod init
  ✓ go get dependencies
  ✓ CLAUDE.md written

$ tree myapi/
myapi/
├── cmd/
│   └── main.go
├── internal/
│   ├── app/
│   │   └── context.go
│   └── commands/
│       ├── root.go
│       └── greet.go
├── go.mod
├── .gokart-manifest.json
├── CLAUDE.md
└── README.md

$ cd myapi && go run ./cmd
myapi 0.0.0
```

Add integrations later without re-scaffolding:

```bash
gokart add sqlite
gokart add ai
gokart add postgres --dry-run   # preview before applying
```

## What You Get Back

GoKart's factory functions return the underlying types directly. You call pgx, chi, and database/sql as if you wrote the setup yourself — because you effectively did.

```go
// postgres — returns *pgxpool.Pool, use pgx directly
pool, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
rows, _ := pool.Query(ctx, "SELECT id, name FROM users WHERE active = $1", true)

// sqlite — returns *sql.DB, use database/sql directly
db, err := sqlite.Open("app.db")
db.QueryContext(ctx, "SELECT id FROM sessions WHERE expires_at > ?", time.Now())

// web — returns chi.Router, use chi directly
router := web.NewRouter(web.RouterConfig{Middleware: web.StandardMiddleware})
router.Get("/health", func(w http.ResponseWriter, r *http.Request) { ... })
router.Use(middleware.RealIP)

// cache — returns *redis.Client, use go-redis directly
client, err := cache.Open(ctx, "localhost:6379")
client.Set(ctx, "key", value, 5*time.Minute)

// config — typed, generics, auto env binding (DB_HOST → db.host)
cfg, err := gokart.LoadConfig[AppConfig]("config.yaml")
```

No wrapper types. No `.Unwrap()`. If GoKart's defaults don't fit, reach past them.

## Packages

| Package | You get | Wraps |
|---------|---------|-------|
| `gokart` | typed config, state persistence, logger aliases | viper, slog |
| `gokart/cli` | `*cli.App`, styled output, tables, spinners, editor bridge | cobra, lipgloss |
| `gokart/web` | `chi.Router`, graceful server, response helpers, templ, CSRF, pagination, health checks, rate limiting, auth middleware | chi/v5, a-h/templ, validator/v10 |
| `gokart/postgres` | `*pgxpool.Pool`, transaction helper | pgx/v5 |
| `gokart/sqlite` | `*sql.DB`, WAL mode, transaction helper | modernc.org/sqlite |
| `gokart/migrate` | schema migrations, embedded FS support | goose/v3 |
| `gokart/cache` | `*redis.Client`, Remember pattern, distributed locks | go-redis/v9 |
| `gokart/kv` | `*KV` expanded Redis (hash, sorted set, set, list, counters) | go-redis/v9 |
| `gokart/fs` | atomic writes, config dir, read-or-create | stdlib only |
| `gokart/ai` | `*openai.Client` factory | openai-go v3 |
| `gokart/logger` | JSON/text slog, file logger for TUI tools | log/slog |

Import only what you need — each is a separate Go module.

## The Scaffolder

`gokart new` generates a structured CLI project wired to your chosen integrations:

```bash
gokart new mycli                    # Structured, global config (~/.config/mycli/)
gokart new mycli --local            # Structured, no global config
gokart new mycli --flat             # Single main.go
gokart new mycli --sqlite           # With SQLite wiring
gokart new mycli --postgres         # With PostgreSQL wiring
gokart new mycli --ai               # With OpenAI client
gokart new mycli --redis            # With Redis cache
gokart new mycli --postgres --ai    # Combined
```

`gokart add` surgically adds integrations to an existing project. It re-renders only the affected files (`internal/app/context.go`, `internal/commands/root.go`), runs `go get`, and updates the manifest.

```bash
gokart add sqlite
gokart add ai --force       # overwrite modified files
gokart add redis
gokart add postgres --dry-run
```

## Philosophy

- **Modular** — import only what you need, no forced dependencies.
- **Thin wrappers** — factory functions, sensible defaults, real types returned.
- **Fight for inclusion** — if stdlib or the underlying package already solves it, GoKart stays out of the way.
- **Web looks big, isn't** — `gokart/web` lists many features but it's 17 small files (~1300 lines total), each ≤144 lines, each returning standard types. It's a toolkit of HTTP helpers sharing an import path, not a framework.

## Examples

See [`examples/`](examples/) for complete working projects:
- [`http-service/`](examples/http-service/) — Minimal HTTP API with chi router
- [`cli-app/`](examples/cli-app/) — CLI with commands, tables, and spinners

## Compatibility

Minimum Go version: 1.22+ (1.26+ recommended)

**API stability:** Library packages (`gokart`, `gokart/cli`, etc.) follow semver. Generator templates may evolve between minor versions — generated code is yours to modify.

## License

MIT
