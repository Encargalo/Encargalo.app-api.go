package app

import (
	"context"
	"log/slog"
	"time"

	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"

	"github.com/google/uuid"
)

type customersApp struct {
	repo   ports.UsersRepo
	pass   bycript.Password
	stream ports.StreamPublisher
}

func NewUserApp(repo ports.UsersRepo, pass bycript.Password, stream ports.StreamPublisher) ports.UsersApp {
	return &customersApp{
		repo,
		pass,
		stream,
	}
}

func (c *customersApp) publishUserUpdated(ctx context.Context, userID uuid.UUID) {
	if c.stream == nil {
		return
	}
	event := models.DataEventUserUpdated{}
	event.BuildFromUserID(userID.String())
	if err := c.stream.PublisherEvent(ctx, event); err != nil {
		slog.Error("error publishing user_updated event", "userID", userID, "error", err)
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

	if err := c.pass.HashPassword(&customer.Password); err != nil {
		return errcustom.ErrUnexpectedError
	}

	customerModel := models.Account{}
	customerModel.BuildCustomerRegisterModel(customer)

	err = c.repo.RegisterCustomer(ctx, &customerModel)
	if err != nil {
		return err
	}

	c.publishUserUpdated(ctx, customerModel.ID)
	return nil
}

func (c *customersApp) RegisterCustomerFromGoogle(ctx context.Context, customer dto.RegisterCustomerFromGoogle) (*models.Account, error) {

	customerModel := models.Account{}
	customerModel.BuildCustomerRegisterFromGoogleModel(customer)

	if err := c.repo.RegisterCustomerFromGoogle(ctx, &customerModel); err != nil {
		return nil, err
	}

	c.publishUserUpdated(ctx, customerModel.ID)
	return &customerModel, nil
}

func (c *customersApp) SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Account, error) {
	return c.repo.SearchCustomerBy(ctx, criteria)
}

func (c *customersApp) SearchCustomerProfileByID(ctx context.Context, userID uuid.UUID) (*models.Account, error) {
	return c.repo.SearchCustomerProfileByID(ctx, userID)
}

func (c *customersApp) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return c.repo.GetUserRoles(ctx, userID)
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

	if customer.Phone != nil {
		cust, err := c.repo.SearchCustomerByPhoneAndNotIDEquals(ctx, customer_id, *customer.Phone)
		if err != nil {
			if err != errcustom.ErrNotFound {
				return err
			}
		}

		if cust != nil {
			return errcustom.ErrPhoneAlreadyExist
		}
	}

	customerModel := models.Account{}
	customerModel.BuildCustomerUpdateModel(customer)

	if err := c.repo.UpdateCustomer(ctx, customer_id, &customerModel); err != nil {
		return err
	}
	c.publishUserUpdated(ctx, customer_id)
	return nil
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
		return errcustom.ErrNotFound
	}

	customerModel := models.Account{}
	customerModel.BuildCustomerUpdatePasswordModel(pass)

	if err := c.pass.HashPassword(&customerModel.Password); err != nil {
		return errcustom.ErrUnexpectedError
	}

	if err := c.repo.UpdatePassword(ctx, customer_id, &customerModel); err != nil {
		return err
	}
	c.publishUserUpdated(ctx, customer_id)
	return nil
}

func (c *customersApp) RequestAccountDeletion(ctx context.Context, userID uuid.UUID) error {
	if err := c.repo.RequestAccountDeletion(ctx, userID); err != nil {
		return err
	}
	c.publishUserUpdated(ctx, userID)
	return nil
}

func (c *customersApp) CancelPendingDeletionIfAny(ctx context.Context, userID uuid.UUID) error {
	if err := c.repo.CancelAccountDeletion(ctx, userID); err != nil {
		return err
	}
	c.publishUserUpdated(ctx, userID)
	return nil
}

func (c *customersApp) FinalizePendingDeletions(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-models.AccountDeletionGracePeriod)
	return c.repo.FinalizePendingDeletions(ctx, cutoff)
}
