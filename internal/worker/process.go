package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Hoaqim/link-archiver/internal/archiver"
	"github.com/Hoaqim/link-archiver/internal/queue"
	"github.com/Hoaqim/link-archiver/internal/storage"
)

type Processor struct {
	Logger   *slog.Logger
	Archiver *archiver.Archiver
	Storage  storage.Storage
}

func (p *Processor) Process(ctx context.Context, msg queue.Message) error {
	var job queue.Job
	if err := json.Unmarshal(msg.Payload(), &job); err != nil {
		p.Logger.Error("unmarshal job; nack", "err", err,
			"receive_count", msg.ReceiveCount)
		return msg.Nack(ctx)
	}

	p.Logger.Info("processing job",
		"id", job.ID,
		"url", job.URL,
		"receive_count", msg.ReceiveCount())

	if err := p.archive(ctx, job); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		p.Logger.Warn("job failed; nack",
			"id", job.ID,
			"receive_count", msg.ReceiveCount(),
			"err", err)
		return msg.Nack(ctx)
	}

	p.Logger.Info("archived", "id", job.ID, "url", job.URL)
	return nil
}

func (p *Processor) archive(ctx context.Context, job queue.Job) error {
	result, err := p.Archiver.Fetch(ctx, job.URL)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	key := job.ID + ".html"
	if err := p.Storage.Put(ctx, key, result.Body, result.ContentType); err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return nil
}
