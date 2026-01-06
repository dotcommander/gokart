# GoKart

Opinionated Go service toolkit. Thin wrappers around best-in-class packages with sensible defaults.

## Philosophy

- **Thin wrappers** — GoKart doesn't reinvent. It wraps battle-tested packages.
- **Sensible defaults** — Zero-config works. Customize when needed.
- **Fight for inclusion** — Every component must justify its existence.

## Install

```bash
go get github.com/dotcommander/gokart
go get github.com/dotcommander/gokart/cli
```

## Components

| Component | Wraps | Purpose |
|-----------|-------|---------|
| Logger | slog | Structured logging |
| Config | viper | Configuration + env vars |
| Router | chi | HTTP routing + middleware |
| HTTP Client | retryablehttp | HTTP client with retries |
| Validator | go-playground/validator | Struct validation |
| PostgreSQL | pgx/v5 | Postgres connection pool |
| SQLite | modernc.org/sqlite | SQLite (zero-CGO) |
| Templates | a-h/templ | Type-safe HTML templates |
| Cache | go-redis/v9 | Redis cache |
| Migrations | goose/v3 | Database migrations |
| CLI | cobra + lipgloss | CLI applications |

---

## Logger

Wraps `log/slog` with configuration helpers.

```go
log := gokart.NewLogger(gokart.LogConfig{
    Level:  "debug",  // debug|info|warn|error
    Format: "text",   // json|text
})

log.Info("server started", "port", 8080)
log.Error("request failed", "err", err, "path", "/api/users")
```

---

## Config

Wraps `spf13/viper` for typed configuration loading.

```go
type Config struct {
    Port int    `mapstructure:"port"`
    DB   string `mapstructure:"database_url"`
}

cfg, err := gokart.LoadConfig[Config]("config.yaml")
// Also reads PORT, DATABASE_URL from environment
```

---

## Router

Wraps `go-chi/chi` with standard middleware.

```go
router := gokart.NewRouter(gokart.RouterConfig{
    Middleware: gokart.StandardMiddleware,  // RequestID, RealIP, Logger, Recoverer
    Timeout:    30 * time.Second,
})

router.Get("/health", healthHandler)
router.Route("/api", func(r chi.Router) {
    r.Get("/users", listUsers)
    r.Post("/users", createUser)
})

http.ListenAndServe(":8080", router)
```

---

## HTTP Client

Wraps `hashicorp/go-retryablehttp` for resilient HTTP calls.

```go
// Simple - returns *http.Client
client := gokart.NewStandardClient()

// Configurable
client := gokart.NewHTTPClient(gokart.HTTPConfig{
    Timeout:   10 * time.Second,
    RetryMax:  5,
    RetryWait: 2 * time.Second,
})

resp, err := client.StandardClient().Get("https://api.example.com/data")
```

---

## Validator

Wraps `go-playground/validator` with JSON field names and common validators.

```go
v := gokart.NewStandardValidator()

type User struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lte=130"`
    Name  string `json:"name" validate:"required,notblank"`
}

if err := v.Struct(user); err != nil {
    for field, msg := range gokart.ValidationErrors(err) {
        fmt.Printf("%s: %s\n", field, msg)
    }
}
```

---

## PostgreSQL

Wraps `jackc/pgx/v5` with connection pooling.

```go
// Simple
pool, err := gokart.OpenPostgres(ctx, "postgres://user:pass@localhost:5432/mydb")
defer pool.Close()

// From DATABASE_URL env
pool, err := gokart.PostgresFromEnv(ctx)

// Custom config
pool, err := gokart.OpenPostgresWithConfig(ctx, gokart.PostgresConfig{
    URL:      "postgres://...",
    MaxConns: 50,
    MinConns: 10,
})

// Query
var name string
err = pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 1).Scan(&name)

// Transaction
err := gokart.WithTransaction(ctx, pool, func(tx pgx.Tx) error {
    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
    return err
})
```

---

## SQLite

Wraps `modernc.org/sqlite` (pure Go, zero CGO) with production defaults.

```go
// Simple
db, err := gokart.OpenSQLite("app.db")
defer db.Close()

// In-memory (for tests)
db, err := gokart.SQLiteInMemory()

// Custom config
db, err := gokart.OpenSQLiteWithConfig(ctx, gokart.SQLiteConfig{
    Path:         "app.db",
    WALMode:      true,
    ForeignKeys:  true,
    MaxOpenConns: 25,
})

// Transaction
err := gokart.SQLiteTransaction(ctx, db, func(tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "John")
    return err
})
```

---

## Templates

Wraps `a-h/templ` for type-safe HTML rendering.

```go
// In handler
func handleHome(w http.ResponseWriter, r *http.Request) {
    gokart.Render(w, r, views.HomePage("Welcome"))
}

// With status code
gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())

// As handler
router.Get("/about", gokart.TemplHandler(views.AboutPage()))

