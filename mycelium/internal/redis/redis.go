package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	rdb *redis.Client
}

type RedisCacheOptions struct {
	Addr string
	Pass string
	DB   int
}

func NewRedisCache(ctx context.Context, options *RedisCacheOptions) (*RedisCache, error) {
	var rc RedisCache

	rc.rdb = redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Pass,
		DB:       options.DB,
	})

	if _, err := rc.rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &rc, nil
}
