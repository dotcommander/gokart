package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if got, want := cfg.Addr, "localhost:6379"; got != want {
		t.Errorf("DefaultConfig().Addr = %q, want %q", got, want)
	}
	if got, want := cfg.DB, 0; got != want {
		t.Errorf("DefaultConfig().DB = %d, want %d", got, want)
	}
	if got, want := cfg.PoolSize, 10; got != want {
		t.Errorf("DefaultConfig().PoolSize = %d, want %d", got, want)
	}
	if got, want := cfg.MinIdleConns, 2; got != want {
		t.Errorf("DefaultConfig().MinIdleConns = %d, want %d", got, want)
	}
	if got, want := cfg.DialTimeout, 5*time.Second; got != want {
		t.Errorf("DefaultConfig().DialTimeout = %v, want %v", got, want)
	}
	if got, want := cfg.ReadTimeout, 3*time.Second; got != want {
		t.Errorf("DefaultConfig().ReadTimeout = %v, want %v", got, want)
	}
	if got, want := cfg.WriteTimeout, 3*time.Second; got != want {
		t.Errorf("DefaultConfig().WriteTimeout = %v, want %v", got, want)
	}
	if cfg.URL != "" {
		t.Errorf("DefaultConfig().URL = %q, want empty", cfg.URL)
	}
	if cfg.Password != "" {
		t.Errorf("DefaultConfig().Password = %q, want empty", cfg.Password)
	}
	if cfg.KeyPrefix != "" {
		t.Errorf("DefaultConfig().KeyPrefix = %q, want empty", cfg.KeyPrefix)
	}
}

func TestCacheKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "no prefix returns key unchanged",
			prefix: "",
			key:    "user:1",
			want:   "user:1",
		},
		{
			name:   "prefix is prepended",
			prefix: "myapp:",
			key:    "user:1",
			want:   "myapp:user:1",
		},
		{
			name:   "empty key with prefix",
			prefix: "myapp:",
			key:    "",
			want:   "myapp:",
		},
		{
			name:   "empty key without prefix",
			prefix: "",
			key:    "",
			want:   "",
		},
		{
			name:   "prefix without trailing colon",
			prefix: "myapp",
			key:    "user:1",
			want:   "myappuser:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &Cache{prefix: tt.prefix}
			if got := c.Key(tt.key); got != tt.want {
				t.Errorf("Cache{prefix:%q}.Key(%q) = %q, want %q",
					tt.prefix, tt.key, got, tt.want)
			}
		})
	}
}

func TestOpenURLWithPrefix_ParseError(t *testing.T) {
	t.Parallel()

	// Invalid URL must return an error before any network call.
	_, err := OpenURLWithPrefix(t.Context(), "not-a-valid-redis-url", "myapp:")
	if err == nil {
		t.Fatal("OpenURLWithPrefix with invalid URL: want error, got nil")
	}
}

type countingValue struct {
	Name       string `json:"name"`
	marshals   *atomic.Int64
	unmarshals *atomic.Int64
}

func (v countingValue) MarshalJSON() ([]byte, error) {
	v.marshals.Add(1)
	return []byte(`{"name":` + strconv.Quote(v.Name) + `}`), nil
}

func (v *countingValue) UnmarshalJSON(data []byte) error {
	v.unmarshals.Add(1)
	var plain struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &plain); err != nil {
		return err
	}
	v.Name = plain.Name
	return nil
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "redis.Nil returns true",
			err:  redis.Nil,
			want: true,
		},
		{
			name: "nil error returns false",
			err:  nil,
			want: false,
		},
		{
			name: "unrelated error returns false",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsNil(tt.err); got != tt.want {
				t.Errorf("IsNil(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func openTestCache(t *testing.T) *Cache {
	t.Helper()
	c, err := Open(context.Background(), "localhost:6379")
	if err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestRememberJSON_Miss(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:miss:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(key)).Err() })

	var marshals, unmarshals atomic.Int64
	calls := 0
	var dest countingValue
	dest.marshals, dest.unmarshals = &marshals, &unmarshals
	err := c.RememberJSON(ctx, key, time.Minute, &dest, func() (interface{}, error) {
		calls++
		return countingValue{Name: "Ada", marshals: &marshals, unmarshals: &unmarshals}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("fn calls = %d, want 1", calls)
	}
	if dest.Name != "Ada" {
		t.Errorf("dest.Name = %q, want Ada", dest.Name)
	}
	if marshals.Load() != 1 {
		t.Errorf("marshals = %d, want exactly 1", marshals.Load())
	}
	if unmarshals.Load() != 1 {
		t.Errorf("unmarshals = %d, want exactly 1", unmarshals.Load())
	}
	var stored struct {
		Name string `json:"name"`
	}
	if err := c.GetJSON(ctx, key, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Name != "Ada" {
		t.Errorf("stored.Name = %q, want Ada", stored.Name)
	}
}

func TestRememberJSON_Hit(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:hit:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(key)).Err() })

	if err := c.SetJSON(ctx, key, map[string]string{"name": "Bob"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	var dest struct {
		Name string `json:"name"`
	}
	calls := 0
	err := c.RememberJSON(ctx, key, time.Minute, &dest, func() (interface{}, error) {
		calls++
		return map[string]string{"name": "Computed"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Errorf("fn calls = %d, want 0 on cache hit", calls)
	}
	if dest.Name != "Bob" {
		t.Errorf("dest.Name = %q, want Bob", dest.Name)
	}
}

func TestRememberJSON_FnError(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:err:%d", time.Now().UnixNano())
	sentinel := errors.New("compute failed")
	var dest struct {
		Name string `json:"name"`
	}
	err := c.RememberJSON(ctx, key, time.Minute, &dest, func() (interface{}, error) {
		return nil, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel", err)
	}
}

func TestRememberJSON_Concurrent(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:concurrent:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(key)).Err() })

	var calls atomic.Int64
	const callers = 8
	var wg sync.WaitGroup
	var marshals, unmarshals atomic.Int64
	dests := make([]countingValue, callers)
	errs := make([]error, callers)
	for i := range dests {
		dests[i].marshals, dests[i].unmarshals = &marshals, &unmarshals
	}
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = c.RememberJSON(ctx, key, time.Minute, &dests[i], func() (interface{}, error) {
				calls.Add(1)
				time.Sleep(50 * time.Millisecond)
				return countingValue{Name: "Ada", marshals: &marshals, unmarshals: &unmarshals}, nil
			})
		}(i)
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("fn calls = %d, want exactly 1", calls.Load())
	}
	for i, err := range errs {
		if err != nil {
			t.Errorf("caller %d err = %v", i, err)
		}
	}
	for i, d := range dests {
		if d.Name != "Ada" {
			t.Errorf("caller %d dest.Name = %q, want Ada", i, d.Name)
		}
	}
	if marshals.Load() != 1 {
		t.Errorf("marshals = %d, want exactly 1", marshals.Load())
	}
	if unmarshals.Load() != callers {
		t.Errorf("unmarshals = %d, want %d (one per caller)", unmarshals.Load(), callers)
	}
}
