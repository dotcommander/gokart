# GoKart

![GoKart logo](logo.png)

GoKart is a modular Go toolkit for recurring infrastructure setup, safe defaults, and user-owned generated code. Its enforceable boundary is defined in [PHILOSOPHY.md](PHILOSOPHY.md).

Install the tagged CLI and scaffold a project:

```bash
go install github.com/dotcommander/gokart/cmd/gokart@v0.13.0
gokart new tvguide --example
cd tvguide
go run . greet --name World
go build -o tvguide .
```

Continue with the [TV-guide tutorial](docs/getting-started.md) to turn the
generated command into a tested, filterable program listing.

## Packages

- `gokart`: typed configuration, platform config directories, and JSON state persistence.
- `gokart/cli`: Cobra application construction for established ecosystem-style consumers plus process-stream presentation helpers. Focused generated CLIs use Kong directly.
- `gokart/web`: chi router/server construction, JSON responses, bounded binding, and validation.
- `gokart/postgres`: pgx pool setup and transaction helpers.
- `gokart/sqlite`: zero-CGO SQLite setup and operations.
- `gokart/migrate`: goose migrations.
- `gokart/cache`: Redis construction, prefixes, JSON operations, and `Remember`.
- `gokart/logger`: `log/slog` setup.

All modules isolate their dependencies. Constructors expose real upstream types or an explicit `Client` escape hatch.

## Generator

`gokart new` creates ordinary Go code. Plain, example, local, and global CLIs default to a flat `main.go`; selecting an integration chooses the structured layout automatically. Use `--structured` when you want the multi-package layout without an integration.

```bash
gokart new mycli
gokart new mycli --structured
gokart new mycli --structured --global
gokart new service --db postgres --ai --redis
gokart add sqlite --dry-run
```

Flat scaffolds are unmanaged and manifest-free. Structured scaffolds are managed
by default and support later integration updates through `gokart add`.

Dependencies are pinned for deterministic generation. PostgreSQL uses `postgres.Open`, SQLite uses `sqlite.Open`, Redis uses `cache.Open`, and AI uses the official OpenAI SDK directly.

See the [generator reference](docs/components/generator.md) and [documentation index](docs/index.md).

## v0.11 migration

| Removed surface | Replacement |
|---|---|
| `ai.NewOpenAIClient(opts...)` | `openai.NewClient(opts...)` |
| `ai.NewOpenAIClientWithKey(key)` | `openai.NewClient(option.WithAPIKey(key))` |
| `fs.ConfigDir` / `fs.EnsureConfigDir` | `gokart.ConfigDir` / `gokart.EnsureConfigDir` |
| `fs.WriteFile` / `fs.ReadOrCreate` | Standard library; no GoKart replacement |
| `GetString`, `GetInt`, `GetFloat`, `GetBool` | Typed configuration parsing or caller-owned assertions |
| Cache command mirrors | `c.Client().Command(ctx, c.Key(key), ...)` |
| `cli.Fatal`, `cli.FatalErr`, `cli.Must` | Return errors; `main` owns `os.Exit` |
| CLI writer overrides | Cobra `SetOut` / `SetErr` and command writers |
| Removed web helpers | Standard library or the named upstream package directly; see `web/README.md` |

No deprecated aliases or forwarding modules are retained. Historical `v0.10.3` tags are the compatibility path.

### Removed identifiers

Use this searchable inventory when migrating:

- Root getters: `GetString`, `GetInt`, `GetFloat`, `GetBool`.
- Cache commands: `Get`, `Set`, `Delete`, `Exists`, `Expire`, `TTL`, `Incr`, `IncrBy`, `SetNX`, `HGet`, `HSet`, `HGetAll`, `HDel`, `HIncrBy`, `ZAdd`, `ZRange`, `ZRangeByScore`, `ZScore`, `ZRem`, `ZCard`, `SAdd`, `SRem`, `SMembers`, `SIsMember`, `LPush`, `RPush`, `LRange`, `LPop`, `RPop`, `Decr`, and `DecrBy`. Call the corresponding go-redis method through `Client`, applying `Key` to every logical key.
- CLI process control and writers: `Fatal`, `FatalErr`, `Must`, `SetOutput`, `SetErrOutput`, `Output`, and `ErrOutput`.
- Web assets and auth: `NewAssets`, `AssetConfig`, `Assets`, `Assets.Path`, `Assets.Handler`, `APIKeyAuth`, and `BearerAuth`.
- Web CSRF and flash: `CSRFProtect`, `CSRFProtectWithOrigins`, `SetFlash`, `GetFlash`, `FlashFromContext`, `FlashMiddleware`, `FlashLevel`, `FlashMessage`, `FlashSuccess`, `FlashError`, `FlashWarning`, and `FlashInfo`.
- Web health and clients: `HealthHandler`, `ReadyHandler`, `HealthCheck`, `HealthFunc`, `NewHTTPClient`, `NewStandardClient`, and `HTTPConfig`.
- Web negotiation and pagination: `WantsJSON`, `IsHTMX`, `Negotiate`, `NegotiateStatus`, `ParsePage`, `ParsePageWithConfig`, `NewPagedResponse`, `Page`, `PageConfig`, and `PagedResponse`.
- Web rate limiting: `RateLimit`, `RateLimitWithKey`, `RateLimitWithEviction`, `WithTTL`, `WithSweepInterval`, `RateLimiter`, `RateLimiter.Middleware`, `RateLimiter.LimiterCount`, `RateLimiter.Stop`, and `RateLimitOption`.
- Web templ adapters: `Render`, `RenderCtx`, `RenderWithStatus`, `TemplHandler`, `TemplHandlerFunc`, and `TemplHandlerFuncE`.

## Verification

Go 1.26 or later is required.

```bash
just verify
```

## License

[MIT](LICENSE)
