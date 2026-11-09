# GoKart

![GoKart logo](logo.png)

Bootstrap a Go project in one command. Output is plain Go you own.

## Prerequisites

- Go 1.26+ (`go version`)
- `$(go env GOBIN)` (or `$(go env GOPATH)/bin`) on PATH. `gokart config show` prints the `Binary:` path.

## Start

```bash
go install github.com/dotcommander/gokart/cmd/gokart@latest
gokart new hello --example
```

```text
Created hello (flat)
Verified: tests and build

Next:
  cd 'hello'
  go run . greet --name World
  go build -o hello .
```

```bash
cd hello
go run . greet --name World
go run . greet --name World --loud
```

```text
Hello, World
HELLO, World!
```

## What you got

```text
hello/
├── .gitignore
├── README.md     # build/test commands for the new project
├── go.mod        # module hello · go 1.26.0 · kong only
├── go.sum
├── main.go       # CLI struct · GreetCommand · run() · main()
└── main_test.go  # greet · usage · invalid-arg · write-failure tests
```

Structured projects (`--structured`, or any integration flag) use a multi-package layout:

```text
mycli/
├── .gokart-manifest.json   # hashes so `gokart add` can safely rewrite wiring
├── .gitignore
├── README.md
├── cmd/main.go                         # entry point, sole os.Exit boundary
├── go.mod / go.sum
└── internal/
    ├── actions/greet.go        # business logic (Validate + Greet)
    ├── actions/greet_test.go
    └── commands/root.go        # Kong CLI struct, Execute()/execute()
        commands/greet.go       # GreetCommand → actions.Greet
        commands/root_test.go
```

**No lock-in.** Generated code imports no `github.com/dotcommander/gokart/*` package at runtime. This was verified by grep on a flat scaffold, a structured scaffold, and a `--db postgres --redis --ai` scaffold. Flat projects import only `kong` and the standard library. Structured integration scaffolds import `pgxpool`, `go-redis`, `openai-go/v3`, and `viper` directly. `go test ./...` and `go build` need no gokart binary.

## Pick your shape

| I want | Command |
|---|---|
| Smallest runnable CLI | `gokart new mycli` |
| CLI with working example | `gokart new mycli --example` |
| Multi-package, no integrations | `gokart new mycli --structured` |
| Global config, still unmanaged | `gokart new mycli --global` |
| Global config + later `gokart add` | `gokart new mycli --structured --global` |
| Service with Postgres, Redis, AI | `gokart new svc --db postgres --redis --ai` |
| Add SQLite to structured project | `gokart add sqlite` |

## Add things later

```bash
cd mycli   # structured, managed
gokart add sqlite --dry-run
gokart add sqlite
```

Flat projects are unmanaged, so `gokart add` refuses with exit 10: "add the relevant GoKart package to this flat project manually, or start future growable projects with `gokart new <name> --structured`".

## Next

- Tutorial: [getting started](docs/getting-started.md)
- Library reference: [documentation index](docs/index.md)
- Generator reference: [generator](docs/components/generator.md)

## Packages

| Module | Purpose |
|---|---|
| `gokart` | Typed configuration, platform config directories, and JSON state persistence. |
| `gokart/kongapp` | Caller-owned Kong command trees with context, streams, and teardown. See [Kong lifecycle](docs/components/kongapp.md). |
| `gokart/web` | chi router/server construction, JSON responses, bounded binding, and validation. |
| `gokart/postgres` | pgx pool setup and transaction helpers. Generated scaffolds use pgxpool directly. |
| `gokart/sqlite` | Zero-CGO SQLite setup and operations. Generated scaffolds use modernc.org/sqlite directly. |
| `gokart/migrate` | goose migrations. |
| `gokart/cache` | Redis construction, prefixes, JSON operations, and `Remember`. Generated scaffolds use go-redis directly. |
| `gokart/logger` | `log/slog` setup. |
| `gokart/cli` | Legacy Cobra helpers for established apps. New apps: generated Kong code. |
| `gokart/fslock` | Advisory whole-file locks for cross-process coordination. See [file locks](docs/components/fslock.md). |

All modules isolate their dependencies. Constructors expose real upstream types or an explicit `Client` escape hatch.

## More

GoKart is v0.x (pre-1.0); breaking changes are recorded in [CHANGELOG.md](CHANGELOG.md). Scaffolds from before v0.14 can be converted with `gokart detach`. Upgrading from an older API? See [MIGRATION.md](MIGRATION.md). Design boundary: [PHILOSOPHY.md](PHILOSOPHY.md).

License: [MIT](LICENSE)
