# Connect to PostgreSQL

```go
pool, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}
defer pool.Close()

var name string
err = pool.QueryRow(ctx, "select name from users where id=$1", id).Scan(&name)
```

The PostgreSQL module configures a real `*pgxpool.Pool`. Use pgx directly after construction.

## Install

```bash
go get github.com/dotcommander/gokart/postgres@v0.13.0
```

## Choose a constructor

| API | Behavior |
|---|---|
| `Open(ctx, url)` | Applies production defaults, creates a pool, and pings it. |
| `OpenWithConfig(ctx, cfg)` | Applies explicit connection and pool settings, then pings. |
| `FromEnv(ctx)` | Lets pgx read its normal PostgreSQL environment variables, then pings. |
| `Config.NewPool(ctx)` | Creates the pool without a startup ping. |

```go
cfg := postgres.DefaultConfig("postgres://app:secret@localhost:5432/app?sslmode=disable")
cfg.MaxConns = 40
cfg.MinConns = 4
pool, err := postgres.OpenWithConfig(ctx, cfg)
```

## Configure the connection

`Config.DSN` selects `URL`, then the deprecated `ConnectionString`, then constructs a URL from `Host`, `Port`, `User`, `Password`, `DBName`, and `SSLMode`. Passwords and database names are URL-escaped.

Pool defaults are 25 maximum connections, 5 minimum connections, a one-hour maximum lifetime, a 30-minute maximum idle time, and a one-minute health-check period. Zero or negative pool values use these defaults.

`BuildConnectionString` is the compatibility name for `DSN`.

## Run a transaction

```go
err := postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
    _, err := tx.Exec(ctx, "update accounts set balance=balance-$1 where id=$2", amount, id)
    return err
})
```

The helper commits on success and rolls back on an error or panic. Nil pools and callbacks return errors.

## Build safe configured identifiers

```go
table, err := postgres.NewPostgresIdentifier("users")
index, err := postgres.NewPostgresIndexIdentifier("users", "email")
fmt.Println(table.Quoted, index.Quoted)
```

Identifiers accept ASCII letters, digits after the first character, and underscores. Generated index names are deterministically shortened to PostgreSQL's 63-byte limit.

## See also

- [Migrations](migrate.md)
- [SQLite](sqlite.md)
- [Root configuration](../api/gokart.md)
