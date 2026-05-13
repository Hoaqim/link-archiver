package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hoaqim/link-archiver/internal/queue"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	q, err := queue.NewRedisQueue(os.Getenv("REDIS_ADDR"), "queue:jobs")

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

		if err := process(ctx, logger, payload); err != nil {
			logger.Error("process job", "err", err)
			//TODO: dead letter queue, retry
		}
	}
}

func process(ctx context.Context, logger *slog.Logger, payload []byte) error {
	var job queue.Job
	if err := json.Unmarshal(payload, &job); err != nil {
		return err
	}
	logger.Info("Processing job", "id", job.ID, "url", job.URL)
	//TODO: fetch, store url, etc
	return nil
}
