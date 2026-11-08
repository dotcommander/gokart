# GoKart migration

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
- CLI presentation: `Success`, `Info`, `Warning`, `Error`, `Dim`, `Bold`, `NewSpinner`, `WithSpinner`, `NewProgress`, `NewTable`, `SimpleTable`, `KeyValue`, `List`, `NumberedList`, and `SetStyledHelp`. Use lipgloss, bubbles, or lipgloss/table directly.
- Postgres compatibility aliases: `PostgresConfig` and `BuildConnectionString`. Use `Config` and `DSN`.
- Web assets and auth: `NewAssets`, `AssetConfig`, `Assets`, `Assets.Path`, `Assets.Handler`, `APIKeyAuth`, and `BearerAuth`.
- Web CSRF and flash: `CSRFProtect`, `CSRFProtectWithOrigins`, `SetFlash`, `GetFlash`, `FlashFromContext`, `FlashMiddleware`, `FlashLevel`, `FlashMessage`, `FlashSuccess`, `FlashError`, `FlashWarning`, and `FlashInfo`.
- Web health and clients: `HealthHandler`, `ReadyHandler`, `HealthCheck`, `HealthFunc`, `NewHTTPClient`, `NewStandardClient`, and `HTTPConfig`.
- Web negotiation and pagination: `WantsJSON`, `IsHTMX`, `Negotiate`, `NegotiateStatus`, `ParsePage`, `ParsePageWithConfig`, `NewPagedResponse`, `Page`, `PageConfig`, and `PagedResponse`.
- Web rate limiting: `RateLimit`, `RateLimitWithKey`, `RateLimitWithEviction`, `WithTTL`, `WithSweepInterval`, `RateLimiter`, `RateLimiter.Middleware`, `RateLimiter.LimiterCount`, `RateLimiter.Stop`, and `RateLimitOption`.
- Web templ adapters: `Render`, `RenderCtx`, `RenderWithStatus`, `TemplHandler`, `TemplHandlerFunc`, and `TemplHandlerFuncE`.
