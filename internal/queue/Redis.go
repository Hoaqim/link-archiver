package queue

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	key    string
}

func NewRedisQueue(addr, key string) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisQueue{client: client, key: key}, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, payload []byte) error {
	return q.client.LPush(ctx, q.key, payload).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) ([]byte, error) {
	res, err := q.client.BRPop(ctx, 0, q.key).Result()
	if err != nil {
		return nil, err
	}

	return []byte(res[1]), nil
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}
