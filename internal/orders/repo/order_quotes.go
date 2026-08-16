package repo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	quoteKeyPrefix       = "order:quote:"
	quoteUsedKeyPrefix   = "order:quote:used:"
	quoteActiveKeyPrefix = "order:quote:active:"
)

type orderQuotesRepo struct {
	redis *redis.Client
}

func NewOrderQuotesRepo(redis *redis.Client) ports.OrderQuotesRepo {
	return &orderQuotesRepo{redis: redis}
}

func quoteKey(id uuid.UUID) string {
	return quoteKeyPrefix + id.String()
}

func quoteUsedKey(id uuid.UUID) string {
	return quoteUsedKeyPrefix + id.String()
}

func quoteActiveKey(customerID uuid.UUID) string {
	return quoteActiveKeyPrefix + customerID.String()
}

func (r *orderQuotesRepo) Save(ctx context.Context, quote *models.OrderQuote, ttl time.Duration) error {
	raw, err := json.Marshal(quote)
	if err != nil {
		slog.Error("orderQuotes Save marshal failed", "error", err)
		return errcustom.ErrUnexpectedError
	}
	if err := r.redis.Set(ctx, quoteKey(quote.QuoteID), raw, ttl).Err(); err != nil {
		slog.Error("orderQuotes Save redis failed", "quoteID", quote.QuoteID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *orderQuotesRepo) Get(ctx context.Context, quoteID uuid.UUID) (*models.OrderQuote, error) {
	raw, err := r.redis.Get(ctx, quoteKey(quoteID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errcustom.ErrQuoteNotFound
		}
		slog.Error("orderQuotes Get redis failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	var quote models.OrderQuote
	if err := json.Unmarshal([]byte(raw), &quote); err != nil {
		slog.Error("orderQuotes Get unmarshal failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &quote, nil
}

func (r *orderQuotesRepo) Delete(ctx context.Context, quoteID uuid.UUID) error {
	if err := r.redis.Del(ctx, quoteKey(quoteID)).Err(); err != nil {
		slog.Error("orderQuotes Delete failed", "quoteID", quoteID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *orderQuotesRepo) TryConsume(ctx context.Context, quoteID uuid.UUID, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = time.Minute
	}
	ok, err := r.redis.SetNX(ctx, quoteUsedKey(quoteID), "1", ttl).Result()
	if err != nil {
		slog.Error("orderQuotes TryConsume failed", "quoteID", quoteID, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	return ok, nil
}

func (r *orderQuotesRepo) ReleaseConsume(ctx context.Context, quoteID uuid.UUID) error {
	if err := r.redis.Del(ctx, quoteUsedKey(quoteID)).Err(); err != nil {
		slog.Error("orderQuotes ReleaseConsume failed", "quoteID", quoteID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *orderQuotesRepo) IsConsumed(ctx context.Context, quoteID uuid.UUID) (bool, error) {
	n, err := r.redis.Exists(ctx, quoteUsedKey(quoteID)).Result()
	if err != nil {
		slog.Error("orderQuotes IsConsumed failed", "quoteID", quoteID, "error", err)
		return false, errcustom.ErrUnexpectedError
	}
	return n > 0, nil
}

func (r *orderQuotesRepo) GetActiveID(ctx context.Context, customerID uuid.UUID) (uuid.UUID, error) {
	raw, err := r.redis.Get(ctx, quoteActiveKey(customerID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, nil
		}
		slog.Error("orderQuotes GetActiveID failed", "customerID", customerID, "error", err)
		return uuid.Nil, errcustom.ErrUnexpectedError
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		_ = r.ClearActiveID(ctx, customerID)
		return uuid.Nil, nil
	}
	return id, nil
}

func (r *orderQuotesRepo) SetActiveID(ctx context.Context, customerID, quoteID uuid.UUID, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Minute
	}
	if err := r.redis.Set(ctx, quoteActiveKey(customerID), quoteID.String(), ttl).Err(); err != nil {
		slog.Error("orderQuotes SetActiveID failed", "customerID", customerID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *orderQuotesRepo) ClearActiveID(ctx context.Context, customerID uuid.UUID) error {
	if err := r.redis.Del(ctx, quoteActiveKey(customerID)).Err(); err != nil {
		slog.Error("orderQuotes ClearActiveID failed", "customerID", customerID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}
