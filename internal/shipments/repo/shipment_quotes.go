package repo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shipments/domain/models"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const quoteKeyPrefix = "shipment:quote:"

type shipmentQuotesRepo struct {
	redis *redis.Client
}

func NewShipmentQuotesRepo(redis *redis.Client) ports.ShipmentQuotesRepo {
	return &shipmentQuotesRepo{redis: redis}
}

// quoteKey incluye customerID: un quote_id ajeno a ese customer directamente
// no matchea ninguna key, así GetAndConsume nunca borra la cotización de otro.
func quoteKey(customerID, quoteID uuid.UUID) string {
	return quoteKeyPrefix + customerID.String() + ":" + quoteID.String()
}

func (r *shipmentQuotesRepo) Save(ctx context.Context, quote *models.ShipmentQuote, ttl time.Duration) error {
	raw, err := json.Marshal(quote)
	if err != nil {
		slog.Error("shipmentQuotes Save marshal failed", "error", err)
		return errcustom.ErrUnexpectedError
	}
	if err := r.redis.Set(ctx, quoteKey(quote.CustomerID, quote.QuoteID), raw, ttl).Err(); err != nil {
		slog.Error("shipmentQuotes Save redis failed", "quoteID", quote.QuoteID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *shipmentQuotesRepo) Get(ctx context.Context, customerID, quoteID uuid.UUID) (*models.ShipmentQuote, error) {
	raw, err := r.redis.Get(ctx, quoteKey(customerID, quoteID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errcustom.ErrQuoteNotFound
		}
		slog.Error("shipmentQuotes Get redis failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	var quote models.ShipmentQuote
	if err := json.Unmarshal([]byte(raw), &quote); err != nil {
		slog.Error("shipmentQuotes Get unmarshal failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &quote, nil
}

// GetAndConsume usa GETDEL: lectura y borrado atómicos, así un QuoteID solo sirve una vez.
func (r *shipmentQuotesRepo) GetAndConsume(ctx context.Context, customerID, quoteID uuid.UUID) (*models.ShipmentQuote, error) {
	raw, err := r.redis.GetDel(ctx, quoteKey(customerID, quoteID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errcustom.ErrQuoteNotFound
		}
		slog.Error("shipmentQuotes GetAndConsume redis failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	var quote models.ShipmentQuote
	if err := json.Unmarshal([]byte(raw), &quote); err != nil {
		slog.Error("shipmentQuotes GetAndConsume unmarshal failed", "quoteID", quoteID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return &quote, nil
}
