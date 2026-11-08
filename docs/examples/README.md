# Examples

Runnable examples use the current `v0.14.0` package boundaries. Offline examples run anywhere; examples marked **(needs service)** require PostgreSQL and/or Redis.

```bash
go run docs/examples/logger/main.go
go run docs/examples/config/main.go
go run docs/examples/state/main.go
go run docs/examples/sqlite/main.go
go run docs/examples/migrate/main.go
go run docs/examples/validate/main.go
go run docs/examples/kongapp/main.go greet --name Ada
go run docs/examples/cli/main.go greet --name Ada
go run docs/examples/fslock/main.go
go run docs/examples/upstream-direct/main.go   # serves :8080, no GoKart imports
go run docs/examples/postgres/main.go          # (needs service) DATABASE_URL
go run docs/examples/cache/main.go             # (needs service) Redis on localhost:6379
go run docs/examples/http-server/main.go       # serves :8080
go run docs/examples/full-service/main.go      # (needs service) PostgreSQL + Redis
```

| Example | What it shows |
|---|---|
| `logger` | Zero-config and file loggers; returns `*slog.Logger` |
| `config` | Typed config, defaults, environment overrides |
| `state` | `SaveState`/`LoadState` under the platform config dir |
| `sqlite` | Defaults, transactions, in-memory DB for tests |
| `migrate` | Goose up/status against a temp migrations dir |
| `validate` | `BindAndValidate`: 201 / 422 / 400 without opening a port |
| `kongapp` | Caller-owned Kong tree with Initialize/cleanup |
| `cli` | Cobra app for ecosystem-style CLIs (new CLIs: prefer Kong) |
| `fslock` | Nonblocking advisory lock, `ErrLocked` handling |
| `postgres` | Pool defaults, transactions, `pool.Stat()` escape hatch |
| `cache` | `Client()` + `Key()` for ordinary commands, `RememberJSON` |
| `http-server` | Router, bounded binding, validation, response helpers |
| `full-service` | Config + logging + PostgreSQL + Redis + HTTP composition |
| `upstream-direct` | Same service with **no** GoKart imports — delete-the-wrapper path |
| `generator` | Copy-paste `gokart new` / `add` / `--verify-only` flow (README) |

The cache example uses `Client` plus `Key` for ordinary Redis commands. AI generation uses the official OpenAI SDK directly and therefore has no GoKart component example.

## Service composition

[`full-service/main.go`](full-service/main.go) is the canonical reference for composing typed configuration, logging, PostgreSQL, Redis, HTTP routing, bounded request handling, and graceful shutdown. It requires PostgreSQL and Redis; start with the offline [SQLite CLI tutorial](../sqlite-cli.md), then use this example when the application becomes a service.

For smaller changes, use the [recipes](../recipes.md). For safe generator edits, read [generated-code ownership](../generated-code.md).
