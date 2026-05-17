package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hoaqim/link-archiver/internal/config"
	"github.com/Hoaqim/link-archiver/internal/httpapi"
	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/Hoaqim/link-archiver/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error(err.Error())
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	queue, err := queue.NewSQS(ctx, cfg.SQSQueueURL)
	if err != nil {
		logger.Error(err.Error())
	}

	store, err := storage.NewS3(ctx, cfg.S3Bucket)
	if err != nil {
		logger.Error(err.Error())
	}

	srv := &httpapi.Server{
		Logger:  logger,
		Queue:   queue,
		Storage: store,
	}

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("listening")
		if err := httpServer.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}

}
