# GoKart API Reference

This document provides a comprehensive API reference for the main `gokart` package.

## Configuration

### `func LoadConfig[T any](paths ...string) (T, error)`

Loads configuration from the first available file path into type T.

**Features:**
- Supports multiple config paths (first found wins)
- Automatic environment variable binding
- DOT to UNDERSCORE env key mapping (e.g., `db.host` → `DB_HOST`)

**Supported formats:** JSON, YAML, TOML, HCL, envfile, Java properties

**Example:**
```go
type Config struct {
    DB struct {
        Host string `mapstructure:"host"`
        Port int    `mapstructure:"port"`
    } `mapstructure:"db"`
}
cfg, err := gokart.LoadConfig[Config]("config.yaml", "config.json")
```

---

### `func LoadConfigWithDefaults[T any](defaults T, paths ...string) (T, error)`

Loads configuration with default values pre-populated.

The defaults parameter provides fallback values that will be overridden by values from config files or environment variables.

**Example:**
```go
defaults := Config{
    DB: struct{Host string; Port int}{
        Host: "localhost",
        Port: 5432,
    },
}
cfg, err := gokart.LoadConfigWithDefaults(defaults, "config.yaml")
```

---

## HTTP Server

### `func ListenAndServe(addr string, handler http.Handler) error`

Starts an HTTP server with graceful shutdown.

Blocks until SIGINT or SIGTERM is received, then gracefully shuts down with a 30 second timeout.

---

### `func ListenAndServeWithTimeout(addr string, handler http.Handler, timeout time.Duration) error`

Starts an HTTP server with graceful shutdown and a custom shutdown timeout.

---

## HTTP Router

### `type RouterConfig struct`

```go
type RouterConfig struct {
    Middleware []func(http.Handler) http.Handler
    Timeout    time.Duration // request timeout (default: none)
}
```

---

### `var StandardMiddleware []func(http.Handler) http.Handler`

Production-ready middleware stack:
- RequestID: Injects request ID for tracing
- RealIP: Extracts real client IP from proxies
- Logger: Structured request/response logging
- Recoverer: Panic recovery

---

### `func NewRouter(cfg RouterConfig) chi.Router`

Creates a new chi router with configured middleware.

**Example:**
```go
router := gokart.NewRouter(gokart.RouterConfig{
    Middleware: gokart.StandardMiddleware,
    Timeout:    30 * time.Second,
})

router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
})

http.ListenAndServe(":8080", router)
```

---

## HTTP Client

### `type HTTPConfig struct`

```go
type HTTPConfig struct {
    Timeout   time.Duration // request timeout (default: 30s)
    RetryMax  int           // max retry attempts (default: 3)
    RetryWait time.Duration // wait between retries (default: 1s)
}
```

---

### `func NewHTTPClient(cfg HTTPConfig) *retryablehttp.Client`

Creates a retryable HTTP client with exponential backoff.

**Default configuration:**
- Timeout: 30s
- RetryMax: 3 attempts
- RetryWait: 1s base delay

The client automatically retries on network errors and 5xx responses.

**Example:**
```go
client := gokart.NewHTTPClient(gokart.HTTPConfig{
    Timeout:   10 * time.Second,
    RetryMax:  5,
    RetryWait: 2 * time.Second,
})
resp, err := client.Get("https://api.example.com/data")
```

---

### `func NewStandardClient() *http.Client`

Creates a standard `http.Client` with retry logic.

This is a convenience wrapper around `NewHTTPClient` that returns a standard library `http.Client` interface for drop-in compatibility.

**Example:**
```go
client := gokart.NewStandardClient()
resp, err := client.Get("https://api.example.com/data")
```

---

## Validation

### `type ValidatorConfig struct`

```go
type ValidatorConfig struct {
    // UseJSONNames uses json tag names in error messages instead of struct field names.
    // Default: true (more useful for API error responses)
    UseJSONNames bool
}
```

---

### `func NewValidator(cfg ValidatorConfig) *validator.Validate`

Creates a configured validator instance.

