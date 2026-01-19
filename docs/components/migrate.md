# Migrations

Database schema versioning built on [pressly/goose](https://github.com/pressly/goose/v3). Supports file-based and embedded migrations with automatic rollback and status tracking.

## Installation

```bash
go get github.com/dotcommander/gokart
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

// Run all pending migrations
err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})

// Or use convenience helpers
err := gokart.PostgresMigrate(ctx, db, "migrations")
err := gokart.SQLiteMigrate(ctx, db, "migrations")
```

---

## Migration Functions

### Running Migrations

#### Migrate

Runs all pending migrations.

```go
err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

#### MigrateUp

Alias for `Migrate`. Runs all pending migrations.

```go
err := gokart.MigrateUp(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

#### MigrateDown

Rolls back the most recently applied migration.

```go
err := gokart.MigrateDown(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

#### MigrateDownTo

Rolls back to a specific version.

```go
// Rollback to version 20230101000000
err := gokart.MigrateDownTo(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
}, 20230101000000)
```

#### MigrateReset

Rolls back all migrations (equivalent to `MigrateDownTo` version 0).

```go
err := gokart.MigrateReset(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

#### MigrateStatus

Prints the status of all migrations to stdout.

```go
err := gokart.MigrateStatus(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

Output format:
```
    Applied At                  Migration
    =======================================
    Sun Jan 1 10:00:00 2024    20230101000000_init.sql
    Sun Jan 2 11:30:00 2024    20230102113000_users.sql
    Pending                    20230103120000_posts.sql
```

#### MigrateVersion

Returns the current migration version.

```go
version, err := gokart.MigrateVersion(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
fmt.Printf("Current version: %d\n", version)
```

---

## Creating Migrations

### MigrateCreate

Creates a new migration file with up/down SQL templates.

```go
// Creates: migrations/20240101120000_add_users_table.sql
err := gokart.MigrateCreate("migrations", "add_users_table", "sql")
```

The generated file includes both up and down migrations:

```sql
-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- +goose Down
DROP TABLE users;
```

---

## Embedded Migrations

### Using embed.FS

Bundle migrations with your binary using Go's embed directive:

```go
package main

import (
    "embed"
    "github.com/dotcommander/gokart"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
    ctx := context.Background()

    err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
        FS:      migrations,
        Dir:     "migrations",
        Dialect: "postgres",
    })
}
```

**Benefits of embedded migrations:**
- Single binary deployment
- No missing migration files
- Version-migration consistency
- Simplified production deployments

---

## Configuration

### MigrateConfig Struct

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Dir` | `string` | `"migrations"` | Directory containing migration files |
| `Table` | `string` | `"goose_db_version"` | Migration tracking table name |
| `Dialect` | `string` | *auto-detected* | Database dialect (`postgres`, `sqlite3`, `mysql`) |
| `FS` | `fs.FS` | `nil` | Optional filesystem for embedded migrations |
| `AllowMissing` | `bool` | `false` | Allow applying out-of-order migrations |
| `NoVersioning` | `bool` | `false` | Disable version tracking (one-off scripts) |

### DefaultMigrateConfig

Returns sensible defaults for migration configuration:

```go
cfg := gokart.DefaultMigrateConfig()
// Returns: MigrateConfig{Dir: "migrations", Table: "goose_db_version"}
```

### Custom Table Name

Override the default migration tracking table:

```go
err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
    Table:   "schema_migrations",  // Custom tracking table
})
```

---

## Database-Specific Helpers

### PostgresMigrate

Convenience function for PostgreSQL migrations:

```go
pool, _ := gokart.OpenPostgres(ctx, url)
db := stdlib.OpenDBFromPool(pool)

err := gokart.PostgresMigrate(ctx, db, "migrations")
```

Equivalent to:

```go
err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

### SQLiteMigrate

Convenience function for SQLite migrations:

```go
db, _ := gokart.OpenSQLite("app.db")

err := gokart.SQLiteMigrate(ctx, db, "migrations")
```

Equivalent to:

```go
err := gokart.Migrate(ctx, db, gokart.MigrateConfig{
    Dir:     "migrations",
    Dialect: "sqlite3",
})
```

---

## Migration File Format

### Naming Convention

Migration files must follow the naming pattern:

```
<version>_<name>_<type>.<ext>
```

**Components:**

| Part | Example | Description |
|------|---------|-------------|
| `version` | `20240101120000` | Timestamp in `YYYYMMDDHHMMSS` format |
| `name` | `add_users_table` | Descriptive migration name (snake_case) |
| `type` | `sql` or `go` | Migration language |
| `ext` | `.sql` or `.go` | File extension |

**Examples:**

```bash
20240101120000_init.sql
20240102140000_add_users_table.sql
20240103160000_create_posts.go
```

### SQL Migrations

Use `-- +goose Up` and `-- +goose Down` markers:

```sql
-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);

