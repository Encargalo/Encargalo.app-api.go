package app

import (
	"context"
	"errors"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"

	"github.com/google/uuid"
)

type customersApp struct {
	repo ports.CustomersRepo
	pass bycript.Password
}

func NewCustomerApp(repo ports.CustomersRepo, pass bycript.Password) ports.CustomersApp {
	return &customersApp{
		repo,
		pass,
	}
}

func (c *customersApp) RegisterCustomer(ctx context.Context, customer dto.RegisterCustomer) (uuid.UUID, error) {

	custo, err := c.SearchCustomerBy(ctx, dto.SearchCustomerBy{Phone: customer.Phone})
	if err != nil {
		if err == errors.New("not found") {
			return uuid.Nil, err
		}
	}

	if custo != nil {
		return uuid.Nil, errors.New("phone al ready exist")
	}

	c.pass.HashPassword(&customer.Password)

	customerModel := models.Accounts{}
	customerModel.BuildCustomerRegisterModel(customer)

	custo, err = c.repo.RegisterCustomer(ctx, &customerModel)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.New(), nil
}

func (c *customersApp) SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Accounts, error) {
	return c.repo.SearchCustomerBy(ctx, criteria)
}

func (c *customersApp) UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer dto.UpdateCustomer) error {

	criteria := dto.SearchCustomerBy{
		ID: customer_id,
	}

	_, err := c.SearchCustomerBy(ctx, criteria)
	if err != nil {
		return err
	}

	cust, err := c.repo.SearchCustomerByPhoneAndNotIDEquals(ctx, customer_id, customer.Phone)
	if err != nil {
		if err.Error() != "not found." {
			return err
		}
	}

	if cust != nil {
		return errors.New("phone al ready exist")
	}

	customerModel := models.Accounts{}
	customerModel.BuildCustomerUpdateModel(customer)

	return c.repo.UpdateCustomer(ctx, customer_id, &customerModel)
}

func (c *customersApp) UpdatePassword(ctx context.Context, customer_id uuid.UUID, pass dto.UpdatePassword) error {

	criteria := dto.SearchCustomerBy{
		ID: customer_id,
	}

	customer, err := c.SearchCustomerBy(ctx, criteria)
	if err != nil {
		return err
	}

	if customer == nil {
		return errors.New("not found")
	}

	customerModel := models.Accounts{}
	customerModel.BuildCustomerUpdatePasswordModel(pass)

	c.pass.HashPassword(&customerModel.Password)

	return c.repo.UpdatePassword(ctx, customer_id, &customerModel)

}
