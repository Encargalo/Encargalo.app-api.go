package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customerFinanceRepo struct {
	db *bun.DB
}

func NewCustomerFinanceRepo(db *bun.DB) ports.CustomerFinanceRepo {
	return &customerFinanceRepo{db}
}

func (r *customerFinanceRepo) GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (float64, error) {
	var b models.Balance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		slog.Error("customer GetBalance: failed", "entityID", entityID, "error", err)
		return 0, errcustom.ErrUnexpectedError
	}
	return b.Amount, nil
}

func (r *customerFinanceRepo) RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.Transaction) (float64, error) {
	var newAmount float64

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.Balance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", entityID, entityType).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("customer RecordMovement: select failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if minBalance > 0 && b.Amount < minBalance {
			return errcustom.ErrInsufficientBalance
		}

		newAmount = b.Amount + delta

		_, err = tx.NewInsert().
			Model(&models.Balance{
				EntityID:   entityID,
				EntityType: entityType,
				Amount:     newAmount,
			}).
			On("CONFLICT (entity_id, entity_type) DO UPDATE").
			Set("amount = EXCLUDED.amount, updated_at = now()").
			Exec(ctx)
		if err != nil {
			slog.Error("customer RecordMovement: balance upsert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		movement.EntityID = entityID
		movement.EntityType = entityType
		movement.Amount = delta
		if movement.ID == uuid.Nil {
			movement.ID = uuid.New()
		}

		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("customer RecordMovement: tx insert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return newAmount, nil
}

func (r *customerFinanceRepo) ListTransactions(ctx context.Context, entityID uuid.UUID, entityType string, page, limit int) ([]models.Transaction, int, error) {
	if limit > 50 {
		limit = 50
	}
	offset := (page - 1) * limit

	var txs []models.Transaction
	total, err := r.db.NewSelect().
		Model(&txs).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		OrderExpr("created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		slog.Error("customer ListTransactions: failed", "entityID", entityID, "error", err)
		return nil, 0, errcustom.ErrUnexpectedError
	}

	return txs, total, nil
}
