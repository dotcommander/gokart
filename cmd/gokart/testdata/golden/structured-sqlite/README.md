# demo

```bash
go test ./...
go run ./cmd greet --name World
go build -o demo ./cmd
```

This is ordinary Go source. You own it and may edit, move, or delete any file.

## Structure and lifecycle

```text
cmd/main.go             Process entry point and sole exit-status boundary
internal/commands/      Cobra commands and dependency initialization
internal/actions/       Business logic independent of Cobra
internal/app/           Typed configuration and shared dependencies
```

`cmd/main.go` calls `commands.Execute`. The root command loads configuration in
`PersistentPreRunE`, creates `app.Context` only for commands that run, and closes
its database, pool, or cache before returning to `main`.

## Selected integrations

| Integration | Wiring | Environment |
| --- | --- | --- |
| sqlite | SQLite through sqlite.Open | DEMO_DB_PATH (optional; defaults to the user cache directory) |

The default SQLite database is in the user cache directory at
`demo/data.db`. Put schema files in `migrations/`; embed and run them from
`internal/app` when the application starts.

## Safe extension points

- Add commands under `internal/commands` and business logic under `internal/actions`.
- After you finish using `gokart add`, add dependency fields and cleanup in
  `internal/app/context.go`; until then, keep custom initialization in a
  user-owned file called from the generated context.
- Keep `cmd/main.go` small so deferred cleanup completes before `os.Exit`.
- `.gokart-manifest.json` records generated-file hashes. `gokart add` may rewrite
  `internal/app/context.go` and `internal/commands/root.go`; it refuses local
  changes unless you pass `--force`. Commit or copy changes before forcing.

## Build and test

Use Cobra's writers for command-owned output so tests can call `SetOut` and
`SetErr`. Test action code directly and use a temporary SQLite path for database
tests.

```bash
go test ./...
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o demo ./cmd
```

See the GoKart [generated-code guide](https://github.com/dotcommander/gokart/blob/main/docs/generated-code.md)
for manifest conflicts, removing integrations, and replacing wrappers with
upstream packages.
