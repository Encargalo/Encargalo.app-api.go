package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"github.com/redis/go-redis/v9"
)

const (
	bcvRateKey = "bcv:usd:rate"
	bcvRateTTL = 7 * 24 * time.Hour
)

type cacheRepo struct {
	client *redis.Client
}

func NewCacheRepo(client *redis.Client) ports.BcvCache {
	return &cacheRepo{client: client}
}

func (r *cacheRepo) Save(ctx context.Context, rate *models.ExchangeRate) error {
	data, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("bcv cache: marshal: %w", err)
	}
	return r.client.Set(ctx, bcvRateKey, data, bcvRateTTL).Err()
}

func (r *cacheRepo) GetLatest(ctx context.Context) (*models.ExchangeRate, error) {
	data, err := r.client.Get(ctx, bcvRateKey).Bytes()
	if err != nil {
		return nil, err
	}
	var rate models.ExchangeRate
	if err := json.Unmarshal(data, &rate); err != nil {
		return nil, fmt.Errorf("bcv cache: unmarshal: %w", err)
	}
	return &rate, nil
}