// Dynamic handler
router.Get("/user/{id}", gokart.TemplHandlerFunc(func(r *http.Request) templ.Component {
    id := chi.URLParam(r, "id")
    return views.UserPage(getUser(id))
}))
```

Note: Write `.templ` files and run `templ generate` - gokart provides HTTP integration.

---

## Cache

Wraps `redis/go-redis/v9` for Redis caching.

```go
// Simple
cache, err := gokart.OpenCache(ctx, "localhost:6379")
defer cache.Close()

// From URL
cache, err := gokart.OpenCacheURL(ctx, "redis://:password@localhost:6379/0")

// With prefix
cache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
    Addr:      "localhost:6379",
    KeyPrefix: "myapp:",
})

// String operations
cache.Set(ctx, "key", "value", time.Hour)
val, err := cache.Get(ctx, "key")
cache.Delete(ctx, "key")

// JSON operations
cache.SetJSON(ctx, "user:1", user, time.Hour)
cache.GetJSON(ctx, "user:1", &user)

// Counters
cache.Incr(ctx, "views")
cache.IncrBy(ctx, "views", 10)

// Distributed lock
ok, err := cache.SetNX(ctx, "lock:job", "worker-1", time.Minute)

// Remember pattern (get or compute)
val, err := cache.Remember(ctx, "expensive", time.Hour, func() (interface{}, error) {
    return computeExpensiveValue()
})

// Check cache miss
if gokart.IsNil(err) {
    // Key doesn't exist
}
```

---

## Migrations

Wraps `pressly/goose/v3` for database schema migrations.

```go
// PostgreSQL
pool, _ := gokart.OpenPostgres(ctx, url)
db := stdlib.OpenDBFromPool(pool)
err := gokart.PostgresMigrate(ctx, db, "migrations")

// SQLite
db, _ := gokart.OpenSQLite("app.db")
err := gokart.SQLiteMigrate(ctx, db, "migrations")

// Embedded migrations
//go:embed migrations/*.sql
var migrations embed.FS

err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    FS:      migrations,
    Dir:     "migrations",
    Dialect: "postgres",
})

// Operations
gokart.MigrateUp(ctx, db, cfg)       // Run pending
gokart.MigrateDown(ctx, db, cfg)     // Rollback one
gokart.MigrateDownTo(ctx, db, cfg, 5) // Rollback to version
gokart.MigrateReset(ctx, db, cfg)    // Rollback all
gokart.MigrateStatus(ctx, db, cfg)   // Print status

// Create new migration
gokart.MigrateCreate("migrations", "add_users_table", "sql")
```

Migration file format (`migrations/001_create_users.sql`):

```sql
-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;
```

---

## CLI

Subpackage `gokart/cli` wraps `spf13/cobra` + `charmbracelet/lipgloss`.

```go
import "github.com/dotcommander/gokart/cli"

func main() {
    app := cli.NewApp("myapp", "1.0.0").
        WithDescription("My application").
        WithEnvPrefix("MYAPP").
        WithStandardFlags()

    app.AddCommand(cli.Command("serve", "Start server", runServe))
    app.AddCommand(cli.Command("migrate", "Run migrations", runMigrate))

    if err := app.Run(); err != nil {
        os.Exit(1)
    }
}

func runServe(cmd *cobra.Command, args []string) error {
    cli.Info("Starting server...")
    return server.Run()
}
```

### Output Styling

```go
cli.Success("Operation completed")  // ✓ green
cli.Error("Operation failed")       // ✗ red
cli.Warning("Deprecated feature")   // ⚠ yellow
cli.Info("Processing...")           // → blue
cli.Dim("Debug info")               // gray

cli.Fatal("Cannot continue")        // prints + os.Exit(1)
cli.FatalErr("Failed", err)         // prints error + os.Exit(1)
```

### Tables

```go
t := cli.NewTable("ID", "Name", "Status")
t.AddRow("1", "Alice", "Active")
t.AddRow("2", "Bob", "Inactive")
t.Print()

// Quick table
cli.SimpleTable(
    []string{"Key", "Value"},
    [][]string{{"Host", "localhost"}, {"Port", "8080"}},
)

// Key-value list
cli.KeyValue(map[string]string{"Host": "localhost", "Port": "8080"})

// Bulleted list
cli.List("First item", "Second item", "Third item")
```

### Spinners & Progress

```go
// Spinner
s := cli.NewSpinner("Loading...")
s.Start()
// do work
s.StopSuccess("Loaded")

// With helper
err := cli.WithSpinner("Processing...", func() error {
    return doSomething()
})

// Progress bar
p := cli.NewProgress("Importing", 100)
for i := 0; i < 100; i++ {
    p.Increment()
    processItem(i)
}
p.Done()
```

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
| AI/LLM clients | domain-specific | Separate packages |
| Document processing | domain-specific | Separate packages |

---

## License

MIT
