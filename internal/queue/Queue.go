package queue

import "context"

type Queue interface {
	Enqueue(ctx context.Context, payload []byte) error
	Dequeue(ctx context.Context) (Message, error)
	Ping(ctx context.Context) error
	Close() error
}

type Message interface {
	Payload() []byte
	ReceiveCount() int
	Ack(ctx context.Context) error
	Nack(ctx context.Context) error
}

type Job struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Req struct {
	URL string `json:"url"`
}
