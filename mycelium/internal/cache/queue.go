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
	res, err := rc.rdb.LPop(ctx, "queue").Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to pop redis queue item: %w", err)
	}

	var item RedisQueueItem

	err = proto.Unmarshal(res, &item)
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
