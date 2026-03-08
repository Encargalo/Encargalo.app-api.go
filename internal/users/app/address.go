package app

import (
	"context"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
)

type customersAddressApp struct {
	repo ports.CustomersAddressRepo
}

func NewCustomersAddressApp(repo ports.CustomersAddressRepo) ports.CustomersAddressApp {
	return &customersAddressApp{repo}
}

func (c *customersAddressApp) RegisterAddress(ctx context.Context, customerID uuid.UUID, address dto.Address) error {

	addresModel := models.Address{}
	if err := addresModel.BuildToModel(customerID, address); err != nil {
		return err
	}

	return c.repo.RegisterAddress(ctx, addresModel)
}

func (c *customersAddressApp) SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (models.Addresses, error) {
	return c.repo.SearchAllAddress(ctx, customer_id)
}

func (c *customersAddressApp) DeleteAddress(ctx context.Context, customer_id uuid.UUID, address_id string) error {
	return c.repo.DeleteAddress(ctx, customer_id, address_id)
}
