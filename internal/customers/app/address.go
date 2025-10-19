package app

import (
	"context"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
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
	addresModel.BuildToModel(customerID, address)

	return c.repo.RegisterAddress(ctx, addresModel)
}

func (c *customersAddressApp) SearchAllAddress(ctx context.Context, customer_id uuid.UUID) (dto.Addresses, error) {

	return c.repo.SearchAllAddress(ctx, customer_id)
}

func (c *customersAddressApp) DeleteAddress(ctx context.Context, customer_id, address_id uuid.UUID) error {
	return c.repo.DeleteAddress(ctx, customer_id, address_id)
}
