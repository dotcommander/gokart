package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// Hash operations

func (c *Cache) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, c.key(key), field).Result()
}

func (c *Cache) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, c.key(key), values...).Err()
}

func (c *Cache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, c.key(key)).Result()
}

func (c *Cache) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, c.key(key), fields...).Err()
}

func (c *Cache) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return c.client.HIncrBy(ctx, c.key(key), field, incr).Result()
}

// Sorted set operations

func (c *Cache) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return c.client.ZAdd(ctx, c.key(key), members...).Err()
}

func (c *Cache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.ZRange(ctx, c.key(key), start, stop).Result()
}

func (c *Cache) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.client.ZRangeByScore(ctx, c.key(key), opt).Result()
}

func (c *Cache) ZScore(ctx context.Context, key, member string) (float64, error) {
	return c.client.ZScore(ctx, c.key(key), member).Result()
}

func (c *Cache) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.ZRem(ctx, c.key(key), members...).Err()
}

func (c *Cache) ZCard(ctx context.Context, key string) (int64, error) {
	return c.client.ZCard(ctx, c.key(key)).Result()
}

// Set operations

func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, c.key(key), members...).Err()
}

func (c *Cache) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SRem(ctx, c.key(key), members...).Err()
}

func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, c.key(key)).Result()
}

func (c *Cache) SIsMember(ctx context.Context, key, member string) (bool, error) {
	return c.client.SIsMember(ctx, c.key(key), member).Result()
}

// List operations

func (c *Cache) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, c.key(key), values...).Err()
}

func (c *Cache) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, c.key(key), values...).Err()
}

func (c *Cache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, c.key(key), start, stop).Result()
}

func (c *Cache) LPop(ctx context.Context, key string) (string, error) {
	return c.client.LPop(ctx, c.key(key)).Result()
}

func (c *Cache) RPop(ctx context.Context, key string) (string, error) {
	return c.client.RPop(ctx, c.key(key)).Result()
}

// Counter operations

func (c *Cache) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, c.key(key)).Result()
}

func (c *Cache) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.DecrBy(ctx, c.key(key), value).Result()
}
