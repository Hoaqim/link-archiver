package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hoaqim/link-archiver/internal/archiver"
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

		arch := archiver.NewArchiver(30*time.Second, 10*1024*1024)
		if err := process(ctx, logger, arch, payload); err != nil {
			logger.Error("process job", "err", err)
			//TODO: dead letter queue, retry
		}
	}
}

func process(ctx context.Context, logger *slog.Logger, arch *archiver.Archiver, payload []byte) error {
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

	logger.Info("Fetched:",
		"id", job.ID,
		"url", result.URL,
		"statusCode", result.StatusCode,
		"contentType", result.ContentType,
		"fetchedat", result.FetchedAt,
		"body", result.Body,
	)
	//TODO: storage instead of logg
	return nil
}
