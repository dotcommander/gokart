# Recipes

```go
cmd := &cobra.Command{Use: "list", RunE: func(cmd *cobra.Command, _ []string) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "no items")
	return err
}}
```

Use command-owned writers so tests can capture output. Keep parsing and presentation in commands; pass typed input to actions or services.

## Commands, flags, and configuration

Create a command constructor in `internal/commands`, then register it with `cliApp.AddCommand`. Bind flags on that command:

```go
limit := 20
cmd.Flags().IntVarP(&limit, "limit", "n", 20, "maximum rows")
```

Generated configuration comes from the `*viper.Viper` passed to `app.New`:

```go
v.SetDefault(app.AppConfigKeyDBPath, "notes.db")
path := v.GetString(app.AppConfigKeyDBPath)
```

The generated environment prefix maps `NOTES_DB_PATH` to `db_path`. `--config` comes from `WithStandardFlags`. For standalone typed files, use `gokart.LoadConfigWithDefaults(defaults, "config.yaml")`.

## SQLite transaction and migrations

```go
err := sqlite.Transaction(ctx, db, func(tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO notes(body) VALUES (?)`, body)
	return err
})
```

`sqlite.Open` returns `*sql.DB`; use its normal query methods and close it from the owning context. Apply migrations with:

```go
err := migrate.Up(ctx, db, migrate.Config{Dir: "migrations", Dialect: "sqlite3"})
```

For PostgreSQL use dialect `postgres`. Set `FS` to an `embed.FS` to ship migrations in the binary.

## PostgreSQL transaction

```go
err := postgres.Transaction(ctx, pool, func(tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `INSERT INTO notes(body) VALUES ($1)`, body)
	return err
})
```

The generated project reads `database_url`, then `DATABASE_URL`. `postgres.Open` returns `*pgxpool.Pool`, so query with pgx and close the pool during shutdown.

## Redis with direct client access

```go
c, err := cache.Open(ctx, os.Getenv("REDIS_ADDR"))
if err != nil { return err }
defer c.Close()
value, err := c.Client().Get(ctx, c.Key("session:42")).Result()
if err != nil && !cache.IsNil(err) { return err }
```

The generated integration reads `redis_addr`, then `REDIS_ADDR`. `Client` exposes go-redis; `Key` applies the configured prefix.

## Bounded HTTP input and validation

```go
var input CreateNote
if err := web.BindJSONWithLimit(r, &input, 1<<20); err != nil {
	web.Error(w, http.StatusBadRequest, err.Error())
	return
}
```

Use `web.BindAndValidate` with a `*validator.Validate` when validation is also required. Choose an application-appropriate body limit.

## Logging and state

```go
log := logger.New(logger.Config{Level: "info", Format: "json"})
log.Info("note created", "id", id)

type State struct { LastNoteID int64 `json:"last_note_id"` }
err := gokart.SaveState("notes", "state.json", State{LastNoteID: id})
```

State lives under the platform user configuration directory. Use a database for concurrent or queryable domain data.

## Graceful shutdown

```go
err := web.ListenAndServeWithTimeout(":8080", router, 10*time.Second)
```

The helper listens for interrupt or termination signals and bounds shutdown. See the [full-service example](examples/README.md#service-composition).

## Add integrations

```bash
gokart add postgres redis --dry-run
gokart add postgres redis
```

Valid names are `sqlite`, `postgres`, `ai`, and `redis`. AI reads `openai_api_key`, falls back to `OPENAI_API_KEY`, and constructs the official `openai-go/v3` client. Read [generated-code ownership](generated-code.md) before editing rewrite targets.
