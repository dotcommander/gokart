package gokart

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheConfig configures Redis connection.
type CacheConfig struct {
	// URL is the Redis connection string.
	// Format: redis://:password@host:port/db or redis://host:port
	URL string

	// Addr is the Redis server address (alternative to URL).
	// Default: localhost:6379
	Addr string

	// Password for Redis authentication.
	Password string

	// DB is the Redis database number.
	// Default: 0
	DB int

	// PoolSize is the maximum number of connections.
	// Default: 10
	PoolSize int

	// MinIdleConns is the minimum number of idle connections.
	// Default: 2
	MinIdleConns int

	// DialTimeout is the timeout for establishing new connections.
	// Default: 5 seconds
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads.
	// Default: 3 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes.
	// Default: 3 seconds
	WriteTimeout time.Duration

	// KeyPrefix is prepended to all keys.
	KeyPrefix string
}

// DefaultCacheConfig returns production-ready defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// Cache wraps Redis client with convenience methods.
type Cache struct {
	client *redis.Client
	prefix string
}

// OpenCache opens a Redis connection with default settings.
//
// Example:
//
//	cache, err := gokart.OpenCache(ctx, "localhost:6379")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cache.Close()
func OpenCache(ctx context.Context, addr string) (*Cache, error) {
	cfg := DefaultCacheConfig()
	cfg.Addr = addr
	return OpenCacheWithConfig(ctx, cfg)
}

// OpenCacheURL opens a Redis connection using a URL.
//
// Example:
//
//	cache, err := gokart.OpenCacheURL(ctx, "redis://:password@localhost:6379/0")
func OpenCacheURL(ctx context.Context, url string) (*Cache, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// OpenCacheWithConfig opens a Redis connection with custom settings.
//
// Example:
//
//	cache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
//	    Addr:      "localhost:6379",
//	    Password:  "secret",
//	    KeyPrefix: "myapp:",
//	})
func OpenCacheWithConfig(ctx context.Context, cfg CacheConfig) (*Cache, error) {
	if cfg.URL != "" {
		cache, err := OpenCacheURL(ctx, cfg.URL)
		if err != nil {
			return nil, err
		}
		cache.prefix = cfg.KeyPrefix
		return cache, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Cache{client: client, prefix: cfg.KeyPrefix}, nil
}

// Client returns the underlying Redis client.
func (c *Cache) Client() *redis.Client {
	return c.client
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	return c.client.Close()
}

// key prefixes the key if a prefix is configured.
func (c *Cache) key(k string) string {
	if c.prefix != "" {
		return c.prefix + k
	}
	return k
}

// Get retrieves a string value.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.key(key)).Result()
}

// Set stores a string value with expiration.
func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, c.key(key), value, ttl).Err()
}

// GetJSON retrieves and unmarshals a JSON value.
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// SetJSON marshals and stores a value as JSON.
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

// Delete removes a key.
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	prefixedKeys := make([]string, len(keys))
	for i, k := range keys {
		prefixedKeys[i] = c.key(k)
	}
	return c.client.Del(ctx, prefixedKeys...).Err()
}

// Exists checks if a key exists.
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, c.key(key)).Result()
	return n > 0, err
}

// Expire sets a TTL on an existing key.
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, c.key(key), ttl).Err()
}

// TTL returns the remaining TTL of a key.
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.key(key)).Result()
}

// Incr increments a counter and returns the new value.
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, c.key(key)).Result()
}

// IncrBy increments a counter by a specific amount.
func (c *Cache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, c.key(key), value).Result()
}

// SetNX sets a value only if the key doesn't exist (for distributed locks).
func (c *Cache) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return c.client.SetNX(ctx, c.key(key), value, ttl).Result()
}

// Remember gets a value or sets it using the provided function.
//
// Example:
//
//	user, err := cache.Remember(ctx, "user:123", time.Hour, func() (interface{}, error) {
//	    return db.GetUser(ctx, 123)
//	})
func (c *Cache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (string, error) {
	val, err := c.Get(ctx, key)
	if err == nil {
		return val, nil
	}
	if err != redis.Nil {
		return "", err
	}

	result, err := fn()
	if err != nil {
		return "", err
	}

	var strVal string
	switch v := result.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		strVal = string(data)
	}

	if err := c.Set(ctx, key, strVal, ttl); err != nil {
		return "", err
	}

	return strVal, nil
}

// RememberJSON gets a value or computes and caches it as JSON.
// Unlike Remember, this preserves type information for GetJSON retrieval.
//
// Example:
//
//	var user User
//	err := cache.RememberJSON(ctx, "user:123", time.Hour, &user, func() (interface{}, error) {
//	    return db.GetUser(ctx, 123)
//	})
func (c *Cache) RememberJSON(ctx context.Context, key string, ttl time.Duration, dest interface{}, fn func() (interface{}, error)) error {
	// Try to get existing value
	err := c.GetJSON(ctx, key, dest)
	if err == nil {
		return nil
	}
	if err != redis.Nil {
		return err
	}

	// Compute new value
	result, err := fn()
	if err != nil {
		return err
	}

	// Store as JSON
	if err := c.SetJSON(ctx, key, result, ttl); err != nil {
		return err
	}

	// Unmarshal into dest (to populate the destination)
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result for dest: %w", err)
	}
	return json.Unmarshal(data, dest)
}

// IsNil returns true if the error is a cache miss.
func IsNil(err error) bool {
	return err == redis.Nil
}
