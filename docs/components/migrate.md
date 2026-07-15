# Run database migrations

```go
err := migrate.Up(ctx, db, migrate.Config{
    Dir:     "migrations",
    Dialect: "postgres",
})
```

The migration module uses goose's provider API with `*sql.DB`.

## Install

```bash
go get github.com/dotcommander/gokart/migrate@v0.13.0
```

## Configure migrations

| Field | Default | Behavior |
|---|---|---|
| `Dir` | `migrations` | Directory inside the OS or embedded filesystem. |
| `Table` | `goose_db_version` from `DefaultConfig` | Version table name. |
| `Dialect` | none | Required: for example `postgres`, `sqlite3`, or `mysql`. |
| `FS` | OS filesystem | Optional embedded filesystem; `Dir` is opened as a subdirectory. |
| `AllowMissing` | false | Allows out-of-order migrations. |
| `NoVersioning` | false | Disables the version table for one-off scripts. |

The dialect is never auto-detected.

## Apply and roll back

```go
cfg := migrate.Config{Dir: "migrations", Dialect: "postgres"}

err := migrate.Up(ctx, db, cfg)
err = migrate.Down(ctx, db, cfg)
err = migrate.DownTo(ctx, db, cfg, 20260712010101)
err = migrate.Reset(ctx, db, cfg)
```

`migrate.Postgres(ctx, db, dir)` and `migrate.SQLite(ctx, db, dir)` are shortcuts for `Up` with the appropriate dialect.

## Inspect status

```go
statuses, err := migrate.MigrationStatuses(ctx, db, cfg)
for _, status := range statuses {
    fmt.Println(status.Version, status.Applied, status.AppliedAt)
}

version, err := migrate.Version(ctx, db, cfg)
```

`Status` only verifies that status can be loaded and returns an error. It does not print. Use `MigrationStatuses` to render results yourself.

## Embed migrations

```go
//go:embed migrations/*.sql
var files embed.FS

err := migrate.Up(ctx, db, migrate.Config{
    FS: files, Dir: "migrations", Dialect: "sqlite3",
})
```

## Create a migration

```go
err := migrate.Create("migrations", "add_users", "sql")
```

An empty directory defaults to `migrations`; an empty type defaults to `sql`.

## See also

- [PostgreSQL](postgres.md)
- [SQLite](sqlite.md)
