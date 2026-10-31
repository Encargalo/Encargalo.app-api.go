package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customerContactLookup struct {
	db *bun.DB
}

func NewCustomerContactLookup(db *bun.DB) ports.CustomerContactLookup {
	return &customerContactLookup{db: db}
}

// GetByCustomerID lee directamente users.accounts, igual patrón que
// riders.repo.passengerLookup para trips, pero sin pasar por riders.
func (l *customerContactLookup) GetByCustomerID(ctx context.Context, customerID uuid.UUID) (string, string, error) {
	var acc usersModels.Account
	err := l.db.NewSelect().
		Model(&acc).
		Where("id = ?", customerID).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errcustom.ErrNotFound
		}
		slog.Error("shipments customerContactLookup failed", "customerID", customerID, "error", err)
		return "", "", errcustom.Unexpected(err)
	}
	phone := ""
	if acc.Phone != nil {
		phone = *acc.Phone
	}
	return acc.Name, phone, nil
}
