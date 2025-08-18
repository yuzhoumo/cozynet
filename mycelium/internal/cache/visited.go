package cache

import (
	"context"
	"mycelium/internal/crawler"
)

func (rc *CrawlerCache) Visit(ctx context.Context, item crawler.QueueItem) error {
	key := item.GetLocation()
	return rc.rdb.SAdd(ctx, "visited", key).Err()
}

func (rc *CrawlerCache) IsVisited(ctx context.Context, item crawler.QueueItem) (bool, error) {
	exists, err := rc.rdb.SIsMember(ctx, "visited", item.GetLocation()).Result()
	if err != nil {
		return false, err
	}
	return exists, nil
}
