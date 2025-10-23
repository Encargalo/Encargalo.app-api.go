package app

import (
	"context"
	"errors"
	"fmt"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"

	auth "Encargalo.app-api.go/internal/auth/domain/ports"

	"github.com/google/uuid"
)

type customersApp struct {
	repo ports.CustomersRepo
	pass bycript.Password
	auth auth.AuthApp
}

func NewCustomerApp(repo ports.CustomersRepo, pass bycript.Password, auth auth.AuthApp) ports.CustomersApp {
	return &customersApp{
		repo,
		pass,
		auth,
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

	_, err = c.repo.RegisterCustomer(ctx, &customerModel)
	if err != nil {
		return uuid.Nil, err
	}

	sessionID, err := c.auth.SignInCustomer(ctx, customer.Phone, customer.Password)
	if err != nil {
		fmt.Println("Error al iniciar session.")
	}

	return sessionID, nil
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
