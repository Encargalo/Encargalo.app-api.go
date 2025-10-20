package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/pkg/logs"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customersRepo struct {
	db        *bun.DB
	slackLogs logs.Logs
}

func NewCustomersRepo(db *bun.DB, slackLogs logs.Logs) ports.CustomersRepo {
	return &customersRepo{db, slackLogs}
}

func (c *customersRepo) RegisterCustomer(ctx context.Context, customer *models.Accounts) (*models.Accounts, error) {

	if err := c.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {

		if _, err := tx.NewInsert().Model(customer).Returning("*").Exec(ctx); err != nil {
			slog.Error("error al insertar el customer", "error", err)
			c.slackLogs.Slack(err)
			return errcustom.ErrUnexpectedError
		}

		activationAccount := new(models.ActivateAccount)
		activationAccount.BuildActivateAccount(customer.ID)

		if _, err := tx.NewInsert().Model(activationAccount).Exec(ctx); err != nil {
			slog.Error("error al registrar el codigo de activación", "error", err)
			c.slackLogs.Slack(err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return customer, nil
}

func (c *customersRepo) SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Accounts, error) {

	account := new(models.Accounts)

	err := c.db.NewSelect().
		Model(account).
		WhereGroup("or", func(sq *bun.SelectQuery) *bun.SelectQuery {
			if criteria.ID != uuid.Nil {
				sq = sq.Where("id = ?", criteria.ID)
			}
			if criteria.Phone != "" {
				sq = sq.Where("phone = ?", criteria.Phone)
			}
			return sq
		}).
		Where("deleted_at IS NULL").
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		c.slackLogs.Slack(err)
		return nil, errcustom.ErrUnexpectedError
	}

	return account, nil

}

func (c *customersRepo) SearchCustomerByPhoneAndNotIDEquals(ctx context.Context, customer_id uuid.UUID, phone string) (*models.Accounts, error) {

	account := new(models.Accounts)

	err := c.db.NewSelect().
		Model(account).
		Where("phone = ? AND id != ?", phone, customer_id).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errcustom.ErrNotFound
		}
		c.slackLogs.Slack(err)
		return nil, errcustom.ErrUnexpectedError
	}

	return account, nil

}

func (c *customersRepo) UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer *models.Accounts) error {

	if _, err := c.db.NewUpdate().Model(customer).OmitZero().Where("id = ?", customer_id).Exec(ctx); err != nil {
		slog.Error("error al modificar la información del cliente", "error", err)
		c.slackLogs.Slack(err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (c *customersRepo) UpdatePassword(ctx context.Context, customer_id uuid.UUID, customer *models.Accounts) error {

	if _, err := c.db.NewUpdate().Model(customer).OmitZero().Where("id = ?", customer_id).Exec(ctx); err != nil {
		slog.Error("error al modificar la contraseña del customer", "error", err)
		c.slackLogs.Slack(err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}
