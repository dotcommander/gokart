# Open SQLite

```go
db, err := sqlite.Open("app.db")
if err != nil {
    return err
}
defer db.Close()

_, err = db.ExecContext(ctx, "create table if not exists users(id integer primary key, name text)")
```

The SQLite module uses the pure-Go `modernc.org/sqlite` driver and returns a standard `*sql.DB`.

## Install

```bash
go get github.com/dotcommander/gokart/sqlite@v0.16.0
```

## Choose a mode

| API | Use |
|---|---|
| `Open(path)` / `OpenContext(ctx, path)` | Read-write database with WAL defaults. |
| `InMemory()` / `InMemoryContext(ctx)` | One-connection in-memory database. |
| `OpenReadOnly(ctx, path)` | Read-only database. |
| `OpenImmutable(ctx, path)` | Read-only file that SQLite may treat as immutable. |
| `OpenWithConfig(ctx, cfg)` | Explicit mode, pragmas, cache, mmap, and pool settings. |

## Understand defaults

`DefaultConfig(path)` enables foreign keys, a five-second busy timeout, a 2,000 KiB cache, one open connection, one idle connection, and a one-hour connection lifetime. Read-write files use WAL with normal (`NORMAL`) synchronization and immediate transaction locking. `NORMAL` is the default when `Synchronous` is empty, but it is **not** the safe default for every store — see “Durability” below.

```go
cfg := sqlite.ReadHeavyConfig("analytics.db")
effective, err := sqlite.ResolveConfig(cfg)
db, err := sqlite.OpenWithConfig(ctx, cfg)
```

`ReadHeavyConfig` uses a 20,000 KiB cache, a 30 GB mmap limit, 10 open connections, and 5 idle connections. `ReadOnlyConfig` and `ImmutableConfig` are templates; set `Path` before opening.

`Config` exposes `Path`, `Mode`, `WALMode`, `JournalMode`, `Synchronous`, `BusyTimeout`, `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`, `ForeignKeys`, `CacheSizeKB`, and `MmapSizeBytes`. `ResolveConfig` validates conflicting modes and returns the effective values.

## Durability

`Synchronous` controls how much a power loss can undo:

| Mode | Meaning | Use when |
|---|---|---|
| `NORMAL` (default with WAL) | A crash can roll back recent *accepted* transactions. | Recovery/rebuild covers that window (cache, derived data). |
| `FULL` | Power-loss durability of acknowledged writes. | Authoritative data you cannot rebuild. |
| `OFF` | No durability guarantee. | Never for authoritative data. |

`ResolveConfig` silently resolves WAL + empty `Synchronous` to `NORMAL`. For authoritative stores, set it explicitly:

```go
cfg := sqlite.DefaultConfig("app.db")
cfg.Synchronous = sqlite.SynchronousFull
db, err := sqlite.OpenWithConfig(ctx, cfg)
```

Verify the effective settings through the application's connection (`PRAGMA synchronous`, `PRAGMA journal_mode`, `PRAGMA foreign_keys`) — a requested `Synchronous` is not evidence the VFS accepted it.

## Run transactions and savepoints

```go
err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, "insert into users(name) values (?)", "Ada")
    return err
})

err = sqlite.Savepoint(ctx, tx, "optional_step", func() error {
    return doOptionalWork(ctx, tx)
})
```

`TransactionWithOptions` accepts `*sql.TxOptions`. Transaction helpers commit on success and roll back on an error or panic.

## Check and maintain a database

```go
quick, err := sqlite.QuickCheck(ctx, db)
integrity, err := sqlite.IntegrityCheck(ctx, db)
foreignKeys, err := sqlite.ForeignKeyCheck(ctx, db)
stats, err := sqlite.Inspect(ctx, db, "app.db")
```

Maintenance APIs include `VacuumInto`, `Backup`, and `WALCheckpoint`. For plain `PRAGMA optimize` or `VACUUM`, call `db.ExecContext` directly. `BackupOptions{Overwrite:true}` permits replacing the destination.

## Retry lock contention

```go
err := sqlite.Retry(ctx, sqlite.RetryPolicy{
    Attempts: 5,
    Delay:    25 * time.Millisecond,
}, func() error {
    _, err := db.ExecContext(ctx, query)
    return err
})
```

Use `IsBusy`, `IsLocked`, and `IsConstraint` to classify SQLite primary result codes. `Retry` only retries busy or locked failures and respects context cancellation.

## See also

- [Migrations](migrate.md)
- [PostgreSQL](postgres.md)
