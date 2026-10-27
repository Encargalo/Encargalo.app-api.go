package repo

import (
	"context"

	"Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/uptrace/bun"
)

type fareQuotesRepo struct {
	db *bun.DB
}

func NewFareQuotesRepo(db *bun.DB) ports.OrderFareQuotesRepo {
	return &fareQuotesRepo{db: db}
}

// Insert es idempotente: si el quote_id ya existe no hace nada, así un reintento
// no duplica la fila ni rompe la cotización.
func (r *fareQuotesRepo) Insert(ctx context.Context, quote *models.OrderFareQuote) error {
	_, err := r.db.NewInsert().
		Model(quote).
		On("CONFLICT (quote_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return errcustom.Unexpected(err)
	}
	return nil
}
