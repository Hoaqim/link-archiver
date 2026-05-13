package queue

import "context"

type Queue interface {
	Enqueue(ctx context.Context, payload []byte) error
	Dequeue(ctx context.Context) ([]byte, error)
	Close() error
}
