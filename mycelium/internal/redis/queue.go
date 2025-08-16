package redis

import (
	"context"
	"fmt"

	"mycelium/internal/crawler"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type RedisQueue struct {
	rc *redis.Client
}

type RedisQueueOptions struct {
	Addr string
	Pass string
	DB   int
}

func NewRedisQueue(options *RedisQueueOptions) *RedisQueue {
	var rq RedisQueue

	rq.rc = redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Pass,
		DB:       options.DB,
	})

	return &rq
}

func (rq *RedisQueue) Enqueue(ctx context.Context, item crawler.QueueItem) error {
	data, err := proto.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to serialize redis queue item: %w", err)
	}

	if err := rq.rc.RPush(ctx, "queue", data).Err(); err != nil {
		return fmt.Errorf("failed to enqueue item: %w", err)
	}

	return nil
}

func (rq *RedisQueue) Pop(ctx context.Context) (crawler.QueueItem, error) {
	res, err := rq.rc.LPop(ctx, "queue").Bytes()
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

func (rq *RedisQueue) Size(ctx context.Context) (int32, error) {
	res, err := rq.rc.LLen(ctx, "queue").Result()
	if err != nil {
		return -1, fmt.Errorf("failed to get redis queue size: %w", err)
	}
	return int32(res), nil
}
