package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hoaqim/link-archiver/internal/archiver"
	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/Hoaqim/link-archiver/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	q, err := queue.NewRedisQueue(os.Getenv("REDIS_ADDR"), "queue:jobs")
	var store storage.Storage
	switch os.Getenv("STORAGE_BACKEND") {
	case "s3":
		// store, err = storage.NewS3(...)
		logger.Error("s3 not implemented yet")
		os.Exit(1)
	default:
		s, err := storage.NewLocal(os.Getenv("STORAGE_DIR"))
		if err != nil {
			logger.Error("storage init", "err", err)
			os.Exit(1)
		}
		store = s
	}

	if err != nil {
		logger.Error(err.Error())
	}
	defer q.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("Worker started")

	for {
		payload, err := q.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logger.Info("Shutting down")
				return
			}
			logger.Error("dequeue", "err", err)
			time.Sleep(time.Second)
			continue
		}

		arch := archiver.NewArchiver(30*time.Second, 10*1024*1024)
		if err := process(ctx, logger, arch, store, payload); err != nil {
			logger.Error("process job", "err", err)
			//TODO: dead letter queue, retry
		}
	}
}

func process(ctx context.Context, logger *slog.Logger, arch *archiver.Archiver, store storage.Storage, payload []byte) error {
	var job queue.Job
	if err := json.Unmarshal(payload, &job); err != nil {
		return err
	}
	logger.Info("Processing job", "id", job.ID, "url", job.URL)

	result, err := arch.Fetch(ctx, job.URL)
	if err != nil {
		logger.Error("Fetch error", "err", err)
		return err
	}

	key := job.ID + ".html"
	if err := store.Put(ctx, key, result.Body, result.ContentType); err != nil {
		return fmt.Errorf("store: %w", err)
	}

	logger.Info("archived", "id", job.ID, "url", job.URL, "key", key, "bytes", len(result.Body))
	return nil
}
