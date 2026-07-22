package repo

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customersAddressRepo struct {
	db *bun.DB
}

func NewUsersAddressRepo(db *bun.DB) ports.UsersAddressRepo {
	return &customersAddressRepo{db}
}

func (c *customersAddressRepo) RegisterAddress(ctx context.Context, address models.Address) error {

	if _, err := c.db.NewInsert().Model(&address).Exec(ctx); err != nil {
		slog.Error("error al insertar la dirección del cliente", "error", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (c *customersAddressRepo) SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (models.Addresses, error) {

	var addresses models.Addresses

	if err := c.db.NewSelect().Model(&addresses).Where("customer_id = ?", customer_id).Scan(ctx); err != nil {
		slog.Error("error al buscar las direcciones del cliente", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	return addresses, nil

}

func (c *customersAddressRepo) DeleteAddress(ctx context.Context, customer_id uuid.UUID, address_id string) error {

	if _, err := c.db.NewDelete().Model(&models.Address{}).Where("customer_id = ? AND id = ?", customer_id, address_id).Exec(ctx); err != nil {
		slog.Error("error al eliminar la dirección del cliente", "error", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}
