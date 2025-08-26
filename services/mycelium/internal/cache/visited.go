package cache

import (
	"context"
)

func (rc *CrawlerCache) Visit(ctx context.Context, location string) error {
	return rc.rdb.SAdd(ctx, "visited", location).Err()
}

func (rc *CrawlerCache) IsVisited(ctx context.Context, location string) (bool, error) {
	exists, err := rc.rdb.SIsMember(ctx, "visited", location).Result()
	if err != nil {
		return false, err
	}
	return exists, nil
}
