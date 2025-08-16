package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"mycelium/internal/crawler"
)

type RedisVisited struct {
	rc *redis.Client
}

type RedisVisitedOptions struct {
	Addr string
	Pass string
	DB   int
}

func NewRedisVisited(options *RedisVisitedOptions) *RedisVisited {
	var rv RedisVisited

	rv.rc = redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Pass,
		DB:       options.DB,
	})

	return &rv
}

func (rv *RedisVisited) Visit(ctx context.Context, item crawler.QueueItem) error {
	key := item.GetLocation()
	if err := rv.rc.Set(ctx, key, nil, 0).Err(); err != nil {
		return err
	}
	return rv.rc.SAdd(ctx, "visited", key).Err()
}

func (rv *RedisVisited) IsVisited(ctx context.Context, item crawler.QueueItem) (bool, error) {
	exists, err := rv.rc.Exists(ctx, item.GetLocation()).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (rv *RedisVisited) Reset(ctx context.Context) error {
	_, err := rv.rc.FlushDB(ctx).Result()
	return err
}
