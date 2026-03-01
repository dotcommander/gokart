# Getting Started

Build a working CLI app with a database in under 10 minutes.

---

## Prerequisites

- Go 1.22 or later (1.26+ recommended)
- `$GOPATH/bin` on your `PATH` (required to run the installed binary)

---

## Install GoKart

```bash
go install github.com/dotcommander/gokart/cmd/gokart@latest
```

Verify it works:

```bash
gokart --version
```

---

## Create Your First Project

The `--sqlite` flag wires up a SQLite database. The `--example` flag adds a sample command so you have something to run immediately.

```bash
gokart new myapp --sqlite --example
cd myapp
```

Generated file tree:

```
myapp/
├── .gitignore
├── .gokart-manifest.json          # Tracks generated files (do not edit)
├── CLAUDE.md                      # AI assistant guidance
├── README.md                      # Build and install commands
├── go.mod
├── cmd/
│   └── main.go                    # Entry point, wires version via ldflags
└── internal/
    ├── app/
    │   ├── config.go              # Config directory bootstrap (~/.config/myapp/)
    │   └── context.go             # Shared deps: logger, DB connection
    ├── actions/
    │   ├── greet.go               # Business logic (testable, no CLI deps)
    │   └── greet_test.go          # Table-driven tests
    └── commands/
        ├── root.go                # CLI setup, PersistentPreRunE wiring
        └── greet.go               # greet command: flags → actions.Greet
```

**What each piece does:**

- `cmd/main.go` — calls `commands.Execute(version)` and exits. No business logic here.
- `internal/app/context.go` — opens the SQLite database, creates the logger. Commands receive a `*app.Context` and call it.
- `internal/app/config.go` — creates `~/.config/myapp/` and a default `config.yaml` on first run. Called in `PersistentPreRunE` so it runs before any command.
- `internal/commands/root.go` — builds the `cli.App`, chains the `PersistentPreRunE` hook, registers commands.
- `internal/commands/greet.go` — parses flags, calls `actions.Greet`, prints the result.
- `internal/actions/greet.go` — pure function: takes `GreetInput`, returns `(string, error)`. No cobra, no globals.
- `internal/actions/greet_test.go` — tests `actions.Greet` directly without starting a CLI.

---

## Build and Run

```bash
go mod tidy
go build -o myapp ./cmd
```

Run with no arguments to see available commands:

```bash
./myapp
```

Expected output:

```
myapp - a GoKart CLI application

Usage:
  myapp [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  greet       Greet someone
  help        Help about any command

Flags:
  -h, --help     help for myapp
  -v, --verbose  Enable verbose output
      --version  version for myapp

Use "myapp [command] --help" for more information about a command.
```

Run the example command:

```bash
./myapp greet --name World
```

Output:

```
✓ Hello, World
```

With the `--loud` flag:

```bash
./myapp greet --name World --loud
```

Output:

```
✓ HELLO, WORLD!
```

Run the tests:

```bash
go test ./...
```

---

## Add a Command

Here is how to add a `count` command that stores a run count in the database.

### 1. Create the action

`internal/actions/count.go`:

```go
package actions

import (
	"context"
	"database/sql"
	"fmt"
)

// CountResult holds the result of a Count action.
type CountResult struct {
	Count int
}

// Count increments a named counter in the database and returns the new value.
func Count(ctx context.Context, db *sql.DB, name string) (CountResult, error) {
	_, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS counters (name TEXT PRIMARY KEY, value INTEGER NOT NULL DEFAULT 0)`,
	)
	if err != nil {
		return CountResult{}, fmt.Errorf("ensure table: %w", err)
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO counters (name, value) VALUES (?, 1)
		 ON CONFLICT(name) DO UPDATE SET value = value + 1`,
		name,
	)
	if err != nil {
		return CountResult{}, fmt.Errorf("increment counter: %w", err)
	}

	var n int
	err = db.QueryRowContext(ctx, `SELECT value FROM counters WHERE name = ?`, name).Scan(&n)
	if err != nil {
		return CountResult{}, fmt.Errorf("read counter: %w", err)
	}

	return CountResult{Count: n}, nil
}
```

### 2. Create the command

`internal/commands/count.go`:

```go
package commands

import (
	"context"
	"fmt"

	"github.com/dotcommander/gokart/cli"
	"github.com/spf13/cobra"

	"myapp/internal/actions"
	"myapp/internal/app"
)

func NewCountCmd(getAppContext func() *app.Context) *cobra.Command {
	return cli.Command("count", "Increment and display a named counter", func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		appCtx := getAppContext()

		result, err := actions.Count(context.Background(), appCtx.DB, name)
		if err != nil {
			cli.Error("count failed: %v", err)
			return err
		}

		cli.Success("counter %q = %d", name, result.Count)
		return nil
	})
}
```

> Replace `myapp/internal/...` with your actual module path from `go.mod` if it differs.

### 3. Register the command in root.go

Open `internal/commands/root.go` and add `NewCountCmd` alongside `NewGreetCmd`:

```go
cliApp.AddCommand(NewGreetCmd(func() *app.Context {
    return appCtx
}))
cliApp.AddCommand(NewCountCmd(func() *app.Context {  // add this
    return appCtx
}))
```

### 4. Try it

```bash
go build -o myapp ./cmd
./myapp count --name hits
```

Output:

```
✓ counter "hits" = 1
```

Run again:

```
✓ counter "hits" = 2
```

The count persists in `~/Library/Caches/myapp/data.db` (macOS) or `~/.cache/myapp/data.db` (Linux). No configuration needed — the path is set automatically in `internal/app/context.go`.

---

## Add an Integration

After a project is scaffolded, use `gokart add` to wire in new integrations without re-scaffolding.

Add the OpenAI client:

```bash
gokart add ai
```

Output:

```
✓ Added ai
  overwrite  internal/app/context.go
  overwrite  internal/commands/root.go
```

`gokart add` re-renders only the two integration-affected files (`context.go` and `root.go`), runs `go get` for the new dependencies, and updates the manifest. Your `cmd/main.go`, action files, and command files are untouched.

After adding, `internal/app/context.go` will include an `AI openai.Client` field wired from `OPENAI_API_KEY`:

```go
type Context struct {
    Log *slog.Logger
    DB  *sql.DB
    AI  openai.Client
}
```

Use it from any command:

```go
resp, err := appCtx.AI.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Model:    openai.ChatModelGPT4o,
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Hello!"),
    },
})
```

To preview what `gokart add` would change before committing:

```bash
gokart add postgres --dry-run
```

---

## What's Next

- [CLI package](/api/cli) — styled output, tables, spinners, editor input
- [SQLite](/components/sqlite) — configuration, transactions, querying patterns
- [Web](/components/web) — chi router, graceful server, response helpers
- [Generator reference](/components/generator) — all `gokart new` and `gokart add` flags
