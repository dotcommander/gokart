# PostgreSQL

Production-ready PostgreSQL connection pooling and transaction management built on [pgx/v5](https://github.com/jackc/pgx).

## Installation

```bash
go get github.com/dotcommander/gokart/postgres
```

## Quick Start

```go
import "github.com/dotcommander/gokart/postgres"

// Open with default settings
pool, err := postgres.Open(ctx, "postgres://user:pass@localhost:5432/mydb")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Query
var name string
err = pool.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", 1).Scan(&name)
```

---

## Connection

### Opening a Pool

#### Using Default Settings

```go
pool, err := postgres.Open(ctx, "postgres://user:pass@localhost:5432/mydb")
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

#### Using Custom Configuration

```go
pool, err := postgres.OpenWithConfig(ctx, postgres.Config{
    URL:               "postgres://user:pass@localhost:5432/mydb",
    MaxConns:          50,
    MinConns:          10,
    MaxConnLifetime:   2 * time.Hour,
    MaxConnIdleTime:   15 * time.Minute,
    HealthCheckPeriod: 30 * time.Second,
})
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

#### From Environment Variable

```go
// Reads DATABASE_URL from environment
pool, err := postgres.FromEnv(ctx)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

---

## Configuration

### Config Struct

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `URL` | `string` | *required* | PostgreSQL connection string |
| `MaxConns` | `int32` | `25` | Maximum connections in pool |
| `MinConns` | `int32` | `5` | Minimum idle connections |
| `MaxConnLifetime` | `time.Duration` | `1 hour` | Connection reuse duration |
| `MaxConnIdleTime` | `time.Duration` | `30 minutes` | Idle time before close |
| `HealthCheckPeriod` | `time.Duration` | `1 minute` | Health check interval |

### Connection String Format (DSN)

```
postgres://user:password@host:port/database?sslmode=disable
```

**Components:**

| Part | Example | Description |
|------|---------|-------------|
| `user` | `app_user` | Database username |
| `password` | `secret123` | Database password |
| `host` | `localhost` | Database host |
| `port` | `5432` | Database port (default: 5432) |
| `database` | `myapp_db` | Database name |
| `sslmode` | `disable` | SSL mode: `disable`, `require`, `verify-ca`, `verify-full` |

**Common DSN Examples:**

```bash
# Local development
postgres://localhost:5432/myapp

# Production with SSL
postgres://app:pass@db.example.com:5432/myapp?sslmode=require

# Via socket (Unix)
postgres:///myapp?host=/var/run/postgresql

# With connection params
postgres://user:pass@localhost:5432/myapp?sslmode=disable&connect_timeout=10
```

### Environment Variables

GoKart follows standard conventions for database configuration:

```bash
# Standard DATABASE_URL (used by postgres.FromEnv)
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb"

# Structured config (for custom config loading)
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="app_user"
export DB_PASSWORD="secret"
export DB_NAME="myapp_db"
export DB_SSLMODE="disable"
```

When using with [`gokart.LoadConfig`](/components/config/), env vars bind automatically:

```go
type Config struct {
    Database URL `env:"DATABASE_URL,required"`
}
```

---

## Transactions

### Transaction Helper

The `postgres.Transaction` function provides automatic commit/rollback behavior:

```go
err := postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
    // Insert user
    _, err := tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "Alice")
    if err != nil {
        return err  // Automatically rolls back
    }

    // Update account
    _, err = tx.Exec(ctx, "UPDATE accounts SET balance = balance - $1 WHERE user_id = $2", 100, 1)
    if err != nil {
        return err  // Automatically rolls back
    }

    return nil  // Automatically commits
})
// If function returns error: transaction rolled back
// If function returns nil: transaction committed
```

### Rollback Behavior

**Automatic rollback on:**

- Error returned from function
- Panic occurs during transaction
- Both error and rollback failure (returns compound error)

**Automatic commit on:**

- Function returns `nil`

**Panic recovery:**

```go
err := postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
    // This panic is caught, transaction rolled back, then re-panicked
    panic("something terrible happened")
})
```

### Transaction Isolation Levels

For custom isolation levels, use the underlying `pgx.Tx` directly:

```go
tx, err := pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// Set isolation level
_, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL SERIALIZABLE")
if err != nil {
    return err
}

// ... perform operations ...

return tx.Commit(ctx)
```

---

## Querying

### Single Row

```go
var name string
var email string

err := pool.QueryRow(ctx, "SELECT name, email FROM users WHERE id = $1", userID).
    Scan(&name, &email)
```

### Multiple Rows

```go
rows, _ := pool.Query(ctx, "SELECT id, name FROM users WHERE active = true")
defer rows.Close()

for rows.Next() {
    var id int
    var name string
    if err := rows.Scan(&id, &name); err != nil {
        return err
    }
    // Process row
}

if err := rows.Err(); err != nil {
    return err
}
```

### Executing Statements

```go
// INSERT
result, err := pool.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "Bob")

// UPDATE
result, err := pool.Exec(ctx, "UPDATE users SET active = $1 WHERE id = $2", true, userID)

// DELETE
result, err := pool.Exec(ctx, "DELETE FROM sessions WHERE expires_at < $1", time.Now())

rowsAffected := result.RowsAffected()
```

---

## Best Practices

### Connection Pool Sizing

For CPU-bound workloads:

```go
// Formula: (GOMAXPROCS * 2) + (GOMAXPROCS / 2)
// For 8 cores: 20 connections
cfg.MaxConns = 20
```

For I/O-bound workloads:

```go
// Allow more connections for slow queries
cfg.MaxConns = 50
```

### Statement Caching

pgx automatically caches prepared statements. Default cache is 512 statements.

### Context Timeouts

Always use context with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

pool, err := postgres.Open(ctx, dsn)
```

### Resource Cleanup

Always defer `pool.Close()`:

```go
pool, err := postgres.Open(ctx, dsn)
if err != nil {
    return err
}
defer pool.Close()  // Ensures clean shutdown
```

---

## Type Mapping

| Go Type | PostgreSQL Type |
|---------|-----------------|
| `int8`, `int16`, `int32`, `int64`, `int` | `smallint`, `integer`, `bigint` |
| `uint8`, `uint16`, `uint32`, `uint64`, `uint` | `oid` (unsigned only) |
| `float32`, `float64` | `real`, `double precision` |
| `bool` | `boolean` |
| `string` | `text`, `varchar`, `char` |
| `time.Time` | `timestamp`, `timestamptz`, `date`, `time` |
| `[]byte` | `bytea` |
| `interface{}` (via `pgtype`) | `json`, `jsonb`, `uuid` |

For JSON/UUID types, use the [`pgtype`](https://pkg.go.dev/github.com/jackc/pgx/v5/pgtype) package.

---

## Reference

### Functions

| Function | Description |
|----------|-------------|
| [`Open`](#opening-a-pool) | Opens pool with default settings |
| [`OpenWithConfig`](#opening-a-pool) | Opens pool with custom config |
| [`FromEnv`](#from-environment-variable) | Opens pool from `DATABASE_URL` |
| [`DefaultConfig`](#config-struct) | Returns default configuration |
| [`Transaction`](#transaction-helper) | Executes function in transaction |

### See Also

- [`pgx` documentation](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [PostgreSQL connection strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [Connection pooling best practices](https://www.postgresql.org/docs/current/pool.html)
- [Migrations](/components/migrate) - Database schema versioning
- [SQLite](/components/sqlite) - Embedded database alternative
