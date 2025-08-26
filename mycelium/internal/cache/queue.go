package cache

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

func (rc *CrawlerCache) PushToFungicide(ctx context.Context, pageJSON string, queueKey string) error {
	if err := rc.rdb.RPush(ctx, queueKey, pageJSON).Err(); err != nil {
		return fmt.Errorf("failed to push to fungicide queue: %w", err)
	}
	return nil
}

func (rc *CrawlerCache) PushToMyceliumIngress(ctx context.Context, itemJSON string, queueKey string) error {
	if err := rc.rdb.RPush(ctx, queueKey, itemJSON).Err(); err != nil {
		return fmt.Errorf("failed to push to mycelium ingress queue: %w", err)
	}
	return nil
}

func (rc *CrawlerCache) PopFromMyceliumIngress(ctx context.Context, queueKey string) (string, error) {
	// Use a 5-second timeout instead of blocking indefinitely
	res, err := rc.rdb.BLPop(ctx, 5*time.Second, queueKey).Result()
	if err != nil {
		// If it's a timeout (no items available), return a specific error
		if err == redis.Nil {
			return "", fmt.Errorf("no items available in queue")
		}
		return "", fmt.Errorf("failed to pop from mycelium ingress: %w", err)
	}
	// BLPop returns [queueKey, value], we want just the value
	if len(res) < 2 {
		return "", fmt.Errorf("unexpected BLPop result format")
	}
	return res[1], nil
}

func (rc *CrawlerCache) IsBlacklisted(ctx context.Context, domain string, blacklistKey string) (bool, error) {
	res, err := rc.rdb.SIsMember(ctx, blacklistKey, domain).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}
	return res, nil
}

func (rc *CrawlerCache) IngressQueueSize(ctx context.Context, queueKey string) (int32, error) {
	res, err := rc.rdb.LLen(ctx, queueKey).Result()
	if err != nil {
		return -1, fmt.Errorf("failed to get ingress queue size: %w", err)
	}
	return int32(res), nil
}
