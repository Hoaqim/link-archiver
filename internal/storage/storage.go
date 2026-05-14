package storage

import "context"

type Storage interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Get(ctx context.Context, key string) (data []byte, contentType string, err error)
	Exists(ctx context.Context, key string) (bool, error)
}