**Default configuration:**
- Uses JSON tag names for field identification
- Registers common custom validators (`notblank`)

**Example:**
```go
v := gokart.NewValidator(gokart.ValidatorConfig{})

type User struct {
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lte=130"`
}

if err := v.Struct(user); err != nil {
    // handle validation errors
}
```

---

### `func NewStandardValidator() *validator.Validate`

Creates a validator with default settings.

Convenience wrapper around `NewValidator` with zero configuration.

**Example:**
```go
v := gokart.NewStandardValidator()
err := v.Struct(myStruct)
```

---

### `func ValidationErrors(err error) map[string]string`

Extracts field-level errors from a validation error.

Returns `nil` if err is not a `validator.ValidationErrors`.

**Example:**
```go
if err := v.Struct(user); err != nil {
    for field, msg := range gokart.ValidationErrors(err) {
        fmt.Printf("%s: %s\n", field, msg)
    }
}
```

---

## PostgreSQL

### `func OpenPostgres(ctx context.Context, url string) (*pgxpool.Pool, error)`

Opens a PostgreSQL connection pool with default settings.

**Deprecated:** Use `postgres.Open` directly.

---

### `func OpenPostgresWithConfig(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error)`

Opens a PostgreSQL connection pool with custom settings.

**Deprecated:** Use `postgres.OpenWithConfig` directly.

---

### `func PostgresFromEnv(ctx context.Context) (*pgxpool.Pool, error)`

Opens a PostgreSQL pool using `DATABASE_URL` environment variable.

**Deprecated:** Use `postgres.FromEnv` directly.

---

### `func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error`

Executes a function within a PostgreSQL transaction.

**Deprecated:** Use `postgres.Transaction` directly.

---

## SQLite

### `func OpenSQLite(path string) (*sql.DB, error)`

Opens a SQLite database with default settings.

**Deprecated:** Use `sqlite.Open` directly.

---

### `func OpenSQLiteContext(ctx context.Context, path string) (*sql.DB, error)`

Opens a SQLite database with context.

**Deprecated:** Use `sqlite.OpenContext` directly.

---

### `func OpenSQLiteWithConfig(ctx context.Context, cfg SQLiteConfig) (*sql.DB, error)`

Opens a SQLite database with custom settings.

**Deprecated:** Use `sqlite.OpenWithConfig` directly.

---

### `func SQLiteInMemory() (*sql.DB, error)`

Creates an in-memory SQLite database for testing.

**Deprecated:** Use `sqlite.InMemory` directly.

---

### `func SQLiteTransaction(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error`

Executes a function within a SQLite transaction.

**Deprecated:** Use `sqlite.Transaction` directly.

---

## Cache (Redis)

### `type CacheConfig struct`

```go
type CacheConfig struct {
    // URL is the Redis connection string.
    // Format: redis://:password@host:port/db or redis://host:port
    URL string

    // Addr is the Redis server address (alternative to URL).
    // Default: localhost:6379
    Addr string

    // Password for Redis authentication.
    Password string

    // DB is the Redis database number.
    // Default: 0
    DB int

    // PoolSize is the maximum number of connections.
    // Default: 10
    PoolSize int

    // MinIdleConns is the minimum number of idle connections.
    // Default: 2
    MinIdleConns int

    // DialTimeout is the timeout for establishing new connections.
    // Default: 5 seconds
    DialTimeout time.Duration

    // ReadTimeout is the timeout for socket reads.
    // Default: 3 seconds
    ReadTimeout time.Duration

    // WriteTimeout is the timeout for socket writes.
    // Default: 3 seconds
    WriteTimeout time.Duration

    // KeyPrefix is prepended to all keys.
    KeyPrefix string
}
```

---

### `type Cache struct`

Wraps Redis client with convenience methods.

---

### `func DefaultCacheConfig() CacheConfig`

Returns production-ready defaults for cache configuration.

---

### `func OpenCache(ctx context.Context, addr string) (*Cache, error)`

Opens a Redis connection with default settings.

**Example:**
```go
cache, err := gokart.OpenCache(ctx, "localhost:6379")
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

---

### `func OpenCacheURL(ctx context.Context, url string) (*Cache, error)`

Opens a Redis connection using a URL.

**Example:**
```go
cache, err := gokart.OpenCacheURL(ctx, "redis://:password@localhost:6379/0")
```

---

### `func OpenCacheWithConfig(ctx context.Context, cfg CacheConfig) (*Cache, error)`

Opens a Redis connection with custom settings.

**Example:**
```go
cache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
    Addr:      "localhost:6379",
    Password:  "secret",
    KeyPrefix: "myapp:",
})
```

---

### `func (c *Cache) Client() *redis.Client`

Returns the underlying Redis client.

---

### `func (c *Cache) Close() error`

Closes the Redis connection.

---

### `func (c *Cache) Get(ctx context.Context, key string) (string, error)`

Retrieves a string value.

---

### `func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error`

Stores a string value with expiration.

---

### `func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error`

Retrieves and unmarshals a JSON value.

---

### `func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error`

Marshals and stores a value as JSON.

---

### `func (c *Cache) Delete(ctx context.Context, keys ...string) error`

Removes a key.

---

### `func (c *Cache) Exists(ctx context.Context, key string) (bool, error)`

Checks if a key exists.

---

### `func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error`

Sets a TTL on an existing key.

---

### `func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error)`

Returns the remaining TTL of a key.

---

### `func (c *Cache) Incr(ctx context.Context, key string) (int64, error)`

Increments a counter and returns the new value.

---

### `func (c *Cache) IncrBy(ctx context.Context, key string, value int64) (int64, error)`

Increments a counter by a specific amount.

---

### `func (c *Cache) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)`

Sets a value only if the key doesn't exist (for distributed locks).

---

### `func (c *Cache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (string, error)`

Gets a value or sets it using the provided function.

**Example:**
```go
user, err := cache.Remember(ctx, "user:123", time.Hour, func() (interface{}, error) {
    return db.GetUser(ctx, 123)
})
```

---

### `func (c *Cache) RememberJSON(ctx context.Context, key string, ttl time.Duration, dest interface{}, fn func() (interface{}, error)) error`

Gets a value or computes and caches it as JSON.

Unlike `Remember`, this preserves type information for `GetJSON` retrieval.

**Example:**
```go
var user User
err := cache.RememberJSON(ctx, "user:123", time.Hour, &user, func() (interface{}, error) {
    return db.GetUser(ctx, 123)
})
```

---

### `func IsNil(err error) bool`

Returns true if the error is a cache miss.

---

## Migrations

### `type MigrateConfig struct`

```go
type MigrateConfig struct {
    // Dir is the directory containing migration files.
    // Default: "migrations"
    Dir string

    // Table is the name of the migrations tracking table.
    // Default: "goose_db_version"
    Table string

    // Dialect is the database dialect (postgres, sqlite3, mysql).
    // Auto-detected if not specified.
    Dialect string

    // FS is an optional filesystem for embedded migrations.
    FS fs.FS

    // AllowMissing allows applying missing (out-of-order) migrations.
    // Default: false
    AllowMissing bool

    // NoVersioning disables version tracking (for one-off scripts).
    // Default: false
    NoVersioning bool
}
```

---

### `func DefaultMigrateConfig() MigrateConfig`

Returns sensible defaults for migration configuration.

---

### `func Migrate(ctx context.Context, db *sql.DB, cfg MigrateConfig) error`

Runs all pending migrations.

**Example with file-based migrations:**
```go
db, _ := gokart.OpenPostgres(ctx, url)
err := gokart.Migrate(ctx, db.Config().ConnConfig.Database, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

**Example with embedded migrations:**
```go
//go:embed migrations/*.sql
var migrations embed.FS

err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    FS:      migrations,
    Dir:     "migrations",
    Dialect: "postgres",
})
```

---

### `func MigrateUp(ctx context.Context, db *sql.DB, cfg MigrateConfig) error`

Runs all pending migrations (alias for `Migrate`).

---

### `func MigrateDown(ctx context.Context, db *sql.DB, cfg MigrateConfig) error`

Rolls back the last migration.

---

### `func MigrateDownTo(ctx context.Context, db *sql.DB, cfg MigrateConfig, version int64) error`

Rolls back to a specific version.

---

### `func MigrateReset(ctx context.Context, db *sql.DB, cfg MigrateConfig) error`

Rolls back all migrations.

---

### `func MigrateStatus(ctx context.Context, db *sql.DB, cfg MigrateConfig) error`

Prints the status of all migrations.

---

### `func MigrateVersion(ctx context.Context, db *sql.DB, cfg MigrateConfig) (int64, error)`

Returns the current migration version.

---

### `func MigrateCreate(dir, name, migrationType string) error`

Creates a new migration file.

**Example:**
```go
err := gokart.MigrateCreate("migrations", "add_users_table", "sql")
```

---

### `func PostgresMigrate(ctx context.Context, db *sql.DB, dir string) error`

Convenience function for PostgreSQL migrations.

**Example:**
```go
pool, _ := gokart.OpenPostgres(ctx, url)
db := stdlib.OpenDBFromPool(pool)
err := gokart.PostgresMigrate(ctx, db, "migrations")
```

---

### `func SQLiteMigrate(ctx context.Context, db *sql.DB, dir string) error`

Convenience function for SQLite migrations.

**Example:**
```go
db, _ := gokart.OpenSQLite("app.db")
err := gokart.SQLiteMigrate(ctx, db, "migrations")
```

---

## Templates (templ)

### `func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error`

Renders a templ component to an `http.ResponseWriter`.

Sets Content-Type to text/html and handles errors.

**Example:**
```go
func handleHome(w http.ResponseWriter, r *http.Request) {
    gokart.Render(w, r, views.HomePage("Welcome"))
}
```

---

### `func RenderWithStatus(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error`

Renders a templ component with a custom status code.

**Example:**
```go
func handleNotFound(w http.ResponseWriter, r *http.Request) {
    gokart.RenderWithStatus(w, r, http.StatusNotFound, views.NotFoundPage())
}
```

---

### `func RenderCtx(ctx context.Context, w http.ResponseWriter, component templ.Component) error`

Renders a templ component with a custom context.

**Example:**
```go
ctx := context.WithValue(r.Context(), "user", currentUser)
gokart.RenderCtx(ctx, w, views.Dashboard(data))
```

---

### `func TemplHandler(component templ.Component) http.Handler`

Creates an `http.Handler` from a templ component.

Useful for static pages or when you don't need request data.

**Example:**
```go
router.Get("/about", gokart.TemplHandler(views.AboutPage()))
```

---

### `func TemplHandlerFunc(fn func(r *http.Request) templ.Component) http.HandlerFunc`

Creates an `http.HandlerFunc` from a function that returns a component.

Useful when the component needs data from the request.

**Example:**
```go
router.Get("/user/{id}", gokart.TemplHandlerFunc(func(r *http.Request) templ.Component {
    id := chi.URLParam(r, "id")
    user := getUser(id)
    return views.UserPage(user)
}))
```

---

### `func TemplHandlerFuncE(fn func(r *http.Request) (templ.Component, error)) http.HandlerFunc`

Creates an `http.HandlerFunc` from a function that can return an error.

**Example:**
```go
router.Get("/dashboard", gokart.TemplHandlerFuncE(func(r *http.Request) (templ.Component, error) {
    data, err := loadDashboardData(r.Context())
    if err != nil {
        return nil, err
    }
    return views.Dashboard(data), nil
}))
```

---

## State Persistence

### `func SaveState[T any](appName, filename string, data T) error`

Saves typed state to `~/.config/{appName}/{filename}`.

The file is written as indented JSON for human readability. Directory is created with 0755, files with 0644 permissions.

**Example:**
```go
type AppState struct {
    LastOpened string `json:"last_opened"`
    WindowSize int    `json:"window_size"`
}
err := gokart.SaveState("myapp", "state.json", AppState{
    LastOpened: "/path/to/file",
    WindowSize: 1024,
})
```

---

### `func LoadState[T any](appName, filename string) (T, error)`

Loads typed state from `~/.config/{appName}/{filename}`.

Returns zero value and `os.ErrNotExist` if the file doesn't exist. This allows callers to distinguish between missing file and parse errors.

**Example:**
```go
state, err := gokart.LoadState[AppState]("myapp", "state.json")
if errors.Is(err, os.ErrNotExist) {
    // First run, use defaults
    state = AppState{WindowSize: 800}
} else if err != nil {
    return err
}
```

---

### `func StatePath(appName, filename string) string`

Returns the full path to a state file.

Returns empty string if the user config directory cannot be determined.

**Example:**
```go
path := gokart.StatePath("myapp", "state.json")
// Returns: /Users/username/.config/myapp/state.json (on macOS)
```

---

## OpenAI

### `func NewOpenAIClient(opts ...option.RequestOption) openai.Client`

Creates an OpenAI client with the given options.

By default, the SDK reads from the `OPENAI_API_KEY` environment variable.

---

### `func NewOpenAIClientWithKey(apiKey string) openai.Client`

Creates an OpenAI client with the specified API key.

---

## HTTP Response Helpers

### `func JSON(w http.ResponseWriter, data any)`

Writes a JSON response with status 200.

---

### `func JSONStatus(w http.ResponseWriter, status int, data any)`

Writes a JSON response with the given status code.

---

### `func JSONStatusE(w http.ResponseWriter, status int, data any) error`

Writes a JSON response with the given status code.

Returns an error if JSON encoding fails.

---

### `func Error(w http.ResponseWriter, status int, message string)`

Writes a JSON error response with the given status code.

---

### `func NoContent(w http.ResponseWriter)`

Writes a 204 No Content response.

---

## Deprecated Functions

The following functions are deprecated and should be replaced with their direct imports:

### Logger (use `github.com/dotcommander/gokart/logger`)

- `type LogConfig` → `logger.Config`
- `func NewLogger(cfg LogConfig) *slog.Logger` → `logger.New(cfg)`
- `func NewFileLogger(appName string) (*slog.Logger, func(), error)` → `logger.NewFile(appName)`
- `func LogPath(appName string) string` → `logger.Path(appName)`

### PostgreSQL (use `github.com/dotcommander/gokart/postgres`)

- `type PostgresConfig` → `postgres.Config`
- `func DefaultPostgresConfig(url string) PostgresConfig` → `postgres.DefaultConfig(url)`
- `func OpenPostgres(ctx context.Context, url string) (*pgxpool.Pool, error)` → `postgres.Open(ctx, url)`
- `func OpenPostgresWithConfig(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error)` → `postgres.OpenWithConfig(ctx, cfg)`
- `func PostgresFromEnv(ctx context.Context) (*pgxpool.Pool, error)` → `postgres.FromEnv(ctx)`
- `func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error` → `postgres.Transaction(ctx, pool, fn)`

### SQLite (use `github.com/dotcommander/gokart/sqlite`)

- `type SQLiteConfig` → `sqlite.Config`
- `func DefaultSQLiteConfig(path string) SQLiteConfig` → `sqlite.DefaultConfig(path)`
- `func OpenSQLite(path string) (*sql.DB, error)` → `sqlite.Open(path)`
- `func OpenSQLiteContext(ctx context.Context, path string) (*sql.DB, error)` → `sqlite.OpenContext(ctx, path)`
- `func OpenSQLiteWithConfig(ctx context.Context, cfg SQLiteConfig) (*sql.DB, error)` → `sqlite.OpenWithConfig(ctx, cfg)`
- `func SQLiteInMemory() (*sql.DB, error)` → `sqlite.InMemory()`
- `func SQLiteTransaction(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error` → `sqlite.Transaction(ctx, db, fn)`
