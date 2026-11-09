package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRemember_Miss(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:remember-miss:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(key)).Err() })

	calls := 0
	val, err := c.Remember(ctx, key, time.Minute, func() (any, error) {
		calls++
		return "computed", nil
	})
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	if val != "computed" || calls != 1 {
		t.Errorf("miss: val=%q calls=%d, want computed/1", val, calls)
	}
}

func TestRemember_HitSkipsFn(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:remember-hit:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(key)).Err() })

	if _, err := c.Remember(ctx, key, time.Minute, func() (any, error) { return "first", nil }); err != nil {
		t.Fatalf("first Remember: %v", err)
	}
	val, err := c.Remember(ctx, key, time.Minute, func() (any, error) {
		t.Error("fn must not run on cache hit")
		return "second", nil
	})
	if err != nil {
		t.Fatalf("second Remember: %v", err)
	}
	if val != "first" {
		t.Errorf("hit val = %q, want %q", val, "first")
	}
}

func TestRemember_FnErrorPropagates(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()
	key := fmt.Sprintf("test:remember-err:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(key)).Err() })

	boom := errors.New("boom")
	_, err := c.Remember(ctx, key, time.Minute, func() (any, error) { return nil, boom })
	if !errors.Is(err, boom) {
		t.Fatalf("Remember err = %v, want boom", err)
	}
	if err := c.Client().Get(ctx, c.Key(key)).Err(); err == nil {
		t.Error("failed fn must not populate cache")
	}
}

func TestRemember_CoercesBytesAndJSON(t *testing.T) {
	t.Parallel()
	c := openTestCache(t)
	ctx := context.Background()

	bkey := fmt.Sprintf("test:remember-bytes:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(bkey)).Err() })
	val, err := c.Remember(ctx, bkey, time.Minute, func() (any, error) { return []byte("raw"), nil })
	if err != nil || val != "raw" {
		t.Errorf("bytes: val=%q err=%v", val, err)
	}

	type payload struct {
		N int `json:"n"`
	}
	jkey := fmt.Sprintf("test:remember-json:%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = c.Client().Del(ctx, c.Key(jkey)).Err() })
	jval, err := c.Remember(ctx, jkey, time.Minute, func() (any, error) { return payload{N: 7}, nil })
	if err != nil || jval != `{"n":7}` {
		t.Errorf("json: val=%q err=%v", jval, err)
	}
}
