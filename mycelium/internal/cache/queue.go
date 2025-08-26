package cache

import (
	"context"
	"fmt"

	"mycelium/internal/crawler"

	"google.golang.org/protobuf/proto"
)

func (rc *CrawlerCache) QueuePush(ctx context.Context, item crawler.QueueItem) error {
	data, err := proto.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to serialize redis queue item: %w", err)
	}

	if err := rc.rdb.RPush(ctx, "queue", data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue item: %w", err)
	}

	return nil
}

func (rc *CrawlerCache) QueuePop(ctx context.Context) (crawler.QueueItem, error) {
	res, err := rc.rdb.BLPop(ctx, 0, "queue").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to pop redis queue item: %w", err)
	}

	// BLPop returns [queueKey, value], we want just the value
	if len(res) < 2 {
		return nil, fmt.Errorf("unexpected BLPop result format")
	}

	var item RedisQueueItem
	err = proto.Unmarshal([]byte(res[1]), &item)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal redis queue item: %w", err)
	}

	return &item, nil
}

func (rc *CrawlerCache) QueueSize(ctx context.Context) (int32, error) {
	res, err := rc.rdb.LLen(ctx, "queue").Result()
	if err != nil {
		return -1, fmt.Errorf("failed to get redis queue size: %w", err)
	}
	return int32(res), nil
}

func (rc *CrawlerCache) PushToFungicide(ctx context.Context, pageJSON string, queueKey string) error {
	if err := rc.rdb.RPush(ctx, queueKey, pageJSON).Err(); err != nil {
		return fmt.Errorf("failed to push to fungicide queue: %w", err)
	}
	return nil
}

func (rc *CrawlerCache) PopFromMyceliumIngress(ctx context.Context, queueKey string) (string, error) {
	res, err := rc.rdb.BLPop(ctx, 0, queueKey).Result()
	if err != nil {
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
