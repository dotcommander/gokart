package kv

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// KV wraps a Redis client with key-value convenience methods
// for data structures beyond basic caching.
// Create from an existing *redis.Client (e.g., cache.Client()).
type KV struct {
	client *redis.Client
}

// New creates a KV from an existing Redis client.
func New(client *redis.Client) *KV {
	return &KV{client: client}
}

// Client returns the underlying Redis client.
func (kv *KV) Client() *redis.Client {
	return kv.client
}

func (kv *KV) HGet(ctx context.Context, key, field string) (string, error) {
	return kv.client.HGet(ctx, key, field).Result()
}

func (kv *KV) HSet(ctx context.Context, key string, values ...interface{}) error {
	return kv.client.HSet(ctx, key, values...).Err()
}

func (kv *KV) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return kv.client.HGetAll(ctx, key).Result()
}

func (kv *KV) HDel(ctx context.Context, key string, fields ...string) error {
	return kv.client.HDel(ctx, key, fields...).Err()
}

func (kv *KV) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return kv.client.HIncrBy(ctx, key, field, incr).Result()
}

func (kv *KV) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return kv.client.ZAdd(ctx, key, members...).Err()
}

func (kv *KV) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return kv.client.ZRange(ctx, key, start, stop).Result()
}

func (kv *KV) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return kv.client.ZRangeByScore(ctx, key, opt).Result()
}

func (kv *KV) ZScore(ctx context.Context, key, member string) (float64, error) {
	return kv.client.ZScore(ctx, key, member).Result()
}

func (kv *KV) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return kv.client.ZRem(ctx, key, members...).Err()
}

func (kv *KV) ZCard(ctx context.Context, key string) (int64, error) {
	return kv.client.ZCard(ctx, key).Result()
}

func (kv *KV) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return kv.client.SAdd(ctx, key, members...).Err()
}

func (kv *KV) SRem(ctx context.Context, key string, members ...interface{}) error {
	return kv.client.SRem(ctx, key, members...).Err()
}

func (kv *KV) SMembers(ctx context.Context, key string) ([]string, error) {
	return kv.client.SMembers(ctx, key).Result()
}

func (kv *KV) SIsMember(ctx context.Context, key, member string) (bool, error) {
	return kv.client.SIsMember(ctx, key, member).Result()
}

func (kv *KV) LPush(ctx context.Context, key string, values ...interface{}) error {
	return kv.client.LPush(ctx, key, values...).Err()
}

func (kv *KV) RPush(ctx context.Context, key string, values ...interface{}) error {
	return kv.client.RPush(ctx, key, values...).Err()
}

func (kv *KV) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return kv.client.LRange(ctx, key, start, stop).Result()
}

func (kv *KV) LPop(ctx context.Context, key string) (string, error) {
	return kv.client.LPop(ctx, key).Result()
}

func (kv *KV) RPop(ctx context.Context, key string) (string, error) {
	return kv.client.RPop(ctx, key).Result()
}

func (kv *KV) Decr(ctx context.Context, key string) (int64, error) {
	return kv.client.Decr(ctx, key).Result()
}

func (kv *KV) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return kv.client.DecrBy(ctx, key, value).Result()
}
