package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/models"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type customersRepo struct {
	db *bun.DB
}

func NewCustomersRepo(db *bun.DB) ports.CustomersRepo {
	return &customersRepo{db}
}

func (c *customersRepo) RegisterCustomer(ctx context.Context, customer *models.Accounts) (*models.Accounts, error) {

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Println("error iniciando transacción: %w", err)
		return nil, errors.New("unexpected error")
	}

	if _, err := tx.NewInsert().Model(customer).Returning("*").Exec(ctx); err != nil {
		_ = tx.Rollback()
		fmt.Println("error al insertar el customer")
		return nil, errors.New("unexpected error")
	}

	activationAccount := new(models.ActivateAccount)
	activationAccount.BuildActivateAccount(customer.ID)

	if _, err := tx.NewInsert().Model(activationAccount).Exec(ctx); err != nil {
		_ = tx.Rollback()
		fmt.Println("error al registrar el codigo de activación")
		return nil, errors.New("unexpected error")
	}

	if err := tx.Commit(); err != nil {
		fmt.Println("error al confirmar transacción: %w", err)
		return nil, errors.New("unexpected error")
	}

	return customer, nil
}

func (c *customersRepo) SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Accounts, error) {

	if criteria.ID == uuid.Nil && criteria.Phone == "" {
		return nil, errors.New("no search criteria provided")
	}

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
			return nil, errors.New("not found")
		}
		return nil, errors.New("unexpected error")
	}

	return account, nil

}

func (c *customersRepo) UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer *models.Accounts) error {

	if _, err := c.db.NewUpdate().Model(customer).OmitZero().Where("id = ?", customer_id).Exec(ctx); err != nil {
		fmt.Println(err.Error())
		return errors.New("unexpected error")
	}
	return nil
}

func (c *customersRepo) UpdatePassword(ctx context.Context, customer_id uuid.UUID, customer *models.Accounts) error {

	if _, err := c.db.NewUpdate().Model(customer).OmitZero().Where("id = ?", customer_id).Exec(ctx); err != nil {
		fmt.Println(err.Error())
		return errors.New("unexpected error")
	}

	return nil
}
