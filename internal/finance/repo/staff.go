package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/finance/domain/models"
	"Encargalo.app-api.go/internal/finance/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type staffFinanceRepo struct {
	db *bun.DB
}

func NewStaffFinanceRepo(db *bun.DB) ports.StaffFinanceRepo {
	return &staffFinanceRepo{db: db}
}

func (r *staffFinanceRepo) GetBalance(ctx context.Context, entityID uuid.UUID, entityType string) (models.Balance, error) {
	var b models.Balance
	err := r.db.NewSelect().
		Model(&b).
		Where("entity_id = ? AND entity_type = ?", entityID, entityType).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Balance{Currency: models.NativeCurrency(entityType)}, nil
		}
		slog.Error("finance GetBalance failed", "entityID", entityID, "entityType", entityType, "error", err)
		return models.Balance{}, errcustom.ErrUnexpectedError
	}
	if b.Currency == "" {
		b.Currency = models.NativeCurrency(entityType)
	}
	return b, nil
}

func (r *staffFinanceRepo) RecordMovement(ctx context.Context, entityID uuid.UUID, entityType string, delta float64, minBalance float64, movement models.Transaction) (float64, error) {
	var newAmount float64

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var b models.Balance
		err := tx.NewSelect().
			Model(&b).
			Where("entity_id = ? AND entity_type = ?", entityID, entityType).
			For("UPDATE").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			slog.Error("finance RecordMovement: select for update failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		if minBalance > 0 && b.Amount < minBalance {
			return errcustom.ErrInsufficientBalance
		}

		newAmount = b.Amount + delta
		currency := movement.Currency
		if currency == "" {
			currency = models.NativeCurrency(entityType)
		}

		_, err = tx.NewInsert().
			Model(&models.Balance{
				EntityID:   entityID,
				EntityType: entityType,
				Amount:     newAmount,
				Currency:   currency,
			}).
			On("CONFLICT (entity_id, entity_type) DO UPDATE").
			Set("amount = EXCLUDED.amount, updated_at = now()").
			Exec(ctx)
		if err != nil {
			slog.Error("finance RecordMovement: balance upsert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		movement.EntityID = entityID
		movement.EntityType = entityType
		movement.Amount = delta
		if movement.Currency == "" {
			movement.Currency = currency
		}
		if movement.CreatedAt.IsZero() {
			movement.CreatedAt = time.Now()
		}

		if _, err = tx.NewInsert().Model(&movement).Exec(ctx); err != nil {
			slog.Error("finance RecordMovement: transaction insert failed", "entityID", entityID, "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return newAmount, nil
}

func (r *staffFinanceRepo) ListPayables(ctx context.Context, entityType string) ([]models.PayableRow, error) {
	var rows []models.PayableRow
	q := r.db.NewSelect().
		TableExpr("finance.balances AS b").
		ColumnExpr("b.entity_id").
		ColumnExpr("b.amount").
		ColumnExpr("b.currency").
		ColumnExpr("b.updated_at").
		Where("b.entity_type = ?", entityType).
		Where("b.amount > 0").
		OrderExpr("b.amount DESC")

	switch entityType {
	case models.EntityTypeShop:
		q = q.ColumnExpr("s.name").
			Join("JOIN business.shops AS s ON s.id = b.entity_id")
	default:
		q = q.ColumnExpr("a.name").
			ColumnExpr("a.sur_name").
			ColumnExpr("a.phone").
			Join("JOIN users.accounts AS a ON a.id = b.entity_id")
	}

	if err := q.Scan(ctx, &rows); err != nil {
		slog.Error("finance ListPayables failed", "entityType", entityType, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return rows, nil
}
