package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hoaqim/link-archiver/internal/archiver"
	"github.com/Hoaqim/link-archiver/internal/config"
	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/Hoaqim/link-archiver/internal/storage"
	"github.com/Hoaqim/link-archiver/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	q, err := queue.NewSQS(ctx, cfg.SQSQueueURL)
	if err != nil {
		logger.Error("queue init", "err", err)
		os.Exit(1)
	}
	defer q.Close()

	store, err := storage.NewS3(ctx, cfg.S3Bucket)
	if err != nil {
		logger.Error("storage init", "err", err)
		os.Exit(1)
	}

	proc := &worker.Processor{
		Logger:   logger,
		Archiver: archiver.NewArchiver(30*time.Second, 10*1024*1024),
		Storage:  store,
	}

	logger.Info("Worker started",
		"queue_url", cfg.SQSQueueURL,
		"S3_bucket", cfg.S3Bucket)

	for {
		msg, err := q.Dequeue(ctx)
		if err != nil {
			if err != ctx.Err() {
				logger.Error("shutdown")
				return
			}

			if errors.Is(err, queue.ErrNoMessage) {
				continue
			}

			logger.Error("dequeue", "err", err)
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
			continue
		}
		if err := proc.Process(ctx, msg); err != nil {
			if ctx.Err() != nil {
				logger.Info("shutting down mid-job")
				return
			}
			logger.Error("process", "err", err)
		}
	}
}
