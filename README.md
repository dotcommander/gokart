# GoKart

![GoKart logo](logo.png)

GoKart is a modular toolkit for building Go CLIs and services with practical
defaults.

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.10.2
gokart new myapp --db sqlite --example
cd myapp
go run ./cmd greet --name World
```

Use `@latest` instead only if you intentionally want the newest published
version rather than a reproducible install.

The generated project is regular Go code built on Cobra and the integrations
you select. Keep using GoKart's helpers, customize the generated code, or use
the underlying libraries directly.

## What GoKart provides

- A project generator for structured CLI applications and single-file tools.
- Focused packages for configuration, CLI output, HTTP services, databases,
  migrations, Redis, logging, files, and OpenAI.
- Safe integration updates with dry runs, generated-file manifests, and
  conflict detection.
- Independently importable Go modules for each component. Ordinary Go module
  transitive dependencies still apply.

## Packages

- `gokart` — typed configuration and JSON state persistence with Viper and the
  standard library.
- `gokart/cli` — app building, styled output, tables, spinners, and editor
  integration with Cobra and Lip Gloss.
- `gokart/web` — router setup, graceful serving, responses, validation, CSRF,
  pagination, health checks, and rate limiting with chi, templ, and validator.
- `gokart/postgres` — PostgreSQL pool setup and transaction helpers with
  pgx/v5.
- `gokart/sqlite` — zero-CGO SQLite setup, WAL defaults, and transaction helpers
  with `database/sql` and modernc SQLite.
- `gokart/migrate` — SQL migrations with embedded filesystem support using
  goose/v3.
- `gokart/cache` — Redis access, key prefixes, JSON helpers, caching, and data
  structures with go-redis/v9.
- `gokart/fs` — atomic writes, configuration paths, and read-or-create helpers
  from the standard library.
- `gokart/ai` — OpenAI client construction with openai-go/v3.
- `gokart/logger` — structured JSON, text, and file logging with `log/slog`.

Many setup functions return standard or upstream types directly, including
`*pgxpool.Pool`, `*sql.DB`, `chi.Router`, `openai.Client`, and `*slog.Logger`.
Convenience types such as `cli.App` and `cache.Cache` expose their underlying
Cobra or Redis client when lower-level control is needed.

## Generate a project

`gokart new` creates a structured CLI project by default:

```bash
gokart new mycli                         # local-only CLI
gokart new mycli --example               # include a greet command and tests
gokart new mycli --flat                  # single main.go
gokart new mycli --global                # platform user config directory
gokart new mycli --db sqlite             # SQLite integration
gokart new mycli --db postgres --ai      # PostgreSQL and OpenAI
gokart new mycli --redis                 # Redis integration
gokart new mycli --dry-run --json        # machine-readable preview
```

Plain local scaffolds stay lightweight and do not write a management manifest.
They are unmanaged and cannot use `gokart add`. If you expect to add
integrations later, select `--global` or an integration during creation. Those
options write `.gokart-manifest.json`, which lets `gokart add` detect edits
before updating generated wiring.

## Add integrations

Run `gokart add` from a managed, structured project:

```bash
gokart add sqlite
gokart add ai redis
gokart add postgres --dry-run
```

The command re-renders `internal/app/context.go` and
`internal/commands/root.go`, runs `go get` and `go mod tidy` (which can change
`go.mod` and `go.sum`), and refreshes `.gokart-manifest.json`. It refuses to
overwrite modified generated wiring or existing wiring that the manifest does
not track. Use `--dry-run` to inspect the plan first.

Destructive override (advanced): `gokart add ai --force` overwrites conflicting
generated wiring. Use it only when you intend to discard those local edits.

See the [generator reference](docs/components/generator.md) for every flag,
manifest behavior, JSON output, and exit code.

## Use the libraries directly

Each component is independently importable:

```go
package main

import (
    "log"
    "net/http"

    "github.com/dotcommander/gokart/web"
)

func main() {
    router := web.NewRouter(web.RouterConfig{
        Middleware: web.StandardMiddleware,
    })
    router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
        web.JSON(w, map[string]string{"status": "ok"})
    })

    if err := web.ListenAndServe(":8080", router); err != nil {
        log.Fatal(err)
    }
}
```

Component guides and runnable examples are indexed in
[`docs/`](docs/index.md). Complete applications live in
[`examples/`](examples/). For database usage, see the maintained
[PostgreSQL example](docs/examples/postgres/main.go).

## Requirements and verification

- Go 1.26 or later.
- `just` for the short contributor verification command.
- External services are required only for the integrations that use them.

Run the repository checks across all modules and examples:

```bash
just verify
```

Without `just`, run the same checks directly:

```bash
scripts/verify-workspace.sh all
```

## License

[MIT](LICENSE)
