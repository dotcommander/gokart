# demo

```bash
go test ./...
go run ./cmd greet --name World
go build -o demo ./cmd
```

This is ordinary Go source. You own it and may edit, move, or delete any file. GoKart records generated shared files so `gokart add` can update
them safely.

## Structure and lifecycle

```text
cmd/main.go             Process entry point and sole exit-status boundary
internal/commands/      Kong commands and dependency initialization
internal/actions/       Business logic independent of Kong
internal/app/           Typed configuration and shared dependencies
```

`cmd/main.go` calls `commands.Execute` and keeps process exit handling at the
boundary.

## Global configuration

The first command creates `config.yaml` in the operating system's user
configuration directory. Edit `internal/app/config.go` to change its defaults.

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

Use `kong.Context.Stdout` for command-owned output and injected writers at the
parser boundary. Test action code directly.

```bash
go test ./...
go build -ldflags "-X main.version=$(git describe --tags 2>/dev/null || echo dev)" -o demo ./cmd
```

See the GoKart [generated-code guide](https://github.com/dotcommander/gokart/blob/main/docs/generated-code.md)
for manifest conflicts, removing integrations, and replacing wrappers with
upstream packages.
