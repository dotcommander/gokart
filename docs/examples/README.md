# Examples

Runnable examples use the current `v0.11.0` package boundaries.

```bash
go run docs/examples/logger/main.go
go run docs/examples/config/main.go
go run docs/examples/sqlite/main.go
go run docs/examples/postgres/main.go
go run docs/examples/cache/main.go
go run docs/examples/http-server/main.go
go run docs/examples/full-service/main.go
```

The cache example uses `Client` plus `Key` for ordinary Redis commands. AI generation uses the official OpenAI SDK directly and therefore has no GoKart component example.

## Service composition

[`full-service/main.go`](full-service/main.go) is the canonical reference for composing typed configuration, logging, PostgreSQL, Redis, HTTP routing, bounded request handling, and graceful shutdown. It requires PostgreSQL and Redis; start with the offline [SQLite CLI tutorial](../getting-started.md), then use this example when the application becomes a service.

For smaller changes, use the [recipes](../recipes.md). For safe generator edits, read [generated-code ownership](../generated-code.md).
