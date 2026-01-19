# GoKart Code Examples

Complete, runnable examples demonstrating GoKart patterns for building production-ready Go services.

## Quick Start

Each example is self-contained and can be run directly:

```bash
cd docs/examples/<example-name>
go run main.go
```

## Prerequisites

| Example | Requirements |
|---------|--------------|
| logger | None |
| config | None (creates config.yaml if needed) |
| http-server | None |
| sqlite | None (uses file-based DB) |
| postgres | PostgreSQL + `DATABASE_URL` env var |
| cache | Redis on localhost:6379 |
| openai | `OPENAI_API_KEY` env var |
| full-service | PostgreSQL + Redis (optional, gracefully degrades) |

## Examples

### [logger](./logger/)

Structured logging with `log/slog`. Demonstrates:

- Default JSON logger for production
- Text formatter for development
- File logger for TUI apps (keeps stdout clean)
- Log levels: debug, info, warn, error
- Structured key-value pairs and grouped attributes

```go
log := logger.New(logger.Config{Level: "debug", Format: "text"})
log.Info("user created", "id", 123, "email", "user@example.com")
```

### [config](./config/)

Configuration loading with Viper. Demonstrates:

- YAML/JSON file loading with multiple path support
- Automatic environment variable binding (db.host -> DB_HOST)
- Type-safe generics: `LoadConfig[T]()` and `LoadConfigWithDefaults[T]()`
- Nested configuration structs

```go
cfg, err := gokart.LoadConfig[AppConfig]("config.yaml", "config.json")
// Environment vars automatically override: DATABASE_HOST, FEATURES_ENABLE_CACHE
```

### [http-server](./http-server/)

HTTP service with chi router. Demonstrates:

- StandardMiddleware: RequestID, RealIP, Logger, Recoverer
- Request timeout configuration
- Graceful shutdown on SIGINT/SIGTERM
- Response helpers: `JSON()`, `Error()`, `NoContent()`, `JSONStatus()`
- RESTful CRUD endpoints

```go
router := gokart.NewRouter(gokart.RouterConfig{
    Middleware: gokart.StandardMiddleware,
    Timeout:    30 * time.Second,
})
gokart.ListenAndServe(":8080", router)
```

### [postgres](./postgres/)

PostgreSQL with pgx connection pool. Demonstrates:

- Production-ready connection pooling defaults
- Query patterns: single row, multiple rows, execute
- Transaction helper with auto-rollback on error/panic
- Custom pool configuration

```go
pool, _ := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "Alice")
    return err // Rollback on error, commit on nil
})
```

### [sqlite](./sqlite/)

SQLite with zero CGO (pure Go). Demonstrates:

- File-based and in-memory databases
- WAL mode and performance pragmas (auto-configured)
- Transaction helper with auto-rollback
- Custom configuration options

```go
db, _ := sqlite.Open("app.db")          // File-based
memDB, _ := sqlite.InMemory()           // In-memory (great for tests)
sqlite.Transaction(ctx, db, func(tx *sql.Tx) error { ... })
```

### [cache](./cache/)

Redis caching with go-redis. Demonstrates:

- String and JSON get/set operations
- Remember pattern (get-or-compute): `cache.Remember()`, `cache.RememberJSON()`
- Counters: `Incr()`, `IncrBy()`
- Distributed locking with `SetNX()`
- Key prefixing for namespacing

```go
cache, _ := gokart.OpenCache(ctx, "localhost:6379")
cache.RememberJSON(ctx, "user:123", time.Hour, &user, func() (interface{}, error) {
    return db.GetUser(ctx, 123) // Only called on cache miss
})
```

### [openai](./openai/)

OpenAI API integration with openai-go v3. Demonstrates:

- Client creation (env var or explicit key)
- Chat completions with messages
- Multi-turn conversations
- Streaming responses
- Different model selection

```go
client := gokart.NewOpenAIClient() // Reads OPENAI_API_KEY from env
client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Messages: []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage("Hello!"),
    },
    Model: openai.ChatModelGPT4oMini,
})
```

### [full-service](./full-service/)

Complete production service combining all components:

- Configuration from file + environment
- Structured logging
- PostgreSQL database with connection pooling
- Redis caching with Remember pattern
- HTTP server with middleware and graceful shutdown
- Health check endpoint with dependency status
- Graceful degradation when services unavailable

```bash
# Run with all services
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"
go run main.go

# Or run with degraded mode (no DB/cache)
go run main.go
```

## Architecture Patterns

### Factory Functions Return Underlying Types

GoKart factory functions return the actual library types, not custom wrappers:

```go
pool, _ := postgres.Open(ctx, url)     // Returns *pgxpool.Pool
router := gokart.NewRouter(cfg)         // Returns chi.Router
client := gokart.NewOpenAIClient()      // Returns openai.Client
```

### Transaction Helpers

Both PostgreSQL and SQLite provide transaction helpers that:

- Auto-commit on success (nil return)
- Auto-rollback on error return
- Auto-rollback on panic (then re-panic)

```go
// PostgreSQL
postgres.Transaction(ctx, pool, func(tx pgx.Tx) error { ... })

// SQLite
sqlite.Transaction(ctx, db, func(tx *sql.Tx) error { ... })
```

### Remember Pattern

The cache Remember pattern simplifies get-or-compute caching:

```go
// Returns cached value or computes it
val, _ := cache.Remember(ctx, key, ttl, computeFunc)

// Same but with typed JSON serialization
var user User
cache.RememberJSON(ctx, key, ttl, &user, computeFunc)
```

## Testing Examples

Examples can serve as integration tests when services are available:

```bash
# Run all examples that don't require external services
go run ./docs/examples/logger
go run ./docs/examples/config
go run ./docs/examples/sqlite

# With PostgreSQL
export DATABASE_URL="postgres://user:pass@localhost:5432/testdb"
go run ./docs/examples/postgres

# With Redis
go run ./docs/examples/cache

# With OpenAI
export OPENAI_API_KEY="sk-..."
go run ./docs/examples/openai
```
