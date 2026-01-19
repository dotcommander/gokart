# SQLite

Production-ready SQLite database integration built on [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) with **zero CGO** - a pure Go implementation that compiles to a standalone binary without external dependencies.

## Installation

```bash
go get github.com/dotcommander/gokart/sqlite
```

## Quick Start

```go
import "github.com/dotcommander/gokart/sqlite"

// Open with production-ready defaults
db, err := sqlite.Open("app.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Query
var name string
err = db.QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", 1).Scan(&name)
```

### Why Zero CGO?

**modernc.org/sqlite** is a pure Go translation of SQLite's C code. Unlike `mattn/go-sqlite3`, it requires no `gcc` compilation and produces **cross-platform binaries that work everywhere** - perfect for CLI tools, lambdas, and containers.

---

## Connection

### Opening a Database

#### Using Default Settings

```go
db, err := sqlite.Open("app.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

#### Using Custom Configuration

```go
db, err := sqlite.OpenWithConfig(ctx, sqlite.Config{
    Path:            "app.db",
    WALMode:         true,
    BusyTimeout:     10 * time.Second,
    MaxOpenConns:    50,
    MaxIdleConns:    10,
    ConnMaxLifetime: 2 * time.Hour,
    ForeignKeys:     true,
})
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

#### In-Memory Database (Testing)

```go
db, err := sqlite.InMemory()
if err != nil {
    t.Fatal(err)
}
defer db.Close()
```

---

## Configuration

### Config Struct

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Path` | `string` | *required* | Database file path (`:memory:` for in-memory) |
| `WALMode` | `bool` | `true` | Enable Write-Ahead Logging for better concurrency |
| `BusyTimeout` | `time.Duration` | `5s` | How long to wait when database is locked |
| `MaxOpenConns` | `int` | `25` | Maximum number of open connections |
| `MaxIdleConns` | `int` | `5` | Maximum number of idle connections |
| `ConnMaxLifetime` | `time.Duration` | `1 hour` | Maximum connection reuse duration |
| `ForeignKeys` | `bool` | `true` | Enable foreign key constraints |

### Applied Pragmas

By default, GoKart applies production-optimized pragmas:

```sql
PRAGMA journal_mode=WAL;          -- Better concurrency (if WALMode=true)
PRAGMA synchronous=NORMAL;         -- Faster than FULL, still safe
PRAGMA busy_timeout=5000;          -- Wait 5s for locks
PRAGMA foreign_keys=ON;            -- Enable FK constraints
PRAGMA cache_size=-2000;           -- 2MB cache
PRAGMA temp_store=MEMORY;          -- Temp tables in memory
```

### Database Path Format

| Path | Description |
|------|-------------|
| `app.db` | Relative path to database file |
| `/var/data/app.db` | Absolute path to database file |
| `:memory:` | In-memory database (testing only) |
| `file:/path/to/db?_txlock=immediate` | Custom DSN options |

**Important:** For in-memory databases, **always** use `sqlite.InMemory()` helper. Direct connection to `:memory:` with multiple connections creates separate databases per connection.

---

## Transactions

### Transaction Helper

The `sqlite.Transaction` function provides automatic commit/rollback with panic recovery:

```go
err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
    // Insert user
    _, err := tx.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Alice")
    if err != nil {
        return err  // Automatically rolls back
    }

    // Update account
    _, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - ? WHERE user_id = ?", 100, 1)
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
- Panic occurs during transaction (caught, rolled back, re-panicked)
- Both error and rollback failure (returns compound error)

**Automatic commit on:**

- Function returns `nil`

**Panic recovery:**

```go
err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
    // Panic is caught, transaction rolled back, then re-panicked
    panic("critical failure")
})
```

### Manual Transactions

For custom isolation levels or manual control:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()  // Rolls back if Commit not reached

// ... perform operations ...

return tx.Commit()  // Commits if no error
```

---

## Querying

### Single Row

```go
var name string
var email string

err := db.QueryRowContext(ctx, "SELECT name, email FROM users WHERE id = ?", userID).
    Scan(&name, &email)
```

### Multiple Rows

```go
rows, err := db.QueryContext(ctx, "SELECT id, name FROM users WHERE active = 1")
if err != nil {
    return err
}
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
result, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Bob")

// UPDATE
result, err = db.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", true, userID)

// DELETE
result, err = db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", time.Now())

rowsAffected, _ := result.RowsAffected()
```

---

## Best Practices

### WAL Mode for Concurrency

**Always use WAL mode** for applications with concurrent writes:

```go
cfg.WALMode = true  // Default in GoKart
```

WAL mode allows readers to proceed without blocking writers:

- **No WAL**: Readers block writers, writers block readers
- **With WAL**: Readers don't block writers, writers don't block readers

### In-Memory Database Trap

**Critical:** Each connection to `:memory:` creates a separate database.

```go
// WRONG - Two separate databases that don't share data
db1, _ := sqlite.Open(":memory:")
db2, _ := sqlite.Open(":memory:")
```

```go
// RIGHT - Always use InMemory() helper (forces MaxOpenConns=1)
db, err := sqlite.InMemory()
```

### Connection Pool Sizing

For single-writer workloads (typical SQLite):

```go
cfg.MaxOpenConns = 1  // Prevents write contention
```

For read-heavy workloads:

```go
cfg.MaxOpenConns = 25  // Default in GoKart
```

### Context Timeouts

Always use context with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

db, err := sqlite.OpenContext(ctx, "app.db")
```

### Foreign Keys

Foreign key constraints are **disabled by default in SQLite**. GoKart enables them:

```go
cfg.ForeignKeys = true  // Default in GoKart
```

---

## Type Mapping

| Go Type | SQLite Type |
|---------|-------------|
| `int8`, `int16`, `int32`, `int64`, `int` | `INTEGER` |
| `uint8`, `uint16`, `uint32`, `uint64`, `uint` | `INTEGER` (truncated to signed) |
| `float32`, `float64` | `REAL` |
| `bool` | `INTEGER` (0/1) |
| `string` | `TEXT` |
| `time.Time` | `TEXT` (ISO8601) or `INTEGER` (Unix timestamp) |
| `[]byte` | `BLOB` |

### JSON Handling

SQLite has no native JSON type. Use `TEXT` with JSON functions:

```go
type User struct {
    ID       int    `db:"id"`
    Name     string `db:"name"`
    Metadata string `db:"metadata"` // JSON as TEXT
}

// Query with JSON extraction
var metadata string
err := db.QueryRowContext(ctx,
    "SELECT json_extract(metadata, '$.role') FROM users WHERE id = ?",
    userID,
).Scan(&metadata)
```

---

## Reference

### Functions

| Function | Description |
|----------|-------------|
| [`Open`](#opening-a-database) | Opens database with default settings |
| [`OpenContext`](#opening-a-database) | Opens database with context |
| [`OpenWithConfig`](#opening-a-database) | Opens database with custom config |
| [`InMemory`](#in-memory-database-testing) | Creates in-memory database for testing |
| [`DefaultConfig`](#config-struct) | Returns default configuration |
| [`Transaction`](#transaction-helper) | Executes function in transaction |

### See Also

- [modernc.org/sqlite documentation](https://pkg.go.dev/modernc.org/sqlite)
- [SQLite WAL mode](https://www.sqlite.org/wal.html)
- [SQLite pragma reference](https://www.sqlite.org/pragma.html)
- [Go database/sql package](https://pkg.go.dev/database/sql)
- [Migrations](/components/migrate) - Database schema versioning
- [PostgreSQL](/components/postgres) - Production database alternative
