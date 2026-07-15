# Cache JSON in Redis

```go
c, err := cache.Open(ctx, "localhost:6379")
if err != nil {
    return err
}
defer c.Close()

if err := c.SetJSON(ctx, "user:1", user, time.Hour); err != nil {
    return err
}
err = c.GetJSON(ctx, "user:1", &user)
```

The cache module owns Redis construction, optional key prefixes, JSON operations, and stampede-resistant `Remember` helpers. Ordinary Redis commands stay owned by go-redis.

## Install

```bash
go get github.com/dotcommander/gokart/cache@v0.12.0
```

## Connect

```go
c, err := cache.OpenURLWithPrefix(ctx, "redis://:secret@localhost:6379/0", "myapp:")
```

| API | Behavior |
|---|---|
| `Open(ctx, addr)` | Uses `DefaultConfig` with the supplied address and pings Redis. |
| `OpenURL(ctx, url)` | Parses a Redis URL, creates a client without a prefix, and pings. |
| `OpenURLWithPrefix(ctx, url, prefix)` | Adds a namespace to URL construction. |
| `OpenWithConfig(ctx, cfg)` | Uses URL or discrete connection/pool settings and an optional prefix. |

Default discrete settings are `localhost:6379`, database 0, pool size 10, 2 idle connections, a 5-second dial timeout, and 3-second read/write timeouts.

## Use ordinary Redis commands

```go
err := c.Client().Set(ctx, c.Key("greeting"), "hello", time.Hour).Err()
value, err := c.Client().Get(ctx, c.Key("greeting")).Result()
```

`Client` returns the real `*redis.Client`. Always pass logical keys through `Key` so configured prefixes remain effective.

## Remember computed values

```go
value, err := c.Remember(ctx, "report:daily", time.Hour, func() (interface{}, error) {
    return buildReport(ctx)
})
```

`Remember` checks Redis, collapses concurrent misses in this process with `singleflight`, computes once, and stores the string representation. `RememberJSON` performs the same pattern for JSON and unmarshals into the destination.

Use `IsNil(err)` to recognize a go-redis cache miss.

## Migration from v0.10

Command mirrors such as `Get`, `Set`, `Delete`, hashes, lists, sets, sorted sets, counters, expiry, and locks were removed. Call the real client with `c.Key(key)`.

## See also

- [Generator](generator.md)
- [Root configuration](../api/gokart.md)
