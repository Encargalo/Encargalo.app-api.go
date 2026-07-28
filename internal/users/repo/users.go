package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type accountWithRole struct {
	models.Account
	ShopID   uuid.UUID `bun:"shop_id"`
	RoleName string    `bun:"role_name"`
}

type customersRepo struct {
	db *bun.DB
}

func NewUsersRepo(db *bun.DB) ports.UsersRepo {
	return &customersRepo{db}
}

func (c *customersRepo) RegisterCustomer(ctx context.Context, customer *models.Account) error {

	if err := c.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {

		if _, err := tx.NewInsert().Model(customer).Exec(ctx); err != nil {
			slog.Error("error al insertar el customer", "error", err)
			return errcustom.ErrUnexpectedError
		}

		activationAccount := new(models.ActivateAccount)
		activationAccount.BuildActivateAccount(customer.ID)

		if _, err := tx.NewInsert().Model(activationAccount).Exec(ctx); err != nil {
			slog.Error("error al registrar el codigo de activación", "error", err)
			return errcustom.ErrUnexpectedError
		}

		if _, err := tx.NewRaw(`
			INSERT INTO users.user_roles (user_id, role_id)
			SELECT ?, id FROM users.roles WHERE name = 'Customer'
		`, customer.ID).Exec(ctx); err != nil {
			slog.Error("error al asignar rol Customer", "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// RegisterCustomerFromGoogle inserta una cuenta cuya identidad ya fue
// verificada por Google: a diferencia de RegisterCustomer, no crea fila en
// activate_account (no hace falta OTP) porque el email ya está confirmado.
func (c *customersRepo) RegisterCustomerFromGoogle(ctx context.Context, customer *models.Account) error {

	if err := c.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {

		if _, err := tx.NewInsert().Model(customer).Exec(ctx); err != nil {
			slog.Error("error al insertar el customer de google", "error", err)
			return errcustom.ErrUnexpectedError
		}

		if _, err := tx.NewRaw(`
			INSERT INTO users.user_roles (user_id, role_id)
			SELECT ?, id FROM users.roles WHERE name = 'Customer'
		`, customer.ID).Exec(ctx); err != nil {
			slog.Error("error al asignar rol Customer", "error", err)
			return errcustom.ErrUnexpectedError
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (c *customersRepo) SearchCustomerBy(ctx context.Context, criteria dto.SearchCustomerBy) (*models.Account, error) {

	if criteria.ID == uuid.Nil && criteria.Phone == "" && criteria.Email == "" {
		return nil, errcustom.ErrNotFound
	}

	account := new(models.Account)

	query := c.db.NewSelect().Model(account)

	query.WhereGroup("AND", func(sq *bun.SelectQuery) *bun.SelectQuery {

		if criteria.ID != uuid.Nil {
			sq.Where("id = ?", criteria.ID)
		}

		if criteria.Phone != "" {
			if criteria.ID != uuid.Nil {
				sq.WhereOr("phone = ?", criteria.Phone)
			} else {
				sq.Where("phone = ?", criteria.Phone)
			}
		}

		if criteria.Email != "" {
			if criteria.ID != uuid.Nil || criteria.Phone != "" {
				sq.WhereOr("LOWER(email) = LOWER(?)", criteria.Email)
			} else {
				sq.Where("LOWER(email) = LOWER(?)", criteria.Email)
			}
		}
		return sq
	})

	err := query.Where("deleted_at IS NULL").Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.ErrUnexpectedError
	}

	return account, nil
}

func (c *customersRepo) SearchCustomerByPhoneAndNotIDEquals(ctx context.Context, customer_id uuid.UUID, phone string) (*models.Account, error) {

	account := new(models.Account)

	err := c.db.NewSelect().
		Model(account).
		Where("phone = ? AND id != ?", phone, customer_id).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.ErrUnexpectedError
	}

	return account, nil

}

func (c *customersRepo) UpdateCustomer(ctx context.Context, customer_id uuid.UUID, customer *models.Account) error {

	if _, err := c.db.NewUpdate().Model(customer).OmitZero().Where("id = ?", customer_id).Exec(ctx); err != nil {
		slog.Error("error al modificar la información del cliente", "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (c *customersRepo) UpdatePassword(ctx context.Context, customer_id uuid.UUID, customer *models.Account) error {

	if _, err := c.db.NewUpdate().Model(customer).OmitZero().Where("id = ?", customer_id).Exec(ctx); err != nil {
		slog.Error("error al modificar la contraseña del customer", "error", err)
		return errcustom.ErrUnexpectedError
	}

	return nil
}

func (c *customersRepo) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var rows []struct {
		Name string `bun:"name"`
	}

	err := c.db.NewSelect().
		TableExpr("users.user_roles ur").
		ColumnExpr("r.name AS name").
		Join("JOIN users.roles r ON r.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Where("ur.enabled = TRUE").
		Scan(ctx, &rows)

	if err != nil {
		return nil, errcustom.ErrUnexpectedError
	}

	roles := make([]string, len(rows))
	for i, r := range rows {
		roles[i] = r.Name
	}
	return roles, nil
}

func (c *customersRepo) SearchCustomerProfileByID(ctx context.Context, userID uuid.UUID) (*models.Account, error) {
	var result accountWithRole

	err := c.db.NewSelect().
		TableExpr("users.accounts AS ua").
		ColumnExpr("ua.*").
		ColumnExpr("COALESCE(ur.shop_id, '00000000-0000-0000-0000-000000000000'::uuid) AS shop_id").
		ColumnExpr("COALESCE(r.name, '') AS role_name").
		Join("LEFT JOIN users.user_roles ur ON ur.user_id = ua.id AND ur.enabled = TRUE").
		Join("LEFT JOIN users.roles r ON r.id = ur.role_id").
		Where("ua.id = ?", userID).
		Where("ua.deleted_at IS NULL").
		Limit(1).
		Scan(ctx, &result)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.ErrUnexpectedError
	}

	result.Account.ShopID = result.ShopID
	result.Role = result.RoleName

	if result.ShopID != (uuid.UUID{}) {
		type shopRow struct {
			ID     uuid.UUID `bun:"id"`
			Name   string    `bun:"name"`
			Opened bool      `bun:"opened"`
		}
		var shop shopRow
		if err := c.db.NewSelect().
			TableExpr("business.shops").
			ColumnExpr("id, name, opened").
			Where("id = ?", result.ShopID).
			Scan(ctx, &shop); err == nil {
			result.Shop = &dto.ShopInfo{
				ID:     shop.ID,
				Name:   shop.Name,
				Opened: shop.Opened,
			}
		}
	}

	return &result.Account, nil
}

func (c *customersRepo) SearchAccountByPhone(ctx context.Context, phone string, roles []string) (*models.Account, error) {

	var result accountWithRole

	err := c.db.NewSelect().
		TableExpr("users.accounts AS ua").
		ColumnExpr("ua.*").
		ColumnExpr("ur.shop_id AS shop_id").
		ColumnExpr("r.name AS role_name").
		Join("JOIN users.user_roles ur ON ur.user_id = ua.id").
		Join("JOIN users.roles r ON r.id = ur.role_id").
		Where("ua.phone = ?", phone).
		Where("r.name IN (?)", bun.In(roles)).
		Where("ua.deleted_at IS NULL").
		Where("ur.enabled = TRUE").
		Scan(ctx, &result)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errcustom.ErrNotFound
		}
		return nil, errcustom.ErrUnexpectedError
	}

	result.Account.ShopID = result.ShopID
	result.Role = result.RoleName

	return &result.Account, nil
}
