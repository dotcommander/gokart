// Example: Redis caching with GoKart.
//
// This example demonstrates:
//   - Opening a Redis connection with default and custom configs
//   - String and JSON get/set operations
//   - The Remember pattern (get-or-compute)
//   - Key prefixing for multi-tenant or namespaced caching
//   - Counters and distributed locking
//
// Prerequisites:
//   - Redis running on localhost:6379
//   - Or set REDIS_URL environment variable
//
// Run with: go run main.go
package main

import (
	"context"
	"log"
	"time"

	"github.com/dotcommander/gokart"
)

// User represents a cached user
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	ctx := context.Background()

	// Example 1: Simple connection with default settings
	cache, err := gokart.OpenCache(ctx, "localhost:6379")
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	log.Println("Connected to Redis")

	// Example 2: String operations
	err = cache.Set(ctx, "greeting", "Hello, World!", time.Hour)
	if err != nil {
		log.Printf("Set failed: %v", err)
	}

	greeting, err := cache.Get(ctx, "greeting")
	if err != nil {
		if gokart.IsNil(err) {
			log.Println("Key not found")
		} else {
			log.Printf("Get failed: %v", err)
		}
	} else {
		log.Printf("Greeting: %s", greeting)
	}

	// Example 3: JSON operations
	user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}

	err = cache.SetJSON(ctx, "user:1", user, time.Hour)
	if err != nil {
		log.Printf("SetJSON failed: %v", err)
	}

	var cachedUser User
	err = cache.GetJSON(ctx, "user:1", &cachedUser)
	if err != nil {
		log.Printf("GetJSON failed: %v", err)
	} else {
		log.Printf("Cached user: %+v", cachedUser)
	}

	// Example 4: Remember pattern (get-or-compute)
	// This is the most common caching pattern:
	// - Check if value exists in cache
	// - If yes, return it
	// - If no, compute it, cache it, return it
	val, err := cache.Remember(ctx, "expensive-computation", time.Hour, func() (interface{}, error) {
		log.Println("Computing value (this won't run on cache hit)...")
		// Simulate expensive computation
		time.Sleep(100 * time.Millisecond)
		return "computed-result", nil
	})
	if err != nil {
		log.Printf("Remember failed: %v", err)
	} else {
		log.Printf("Value: %s", val)
	}

	// Call again - will return cached value without re-computing
	val, _ = cache.Remember(ctx, "expensive-computation", time.Hour, func() (interface{}, error) {
		log.Println("This won't be printed - value is cached")
		return "new-value", nil
	})
	log.Printf("Cached value: %s", val)

	// Example 5: RememberJSON for typed caching
	var remUser User
	err = cache.RememberJSON(ctx, "user:2", time.Hour, &remUser, func() (interface{}, error) {
		log.Println("Fetching user from database...")
		return User{ID: 2, Name: "Bob", Email: "bob@example.com"}, nil
	})
	if err != nil {
		log.Printf("RememberJSON failed: %v", err)
	} else {
		log.Printf("User: %+v", remUser)
	}

	// Example 6: Counter operations
	count, err := cache.Incr(ctx, "page-views")
	if err != nil {
		log.Printf("Incr failed: %v", err)
	} else {
		log.Printf("Page views: %d", count)
	}

	count, _ = cache.IncrBy(ctx, "page-views", 10)
	log.Printf("Page views after +10: %d", count)

	// Example 7: Distributed locking with SetNX
	lockKey := "lock:resource"
	acquired, err := cache.SetNX(ctx, lockKey, "owner-1", 10*time.Second)
	if err != nil {
		log.Printf("Lock failed: %v", err)
	} else if acquired {
		log.Println("Lock acquired!")
		// Do protected work...
		cache.Delete(ctx, lockKey) // Release lock
	} else {
		log.Println("Lock already held by another process")
	}

	// Example 8: Key prefix for namespacing
	prefixedCache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
		Addr:      "localhost:6379",
		KeyPrefix: "myapp:",
	})
	if err != nil {
		log.Printf("Prefixed cache failed: %v", err)
	} else {
		defer prefixedCache.Close()

		// All keys will be prefixed with "myapp:"
		prefixedCache.Set(ctx, "setting", "value", time.Hour)
		// Actually stored as "myapp:setting"

		log.Println("Using prefixed cache")
	}

	// Example 9: Check existence and TTL
	exists, _ := cache.Exists(ctx, "greeting")
	log.Printf("Key 'greeting' exists: %v", exists)

	ttl, _ := cache.TTL(ctx, "greeting")
	log.Printf("TTL remaining: %v", ttl)

	// Example 10: Delete keys
	err = cache.Delete(ctx, "greeting", "user:1", "user:2")
	if err != nil {
		log.Printf("Delete failed: %v", err)
	}

	// Example 11: Access underlying Redis client for advanced operations
	client := cache.Client()
	result := client.Ping(ctx)
	log.Printf("Ping result: %s", result.Val())

	// Clean up demo keys
	cache.Delete(ctx, "expensive-computation", "page-views")
}

// Example: URL-based connection
//
// For production with authentication:
//
//	cache, err := gokart.OpenCacheURL(ctx, "redis://:password@redis.example.com:6379/0")

// Example: Custom configuration
//
//	cache, err := gokart.OpenCacheWithConfig(ctx, gokart.CacheConfig{
//	    Addr:         "redis.example.com:6379",
//	    Password:     "secret",
//	    DB:           0,
//	    PoolSize:     20,
//	    MinIdleConns: 5,
//	    DialTimeout:  10 * time.Second,
//	    ReadTimeout:  5 * time.Second,
//	    WriteTimeout: 5 * time.Second,
//	    KeyPrefix:    "prod:",
//	})

// Example: Error handling pattern
//
//	val, err := cache.Get(ctx, "key")
//	switch {
//	case err == nil:
//	    // Use val
//	case gokart.IsNil(err):
//	    // Cache miss - fetch from source
//	default:
//	    // Redis error - handle or fall back
//	}
