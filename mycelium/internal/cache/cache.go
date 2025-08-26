package cache

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

type CrawlerCache struct {
	rdb *redis.Client
}

type CrawlerCacheOptions struct {
	Addr string
	Pass string
	DB   int
}

func NewRedisCache(ctx context.Context, options *CrawlerCacheOptions) (*CrawlerCache, error) {
	var rc CrawlerCache

	rc.rdb = redis.NewClient(&redis.Options{
		Addr:         options.Addr,
		Password:     options.Pass,
		DB:           options.DB,
		PoolSize:     50, // Increase pool size for multiple crawlers
		MinIdleConns: 10, // Keep minimum connections open
		MaxRetries:   3,  // Retry failed commands
	})

	if _, err := rc.rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &rc, nil
}
