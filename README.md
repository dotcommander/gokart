<p align="center">
  <img src="logo.png" alt="GoKart Logo" width="200">
</p>

# GoKart

Opinionated Go service toolkit. Thin wrappers around best-in-class packages with sensible defaults, so the repetitive setup is handled and you can get to the interesting part.

> **Note:** Not affiliated with [Praetorian's GoKart](https://github.com/praetorian-inc/gokart) (static security scanner). This GoKart is a service/CLI toolkit.

## Why?

Every Go service has the same 50-100 lines of setup boilerplate: configure slog with JSON/text switching, set up chi with standard middleware, parse postgres URLs with pool limits, wire viper to read config + env vars. You have run this experiment dozens of times. It works, but it is tedious and easy to get slightly wrong.

**GoKart turns that repeatable setup into one clean, reliable move.**

```go
app := cli.NewApp("myapp", "1.0.0").
    WithDescription("My tool").
    WithStandardFlags()
app.AddCommand(cli.Command("run", "Execute task", runTask))
app.Run()
```

Five lines. Real app. No ceremony.

- Start fast with a CLI, then grow into an HTTP microservice using the same core GoKart tools.

It's not a framework. No hidden runtime. Factory functions return `*pgxpool.Pool`, `chi.Router`, `*redis.Client` - use them directly. If you disagree with a default, change it or use the underlying package. GoKart doesn't lock you in.

And for web services, the same philosophy:

```go
pool, _ := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
router := web.NewRouter(web.RouterConfig{Middleware: web.StandardMiddleware})
cache, _ := cache.Open(ctx, "localhost:6379")
```

## Philosophy

- **Modular** — Import only what you need. Each component is its own Go module.
- **Thin wrappers** — GoKart doesn't reinvent. It wraps battle-tested packages.
- **Sensible defaults** — Zero-config works. Customize when needed.
- **Fight for inclusion** — If stdlib or the underlying package already solves it well, it stays out.

## Quick Start

```bash
go install github.com/dotcommander/gokart/cmd/gokart@latest

gokart new mycli           # Structured CLI project
gokart new mycli --sqlite  # With SQLite wiring
gokart new mycli --ai      # With OpenAI client
gokart new mycli --postgres --ai  # Full stack
```

See [Getting Started](docs/getting-started.md) for a full walkthrough: scaffold, add integrations, run tests.

## Components

| Package | Description | Docs |
|---------|-------------|------|
| `gokart/cli` | App builder, styled output, tables, spinners, editor bridge | [CLI Package](docs/api/cli.md) |
| `cmd/gokart` | `gokart new` + `gokart add` project scaffolder | [Generator](docs/components/generator.md) |
| `gokart` | Typed config (viper), state persistence, logger aliases | [Root Package](docs/api/gokart.md) |
| `gokart/logger` | slog wrapper — JSON/text, file logger for TUI tools | [Logger](docs/components/logger.md) |
| `gokart/web` | chi router, graceful server, response helpers, templ, validation, CSRF, pagination | [Web Toolkit](docs/components/web.md) |
| `gokart/postgres` | pgx/v5 connection pool, transaction helper | [PostgreSQL](docs/components/postgres.md) |
| `gokart/sqlite` | Zero-CGO SQLite, WAL mode, transaction helper | [SQLite](docs/components/sqlite.md) |
| `gokart/migrate` | goose/v3 schema migrations, embedded FS support | [Migrations](docs/components/migrate.md) |
| `gokart/cache` | Redis client, Remember pattern, distributed locks | [Cache](docs/components/cache.md) |
| `gokart/ai` | openai-go v3 client factory | [OpenAI](docs/components/openai.md) |

### Install packages individually

```bash
go get github.com/dotcommander/gokart           # Config, state, logger
go get github.com/dotcommander/gokart/cli        # CLI framework
go get github.com/dotcommander/gokart/web        # Router, server, templ, validation
go get github.com/dotcommander/gokart/postgres   # PostgreSQL pool
go get github.com/dotcommander/gokart/sqlite     # SQLite (zero CGO)
go get github.com/dotcommander/gokart/cache      # Redis cache
go get github.com/dotcommander/gokart/migrate    # Database migrations
go get github.com/dotcommander/gokart/ai         # OpenAI client
```

## Examples

See [`examples/`](examples/) for complete working examples:
- [`http-service/`](examples/http-service/) — Minimal HTTP API with chi router
- [`cli-app/`](examples/cli-app/) — CLI with commands, tables, and spinners

Additional runnable examples for every component: [docs/examples/](docs/examples/README.md)

---

## Not Included

GoKart intentionally excludes:

| What | Why | Use Instead |
|------|-----|-------------|
| Error helpers | stdlib sufficient | `errors.Is/As`, `fmt.Errorf("%w")` |
| File utilities | stdlib sufficient | `os`, `io`, `filepath` |
| String utilities | stdlib sufficient | `strings` |
| Env helpers | viper handles it | `viper.AutomaticEnv()` |
| DI container | architecture choice | Constructor injection |
| AI/LLM clients | now a subpackage | `github.com/dotcommander/gokart/ai` |
| Document processing | domain-specific | Separate packages |

---

## Compatibility

**Minimum Go version:** 1.22+ (1.26+ recommended)

**Stability:**
- **Library API** (`gokart`, `gokart/cli`): Follows semver. Breaking changes only in major versions.
- **Generator templates** (`gokart new`): May evolve between minor versions. Generated code is yours to modify.

---

## License

MIT
