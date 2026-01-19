# Cache

Production-ready Redis client with convenient key-value operations and Remember pattern for automatic cache-population. Built on [go-redis/v9](https://github.com/redis/go-redis).

## Installation

```bash
go get github.com/dotcommander/gokart
```

## Quick Start

```go
import "github.com/dotcommander/gokart"

// Open with default settings
cache, err := gokart.OpenCache(ctx, "localhost:6379")
if err != nil {
    log.Fatal(err)
}
defer cache.Close()

// Set and get
err = cache.Set(ctx, "greeting", "Hello, World!", time.Hour)
val, _ := cache.Get(ctx, "greeting")
// val == "Hello, World!"
```

---

## Connection

### Opening a Cache

#### Using Default Settings

```go
cache, err := gokart.OpenCache(ctx, "localhost:6379")
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

#### Using Connection URL

```go
cache, err := gokart.OpenCacheURL(ctx, "redis://:password@localhost:6379/0")
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

#### Using Custom Configuration

```go
cache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
    Addr:         "localhost:6379",
    Password:     "secret",
    DB:           0,
    PoolSize:     20,
    MinIdleConns: 5,
    KeyPrefix:    "myapp:",
})
if err != nil {
    log.Fatal(err)
}
defer cache.Close()
```

---

## Configuration

### CacheConfig Struct

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `URL` | `string` | *empty* | Redis connection URL (alternative to Addr) |
| `Addr` | `string` | `localhost:6379` | Redis server address |
| `Password` | `string` | *empty* | Password for authentication |
| `DB` | `int` | `0` | Redis database number |
| `PoolSize` | `int` | `10` | Maximum number of connections |
| `MinIdleConns` | `int` | `2` | Minimum number of idle connections |
| `DialTimeout` | `time.Duration` | `5s` | Timeout for establishing connections |
| `ReadTimeout` | `time.Duration` | `3s` | Timeout for socket reads |
| `WriteTimeout` | `time.Duration` | `3s` | Timeout for socket writes |
| `KeyPrefix` | `string` | *empty* | Prefix prepended to all keys |

### Connection URL Format

```
redis://[:password@]host:port[/db]
```

**Components:**

| Part | Example | Description |
|------|---------|-------------|
| `password` | `secret123` | Authentication password (optional) |
| `host` | `localhost` | Redis host |
| `port` | `6379` | Redis port (default: 6379) |
| `db` | `0` | Database number 0-15 (default: 0) |

**Common URL Examples:**

```bash
# Local development
redis://localhost:6379

# With authentication
redis://:mypassword@localhost:6379

# Specific database
redis://localhost:6379/2

# Production with auth and database
redis://:prod_pass@redis.example.com:6379/1
```

### Environment Variables

```bash
# Structured config (for custom config loading)
export CACHE_ADDR="localhost:6379"
export CACHE_PASSWORD="secret"
export CACHE_DB="0"
```

When using with [`gokart.LoadConfig`](/components/config/), env vars bind automatically:

```go
type Config struct {
    CacheAddr     string        `env:"CACHE_ADDR"`
    CachePassword string        `env:"CACHE_PASSWORD"`
    CacheDB       int           `env:"CACHE_DB"`
    CacheTTL      time.Duration `env:"CACHE_TTL"`
}
```

---

## Basic Operations

### Set and Get

```go
// Store a string value
err := cache.Set(ctx, "user:123:name", "Alice", time.Hour)

// Retrieve a string value
name, err := cache.Get(ctx, "user:123:name")
```

### JSON Operations

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Store as JSON
user := User{ID: 123, Name: "Alice", Email: "alice@example.com"}
err := cache.SetJSON(ctx, "user:123", user, time.Hour)

// Retrieve and unmarshal
var retrieved User
err := cache.GetJSON(ctx, "user:123", &retrieved)
```

### Delete

```go
// Delete single key
err := cache.Delete(ctx, "user:123")

// Delete multiple keys
err := cache.Delete(ctx, "user:123", "user:456", "session:abc")
```

### Existence Check

```go
exists, err := cache.Exists(ctx, "user:123")
if exists {
    // Key exists
}
```

---

## Remember Pattern

The Remember pattern implements "get-or-compute" caching: retrieve cached value if present, otherwise compute and cache it.

### Remember

`Remember` retrieves a string value or computes it using the provided function. Returns a `string`.

```go
// Cache a computed string value
greeting, err := cache.Remember(ctx, "greeting", time.Hour, func() (interface{}, error) {
    // Computed only on cache miss
    return "Hello, World!", nil
})

// Cache a database query result (converted to JSON string)
userData, err := cache.Remember(ctx, "user:123", time.Hour, func() (interface{}, error) {
    // Computed only on cache miss
    user, err := db.GetUser(ctx, 123)
    if err != nil {
        return nil, err
    }
    return user, nil  // Automatically marshaled to JSON
})
```

**Return value:** Always returns `string`. If the computed value is not a string, it is JSON-encoded.

### RememberJSON

`RememberJSON` retrieves a JSON value or computes it. Preserves type information via the destination parameter.

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

var user User

err := cache.RememberJSON(ctx, "user:123", time.Hour, &user, func() (interface{}, error) {
    // Computed only on cache miss
    return db.GetUser(ctx, 123)
})
// user is populated, either from cache or database
```

**Return value:** Returns `error` only. The destination is populated on both hit and miss.

### Remember vs RememberJSON

| Pattern | Returns | Use When |
|---------|---------|----------|
| `Remember` | `(string, error)` | You need a string value or don't care about type |
| `RememberJSON` | `error` | You need a typed struct from cache |

**Choose `Remember` for:**
- Simple string values
- HTML fragments
- Raw data where type preservation isn't critical

**Choose `RememberJSON` for:**
- Structured data (API responses, database rows)
- Type-safe operations
- When you need to unmarshal into specific structs

### Cache Miss Behavior

On cache miss (`redis.Nil` error):
1. Function is called to compute the value
2. Result is stored in Redis with the specified TTL
3. Cached (or computed) value is returned

On function error:
- Value is **not** cached
- Error is returned immediately

### TTL in Remember

The TTL is applied **only on cache miss** when the value is computed. Existing cached values retain their original TTL.

```go
// On miss: cached with 1-hour TTL
// On hit: returned with whatever TTL remains
val, err := cache.Remember(ctx, "key", time.Hour, computeFn)
```

---

## Expiration (TTL)

### Setting TTL

TTL is set when storing values:

```go
// Expire after 1 hour
cache.Set(ctx, "key", "value", time.Hour)

// Expire after 5 minutes
cache.Set(ctx, "temp", "data", 5*time.Minute)

// No expiration (persistent)
cache.Set(ctx, "permanent", "data", 0)
```

### Checking TTL

```go
// Get remaining time-to-live
ttl, err := cache.TTL(ctx, "user:123")
if ttl > 0 {
    fmt.Printf("Key expires in %v\n", ttl)
} else if ttl == -2 {
    fmt.Println("Key does not exist")
} else if ttl == -1 {
    fmt.Println("Key exists but has no expiration")
}
```

### Updating TTL

```go
// Extend expiration of existing key
err := cache.Expire(ctx, "user:123", 2*time.Hour)
```

### TTL Return Values

| Value | Meaning |
|-------|---------|
| `> 0` | Time remaining until expiration |
| `-1` | Key exists but has no expiration |
| `-2` | Key does not exist |

---

## Advanced Operations

### Counters

```go
// Increment by 1
count, err := cache.Incr(ctx, "pageviews:home")

// Increment by custom amount
count, err := cache.IncrBy(ctx, "score:player1", 10)
```

### Distributed Locks (SetNX)

```go
// Acquire lock (only if key doesn't exist)
acquired, err := cache.SetNX(ctx, "lock:resource1", "owner1", 10*time.Second)
if acquired {
    // Lock acquired - do work
    defer cache.Delete(ctx, "lock:resource1")
} else {
    // Lock already held
}
```

### Accessing Underlying Client

```go
client := cache.Client()

// Use full go-redis API
result := client.ZAdd(ctx, "leaderboard", redis.Z{
    Score:  100,
    Member: "player1",
})
```

---

## Key Prefixing

Use `KeyPrefix` to namespace all cache keys:

```go
cache, _ := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
    Addr:      "localhost:6379",
    KeyPrefix: "myapp:v1:",
})

// All keys are automatically prefixed
cache.Set(ctx, "user:123", "data", time.Hour)
// Actual Redis key: "myapp:v1:user:123"

cache.Get(ctx, "user:123")
// Retrieves "myapp:v1:user:123"
```

**Benefits:**
- Prevents key collisions between applications
- Easy cache invalidation by prefix (use `SCAN` with `MATCH`)
- Multi-version deployment support

---

## Error Handling

### Cache Miss Detection

Use `gokart.IsNil` to detect cache misses:

```go
val, err := cache.Get(ctx, "user:123")
if gokart.IsNil(err) {
    // Key does not exist - cache miss
} else if err != nil {
    // Actual error (connection, timeout, etc.)
    log.Fatal(err)
} else {
    // Cache hit
    fmt.Println(val)
}
```

### Remember Error Handling

```go
val, err := cache.Remember(ctx, "key", time.Hour, func() (interface{}, error) {
    // Compute function errors are NOT cached
    return nil, errors.New("computation failed")
})
// err != nil, nothing written to cache
```

---

## Best Practices

### Connection Pool Sizing

```go
// For I/O-bound caching (many small gets)
cfg.PoolSize = 20

// For compute-intensive caching (large values)
cfg.PoolSize = 10
```

### TTL Selection

```go
// Hot data (frequently accessed)
cache.Set(ctx, "config", data, 24*time.Hour)

// Warm data (moderately accessed)
cache.Set(ctx, "user:123", profile, 1*time.Hour)

// Cold data (rarely accessed)
cache.Set(ctx, "archive:2023", data, 7*24*time.Hour)
```

### Context Timeouts

Always use context with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
defer cancel()

val, err := cache.Get(ctx, "key")
```

### Resource Cleanup

Always defer `cache.Close()`:

```go
cache, err := gokart.OpenCache(ctx, addr)
if err != nil {
    return err
}
defer cache.Close()  // Ensures clean shutdown
```

### Remember Pattern Usage

Prefer `RememberJSON` for structured data:

```go
// Good - type-safe
var user User
err := cache.RememberJSON(ctx, "user:123", time.Hour, &user, fetchUser)

// Avoids manual unmarshaling
var user User
data, _ := cache.Remember(ctx, "user:123", time.Hour, fetchUser)
json.Unmarshal([]byte(data), &user)  // Unnecessary work
```

---

## Reference

### Functions

| Function | Description |
|----------|-------------|
| [`OpenCache`](#opening-a-cache) | Opens cache with default settings |
| [`OpenCacheURL`](#opening-a-cache) | Opens cache from connection URL |
| [`OpenCacheWithConfig`](#opening-a-cache) | Opens cache with custom config |
| [`DefaultCacheConfig`](#cacheconfig-struct) | Returns default configuration |

### Cache Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `Get` | `(string, error)` | Retrieve string value |
| `Set` | `error` | Store string value with TTL |
| `GetJSON` | `error` | Retrieve and unmarshal JSON |
| `SetJSON` | `error` | Marshal and store JSON with TTL |
| `Delete` | `error` | Delete one or more keys |
| `Exists` | `(bool, error)` | Check if key exists |
| `Expire` | `error` | Set TTL on existing key |
| `TTL` | `(time.Duration, error)` | Get remaining TTL |
| `Incr` | `(int64, error)` | Increment counter by 1 |
| `IncrBy` | `(int64, error)` | Increment counter by amount |
| `SetNX` | `(bool, error)` | Set if not exists (lock) |
| `Remember` | `(string, error)` | Get or compute string value |
| `RememberJSON` | `error` | Get or compute JSON value |
| `Client` | `*redis.Client` | Access underlying client |
| `Close` | `error` | Close connection |

### Utility Functions

| Function | Description |
|----------|-------------|
| [`IsNil`](#cache-miss-detection) | Returns true if error is cache miss (`redis.Nil`) |

### See Also

- [`go-redis` documentation](https://redis.uptrace.dev/)
- [Redis data types](https://redis.io/docs/data-types/)
- [Redis best practices](https://redis.io/docs/manual/patterns/)
