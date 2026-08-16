package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shops/domain/models"
	"Encargalo.app-api.go/internal/shops/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type shopFinanceRepo struct {
	db *bun.DB
}

func NewShopFinanceRepo(db *bun.DB) ports.ShopFinanceRepo {
	return &shopFinanceRepo{db}
}

func (r *shopFinanceRepo) GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.ShopBalance, error) {
	var b models.ShopBalance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ShopBalance{Currency: models.CurrencyUSD}, nil
		}
		slog.Error("shop GetBalance: failed", "entityID", entityID, "error", err)
		return models.ShopBalance{}, errcustom.ErrUnexpectedError
	}
	if b.Currency == "" {
		b.Currency = models.CurrencyUSD
	}
	return b, nil
}

func (r *shopFinanceRepo) RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.ShopTransaction) (float64, error) {
	var newAmount float64

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.ShopBalance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", entityID, entityType).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("shop RecordMovement: select for update failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if minBalance > 0 && b.Amount < minBalance {
			return errcustom.ErrInsufficientBalance
		}

		newAmount = b.Amount + delta

		_, err = tx.NewInsert().
			Model(&models.ShopBalance{
				EntityID:   entityID,
				EntityType: entityType,
				Amount:     newAmount,
				Currency:   models.CurrencyUSD,
			}).
			On("CONFLICT (entity_id, entity_type) DO UPDATE").
			Set("amount = EXCLUDED.amount, updated_at = now()").
			Exec(ctx)
		if err != nil {
			slog.Error("shop RecordMovement: balance upsert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		movement.EntityID = entityID
		movement.EntityType = entityType
		movement.Amount = delta
		if movement.Currency == "" {
			movement.Currency = models.CurrencyUSD
		}
		if movement.CreatedAt.IsZero() {
			movement.CreatedAt = time.Now()
		}

		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("shop RecordMovement: transaction insert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return newAmount, nil
}

func (r *shopFinanceRepo) ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.ShopTransaction, int, error) {
	if limit > 50 {
		limit = 50
	}
	offset := (page - 1) * limit

	var txs []models.ShopTransaction
	total, err := r.db.NewSelect().
		Model(&txs).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		OrderExpr("created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		slog.Error("shop ListTransactions: failed", "entityID", entityID, "error", err)
		return nil, 0, errcustom.ErrUnexpectedError
	}

	return txs, total, nil
}
