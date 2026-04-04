package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
	prefix string
}

func NewRedisService(client *redis.Client, prefix string) *RedisService {
	return &RedisService{
		client: client,
		prefix: prefix,
	}
}

func (r *RedisService) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, r.prefix+key, value, ttl).Err()
}

func (r *RedisService) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, r.prefix+key).Result()
}

func (r *RedisService) Exists(ctx context.Context, key string) (bool, error) {
	val, err := r.client.Exists(ctx, r.prefix+key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (r *RedisService) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.prefix+key).Err()
}
