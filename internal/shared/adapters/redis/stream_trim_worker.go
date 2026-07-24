package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
	"github.com/redis/go-redis/v9"
)

const (
	streamTrimInterval  = 24 * time.Hour
	streamTrimRetention = 48 * time.Hour
)

// StreamTrimWorker recorta periódicamente el stream de eventos para que no
// crezca indefinidamente (XADD no aplica ningún límite por sí solo).
type StreamTrimWorker struct {
	client     *redis.Client
	streamName string
}

func NewStreamTrimWorker(client *redis.Client, cfg config.Config) *StreamTrimWorker {
	return &StreamTrimWorker{client: client, streamName: cfg.RedisStream.StreamName}
}

func (w *StreamTrimWorker) Start(ctx context.Context) {
	go w.loop(ctx)
}

func (w *StreamTrimWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(streamTrimInterval)
	defer ticker.Stop()

	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *StreamTrimWorker) tick(ctx context.Context) {
	minID := fmt.Sprintf("%d-0", time.Now().Add(-streamTrimRetention).UnixMilli())
	if err := w.client.XTrimMinIDApprox(ctx, w.streamName, minID, 0).Err(); err != nil {
		slog.Error("StreamTrimWorker trim failed", "error", err)
	}
}