-- +goose Down
DROP INDEX idx_users_username;
DROP TABLE users;
```

### Go Migrations

For complex logic, use Go migrations:

```go
-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL here'
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL here'
-- +goose StatementEnd
```

Then implement in Go:

```go
//go:build goose
// +build goose

package migrations

import (
    "database/sql"
    "github.com/pressly/goose/v3"
)

func init() {
    goose.AddMigration(Up, Down)
}

func Up(tx *sql.Tx) error {
    _, err := tx.Exec(`
        CREATE TABLE users (
            id SERIAL PRIMARY KEY,
            data JSONB NOT NULL
        )
    `)
    return err
}

func Down(tx *sql.Tx) error {
    _, err := tx.Exec(`DROP TABLE users`)
    return err
}
```

---

## Best Practices

### Migration Design

**Keep migrations idempotent:**

```sql
-- Good: Uses IF NOT EXISTS
CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY);

-- Avoid: Fails if table exists
CREATE TABLE users (id SERIAL PRIMARY KEY);
```

**Avoid data-dependent rollbacks:**

```sql
-- Up: Safe
ALTER TABLE users ADD COLUMN bio TEXT;

-- Down: Unsafe (loses existing data)
ALTER TABLE users DROP COLUMN bio;

-- Better: Use NULLABLE and clean up later
ALTER TABLE users ADD COLUMN bio TEXT;
```

**Index separately from tables:**

```sql
-- +goose Up
CREATE TABLE posts (id SERIAL PRIMARY KEY, slug TEXT NOT NULL);
CREATE INDEX idx_posts_slug ON posts(slug);

-- +goose Down
DROP INDEX idx_posts_slug;
DROP TABLE posts;
```

### Production Safety

**Test rollbacks locally:**

```bash
# Apply migration
goose up "postgres://localhost/mydb?sslmode=disable" ./migrations

# Verify application works

# Rollback
goose down "postgres://localhost/mydb?sslmode=disable" ./migrations
```

**Use transactions for multi-step migrations:**

```sql
-- +goose Up
BEGIN;
CREATE TABLE users (id SERIAL PRIMARY KEY);
CREATE TABLE posts (user_id INT REFERENCES users(id));
COMMIT;
```

**Zero-downtime deployments:**

1. Deploy code compatible with old schema
2. Run migration (additive changes only)
3. Deploy code using new schema
4. Clean up old columns in separate migration

### Version Management

**Use timestamps for ordering:**

```bash
# Generate with date command
TIMESTAMP=$(date +%Y%m%d%H%M%S)
goose create add_users_table sql
mv migrations/*_add_users_table.sql migrations/${TIMESTAMP}_add_users_table.sql
```

**Never modify existing migrations:**

- If a mistake is in a committed migration, create a new one to fix it
- Modifying applied migrations breaks consistency across environments

---

## Reference

### Functions

| Function | Description |
|----------|-------------|
| [`Migrate`](#migrate) | Runs all pending migrations |
| [`MigrateUp`](#migrateup) | Alias for `Migrate` |
| [`MigrateDown`](#migratedown) | Rolls back the last migration |
| [`MigrateDownTo`](#migratedownto) | Rolls back to a specific version |
| [`MigrateReset`](#migratereset) | Rolls back all migrations |
| [`MigrateStatus`](#migratestatus) | Prints migration status |
| [`MigrateVersion`](#migrateversion) | Returns current version |
| [`MigrateCreate`](#migratecreate) | Creates a new migration file |
| [`PostgresMigrate`](#postgresmigrate) | PostgreSQL convenience helper |
| [`SQLiteMigrate`](#sqlitemigrate) | SQLite convenience helper |
| [`DefaultMigrateConfig`](#defaultmigrateconfig) | Returns default configuration |

### Types

| Type | Description |
|------|-------------|
| [`MigrateConfig`](#migrateconfig-struct) | Migration configuration options |

### See Also

- [goose documentation](https://github.com/pressly/goose)
- [PostgreSQL](/components/postgres) - PostgreSQL integration
- [SQLite](/components/sqlite) - SQLite integration
