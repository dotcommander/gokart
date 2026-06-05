package cache

import (
	"errors"
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
			if got := c.key(tt.key); got != tt.want {
				t.Errorf("Cache{prefix:%q}.key(%q) = %q, want %q",
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
