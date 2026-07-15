# Build an offline SQLite CLI

Install the tagged CLI:

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.13.0
gokart new counter --module example.com/counter --db sqlite --example
cd counter
go run ./cmd greet --name Ada
```

You now have a verified, file-backed CLI with no external service. This tutorial adds a persistent `counter` command while preserving the generated ownership boundaries.

## 1. Understand the lifecycle

```text
cmd/main.go                         sole process-exit boundary
internal/commands/root.go           Kong tree and dependency initialization
internal/commands/greet.go          flags and presentation
internal/actions/greet.go           testable application behavior
internal/app/context.go             shared dependencies and cleanup
.gokart-manifest.json               generated hashes and integrations
```

`main` calls `commands.Execute` and alone converts an error into `os.Exit(1)`.
`Execute` parses the typed Kong command tree and registers a singleton provider
for `*app.Dependencies`. Configuration and SQLite are created only when the
selected command requests that dependency. After the command returns, `Execute`
joins the command error with `Dependencies.Close` so cleanup failures are not
lost. `app.Context` remains an alias for compatibility.

Keep flags and output in `internal/commands`. Put business operations in `internal/actions` or a service package.

## 2. Add typed configuration

The scaffold defines `app.AppConfigKeyDBPath` and reads `db_path`. In `internal/commands/root.go`, set an explicit default in `loadConfig` before `app.New` runs:

```go
v.SetDefault(app.AppConfigKeyDBPath, "counter.db")
```

The generated Viper setup maps `COUNTER_DB_PATH` to `db_path`. The typed root fields provide `--config`, `--verbose`, and `--quiet`. Without this default, the generated context places the database under the platform cache directory.

## 3. Create and run a migration

```bash
go get github.com/dotcommander/gokart/migrate@v0.13.0
mkdir -p migrations
```

Create `migrations/00001_create_counter.sql`:

```sql
-- +goose Up
CREATE TABLE counters (
    name TEXT PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE counters;
```

In `internal/app/context.go`, after `sqlite.Open` succeeds, run it before assigning `appCtx.DB`:

```go
if err := migrate.Up(ctx, db, migrate.Config{
	Dir: "migrations", Dialect: "sqlite3",
}); err != nil {
	_ = db.Close()
	return nil, fmt.Errorf("migrate database: %w", err)
}
appCtx.DB = db
```

Import `fmt` and `github.com/dotcommander/gokart/migrate`. File migrations resolve from the working directory. For a binary that runs anywhere, embed the directory and set `Config.FS`; see [migrations](components/migrate.md).

## 4. Add a transactional action

Create `internal/actions/counter.go`:

```go
package actions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dotcommander/gokart/sqlite"
)

func Increment(ctx context.Context, db *sql.DB, name string, by int64) (int64, error) {
	var value int64
	err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO counters(name, value) VALUES (?, ?)
			ON CONFLICT(name) DO UPDATE SET value = value + excluded.value`, name, by)
		if err != nil { return fmt.Errorf("increment counter: %w", err) }
		return tx.QueryRowContext(ctx,
			`SELECT value FROM counters WHERE name = ?`, name).Scan(&value)
	})
	return value, err
}
```

The action owns the transactional write and query. It accepts the narrow `*sql.DB` dependency rather than the entire application context.

## 5. Add the `counter` command

Create `internal/commands/counter.go`:

```go
package commands

import (
	"context"
	"fmt"

	"example.com/counter/internal/actions"
	"example.com/counter/internal/app"
	"github.com/alecthomas/kong"
)

type CounterCommand struct {
	Name string `arg:"" optional:"" help:"Counter name."`
	By   int64  `default:"1" help:"Amount to add."`
}

func (c *CounterCommand) Run(kctx *kong.Context, ctx context.Context, appCtx *app.Context) error {
	name := c.Name
	if name == "" { name = "default" }
	value, err := actions.Increment(ctx, appCtx.DB, name, c.By)
	if err != nil { return err }
	appCtx.Log.Info("counter incremented", "name", name, "value", value)
	_, err = fmt.Fprintln(kctx.Stdout, value)
	return err
}
```

Register it beside the generated greet command in the root `CLI` type:

```go
type CLI struct {
	// existing fields...
	Greet   GreetCommand   `cmd:"" help:"Greet someone."`
	Counter CounterCommand `cmd:"" help:"Increment a persistent counter."`
}
```

No command-specific assignment is required. Requesting `*app.Context` (the
compatibility alias for `*app.Dependencies`) makes Kong invoke the lazy provider.
The dependency-free generated `greet` command does not open SQLite.

Run the complete path:

```bash
go run ./cmd counter
go run ./cmd counter build --by 3
go run ./cmd counter build
```

Command results use `kctx.Stdout`, so tests can capture output through `kong.Writers`. Operational details go to `appCtx.Log`.

## 6. Add state and tests

Use SQLite for the counters. For a small preference such as the last selected name, use typed state:

```go
type UIState struct { LastCounter string `json:"last_counter"` }
err := gokart.SaveState("counter", "state.json", UIState{LastCounter: name})
```

`SaveState` publishes atomically under the platform user configuration directory; it is not a replacement for transactional domain data.

Test the action with `sqlite.OpenContext(t.Context(), filepath.Join(t.TempDir(), "test.db"))`, apply the same migration, call `Increment`, and assert the returned values. Test the command with explicit arguments, `kong.Writers`, and test `app.Dependencies`. See [testing](testing.md) for complete patterns.

## 7. Build and run

```bash
go mod tidy
go test ./...
go build -o counter ./cmd
./counter greet --name Ada
./counter counter release --by 2
```

The new command and action files are application-owned. `gokart add` may rewrite `internal/app/context.go` and `internal/commands/root.go`; read [generated-code ownership](generated-code.md) before further customization.

## Expand into a service

- PostgreSQL: `gokart add postgres`, configure `DATABASE_URL`, and adapt repositories to `*pgxpool.Pool`.
- HTTP: add the `web` module, bounded validated handlers, and graceful shutdown.
- Redis: `gokart add redis`, configure `REDIS_ADDR`, then use `Cache.Client()` and `Cache.Key()`.
- OpenAI: `gokart add ai`, configure `OPENAI_API_KEY`, and call the official SDK client in `app.Dependencies`.

Use the [recipes](recipes.md) and [full-service example](examples/README.md#service-composition) rather than creating a second application architecture.

## Other verified generator flows

```bash
gokart new mycli --db sqlite --example
gokart new service --structured --global
gokart new "$PWD" --verify-only
```

The first is the short form of this tutorial scaffold and selects structured mode because SQLite is an integration. The second explicitly combines a structured layout with platform-global configuration so it remains compatible with `gokart add`. The third runs `go mod tidy` and `go test ./...` against an existing target without scaffolding.
