package repo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/trips/domain/models"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const quoteKeyPrefix = "trip:quote:"

type tripQuotesRepo struct {
	redis *redis.Client
}

func NewTripQuotesRepo(redis *redis.Client) ports.TripQuotesRepo {
	return &tripQuotesRepo{redis: redis}
}

// quoteKey incluye customerID: un quote_id ajeno a ese customer directamente
// no matchea ninguna key, así GetAndConsume nunca borra la cotización de otro.
func quoteKey(customerID, quoteID uuid.UUID) string {
	return quoteKeyPrefix + customerID.String() + ":" + quoteID.String()
}

func (r *tripQuotesRepo) Save(ctx context.Context, quote *models.TripQuote, ttl time.Duration) error {
	raw, err := json.Marshal(quote)
	if err != nil {
		slog.Error("tripQuotes Save marshal failed", "error", err)
		return errcustom.ErrUnexpectedError
	}
	if err := r.redis.Set(ctx, quoteKey(quote.CustomerID, quote.QuoteID), raw, ttl).Err(); err != nil {
		slog.Error("tripQuotes Save redis failed", "quoteID", quote.QuoteID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *tripQuotesRepo) Get(ctx context.Context, customerID, quoteID uuid.UUID) (*models.TripQuote, error) {
	raw, err := r.redis.Get(ctx, quoteKey(customerID, quoteID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errcustom.ErrQuoteNotFound
		}
		slog.Error("tripQuotes Get redis failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	var quote models.TripQuote
	if err := json.Unmarshal([]byte(raw), &quote); err != nil {
		slog.Error("tripQuotes Get unmarshal failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &quote, nil
}

// GetAndConsume usa GETDEL: lectura y borrado atómicos, así un QuoteID solo sirve una vez.
func (r *tripQuotesRepo) GetAndConsume(ctx context.Context, customerID, quoteID uuid.UUID) (*models.TripQuote, error) {
	raw, err := r.redis.GetDel(ctx, quoteKey(customerID, quoteID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errcustom.ErrQuoteNotFound
		}
		slog.Error("tripQuotes GetAndConsume redis failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	var quote models.TripQuote
	if err := json.Unmarshal([]byte(raw), &quote); err != nil {
		slog.Error("tripQuotes GetAndConsume unmarshal failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &quote, nil
}
