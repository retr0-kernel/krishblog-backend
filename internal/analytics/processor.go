package analytics

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"krishblog/internal/database"
)

const (
	flushInterval  = 5 * time.Second // how often we drain the queue
	batchSize      = 100             // events per flush batch
	processorRetry = 3 * time.Second // backoff on error
)

// Processor is a background worker that drains the Redis analytics queue
// and persists events to Postgres in batches.
//
// Run it in a goroutine:
//
//	go processor.Run(ctx)
type Processor struct {
	repo  *Repository
	redis *database.Redis
	log   *slog.Logger
}

// NewProcessor creates a processor. Share the same repo as the service.
func NewProcessor(repo *Repository, redis *database.Redis, log *slog.Logger) *Processor {
	return &Processor{repo: repo, redis: redis, log: log}
}

// Run starts the flush loop. Blocks until ctx is cancelled.
func (p *Processor) Run(ctx context.Context) {
	p.log.Info("analytics processor started",
		slog.Duration("interval", flushInterval),
		slog.Int("batch_size", batchSize),
	)

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush before exit
			p.log.Info("analytics processor shutting down, flushing remaining events")
			p.flush(context.Background())
			p.log.Info("analytics processor stopped")
			return

		case <-ticker.C:
			p.flush(ctx)
		}
	}
}

// flush drains up to batchSize events from Redis and inserts them into Postgres.
func (p *Processor) flush(ctx context.Context) {
	for {
		// LRANGE + LTRIM is effectively a non-blocking pop of N items
		items, err := p.redis.LRange(ctx, redisQueueKey, 0, int64(batchSize-1))
		if err != nil || len(items) == 0 {
			return
		}

		// Remove the items we just read
		if err := p.redis.LTrim(ctx, redisQueueKey, int64(len(items)), -1); err != nil {
			p.log.Error("analytics ltrim failed", slog.String("error", err.Error()))
			return
		}

		inserted := 0
		failed := 0

		for _, item := range items {
			var event enrichedEvent
			if err := json.Unmarshal([]byte(item), &event); err != nil {
				p.log.Warn("analytics: malformed queue item, skipping",
					slog.String("error", err.Error()),
				)
				failed++
				continue
			}

			if err := p.repo.InsertEvent(ctx, event); err != nil {
				p.log.Error("analytics: insert failed",
					slog.String("error", err.Error()),
					slog.String("type", event.Type),
				)
				failed++
				// Re-queue failed event so it isn't lost (best-effort)
				_ = p.redis.RPush(ctx, redisQueueKey+":dlq", item)
				continue
			}
			inserted++
		}

		if inserted > 0 || failed > 0 {
			p.log.Info("analytics flush",
				slog.Int("inserted", inserted),
				slog.Int("failed", failed),
				slog.Int("batch", len(items)),
			)
		}

		// If we read a full batch there may be more — loop immediately
		if len(items) < batchSize {
			return
		}
	}
}
