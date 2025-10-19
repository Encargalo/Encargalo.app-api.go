package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customersAddressRepo struct {
	db *bun.DB
}

func NewCustomersAddressRepo(db *bun.DB) ports.CustomersAddressRepo {
	return &customersAddressRepo{db}
}

func (c *customersAddressRepo) RegisterAddress(ctx context.Context, address models.Address) error {

	if _, err := c.db.NewInsert().Model(&address).Exec(ctx); err != nil {
		fmt.Println("error al insertar la direccion del customer - ", err.Error())
		return fmt.Errorf("unexpected error")
	}

	return nil
}

func (c *customersAddressRepo) SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (dto.Addresses, error) {

	var addresses models.Addresses

	if err := c.db.NewSelect().Model(&addresses).
		Where("customer_id = ?", customer_id).
		Where("deleted_at IS NULL").Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return dto.Addresses{}, errors.New("not found")
		}
		fmt.Println(err.Error())
		return dto.Addresses{}, errors.New("unexpected error")

	}

	return addresses.ToDomainDTO(), nil
}

func (c *customersAddressRepo) DeleteAddress(ctx context.Context, customer_id, address_id uuid.UUID) error {
	_, err := c.db.NewDelete().
		Model(&models.Address{}).
		Where("id = ?", address_id).
		Where("customer_id = ?", customer_id).
		Exec(ctx)

	if err != nil {
		fmt.Println(err.Error())
		return errors.New("unexpected error")
	}

	return nil
}
