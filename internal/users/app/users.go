package app

import (
	"context"
	"errors"

	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"

	"github.com/google/uuid"
)

type customersApp struct {
	repo ports.UsersRepo
	pass bycript.Password
}

func NewUserApp(repo ports.UsersRepo, pass bycript.Password) ports.UsersApp {
	return &customersApp{
		repo,
		pass,
	}
}

func (c *customersApp) RegisterCustomer(ctx context.Context, customer dto.RegisterCustomer) error {

	custo, err := c.SearchCustomerBy(ctx, dto.SearchCustomerBy{Phone: customer.Phone})
	if err != nil {
		if err != errcustom.ErrNotFound {
			return err
		}
	}

	if custo != nil {
		return errcustom.ErrPhoneAlreadyExist
	}

	c.pass.HashPassword(&customer.Password)

	customerModel := models.Account{}
	customerModel.BuildCustomerRegisterModel(customer)

	err = c.repo.RegisterCustomer(ctx, &customerModel)
	if err != nil {
		return err
	}

	return nil
}

func (c *customersApp) SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Account, error) {
	return c.repo.SearchCustomerBy(ctx, criteria)
}

func (c *customersApp) SearchCustomerProfileByID(ctx context.Context, userID uuid.UUID) (*models.Account, error) {
	return c.repo.SearchCustomerProfileByID(ctx, userID)
}

func (c *customersApp) SearchAccountByPhone(ctx context.Context, phone string, roles []string) (*models.Account, error) {
	return c.repo.SearchAccountByPhone(ctx, phone, roles)
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
		if err != errcustom.ErrNotFound {
			return err
		}
	}

	if cust != nil {
		return errors.New("phone al ready exist")
	}

	customerModel := models.Account{}
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

	customerModel := models.Account{}
	customerModel.BuildCustomerUpdatePasswordModel(pass)

	c.pass.HashPassword(&customerModel.Password)

	return c.repo.UpdatePassword(ctx, customer_id, &customerModel)

}
