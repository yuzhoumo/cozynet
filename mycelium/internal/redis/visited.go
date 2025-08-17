package redis

import (
	"context"
	"mycelium/internal/crawler"
)

func (rc *RedisCache) Visit(ctx context.Context, item crawler.QueueItem) error {
	key := item.GetLocation()
	if err := rc.rdb.Set(ctx, key, nil, 0).Err(); err != nil {
		return err
	}
	return rc.rdb.SAdd(ctx, "visited", key).Err()
}

func (rc *RedisCache) IsVisited(ctx context.Context, item crawler.QueueItem) (bool, error) {
	exists, err := rc.rdb.Exists(ctx, item.GetLocation()).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}
