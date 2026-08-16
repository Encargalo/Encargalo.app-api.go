package repo

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/pkg/cursor"
	"Encargalo.app-api.go/internal/pkg/sqlutil"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/models"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type staffUsersRepo struct {
	db *bun.DB
}

func NewStaffUsersRepo(db *bun.DB) ports.StaffUsersRepo {
	return &staffUsersRepo{db: db}
}

func (r *staffUsersRepo) accountsSelect() *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("users.accounts AS a").
		ColumnExpr("a.*").
		Where("a.deleted_at IS NULL")
}

func (r *staffUsersRepo) applyListFilters(q *bun.SelectQuery, query, status string) *bun.SelectQuery {
	switch status {
	case "active":
		q = q.Where("a.enabled = TRUE")
	case "blocked":
		q = q.Where("a.enabled = FALSE")
	}

	query = strings.TrimSpace(query)
	if query != "" {
		pattern := sqlutil.ILIKEContains(query)
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr("a.name ILIKE ? ESCAPE '\\'", pattern).
				WhereOr("a.sur_name ILIKE ? ESCAPE '\\'", pattern).
				WhereOr("COALESCE(a.email, '') ILIKE ? ESCAPE '\\'", pattern).
				WhereOr("COALESCE(a.phone, '') ILIKE ? ESCAPE '\\'", pattern)
		})
	}
	return q
}

func (r *staffUsersRepo) ListManagedCustomers(ctx context.Context, query, status string, limit int, after *cursor.Key) ([]models.Account, error) {
	q := r.accountsSelect()
	q = r.applyListFilters(q, query, status)

	if after != nil {
		q = q.Where("(a.created_at, a.id) < (?, ?)", after.CreatedAt, after.ID)
	}

	var rows []models.Account
	err := q.
		OrderExpr("a.created_at DESC, a.id DESC").
		Limit(limit).
		Scan(ctx, &rows)
	if err != nil {
		slog.Error("staffUsers ListManagedCustomers failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return rows, nil
}

func (r *staffUsersRepo) FindManagedCustomerByPhone(ctx context.Context, phone string) (*models.Account, error) {
	phone = strings.TrimSpace(phone)
	acc := new(models.Account)
	err := r.accountsSelect().
		Where("a.phone = ?", phone).
		Limit(1).
		Scan(ctx, acc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("staffUsers FindManagedCustomerByPhone failed", "phone", phone, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return acc, nil
}

func (r *staffUsersRepo) GetManagedCustomerByID(ctx context.Context, userID uuid.UUID) (*models.Account, error) {
	acc := new(models.Account)
	err := r.accountsSelect().
		Where("a.id = ?", userID).
		Limit(1).
		Scan(ctx, acc)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("staffUsers GetManagedCustomerByID failed", "userID", userID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return acc, nil
}

func (r *staffUsersRepo) SearchCustomerByEmailAndNotIDEquals(ctx context.Context, userID uuid.UUID, email string) (*models.Account, error) {
	acc := new(models.Account)
	err := r.db.NewSelect().
		Model(acc).
		Where("LOWER(email) = LOWER(?)", strings.TrimSpace(email)).
		Where("id != ?", userID).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errcustom.ErrNotFound
		}
		slog.Error("staffUsers SearchCustomerByEmailAndNotIDEquals failed", "userID", userID, "error", err)
		return nil, errcustom.ErrUnexpectedError
	}
	return acc, nil
}

func (r *staffUsersRepo) UpdateManagedCustomer(ctx context.Context, userID uuid.UUID, account *models.Account) error {
	account.UpdatedAt = time.Now()
	if _, err := r.db.NewUpdate().
		Model(account).
		Column("name", "sur_name", "email", "updated_at").
		Where("id = ?", userID).
		Where("deleted_at IS NULL").
		Exec(ctx); err != nil {
		slog.Error("staffUsers UpdateManagedCustomer failed", "userID", userID, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *staffUsersRepo) SetAccountEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	if _, err := r.db.NewUpdate().
		Model((*models.Account)(nil)).
		Set("enabled = ?", enabled).
		Set("updated_at = now()").
		Where("id = ?", userID).
		Where("deleted_at IS NULL").
		Exec(ctx); err != nil {
		slog.Error("staffUsers SetAccountEnabled failed", "userID", userID, "enabled", enabled, "error", err)
		return errcustom.ErrUnexpectedError
	}
	return nil
}

func (r *staffUsersRepo) GetRolesForUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}

	var rows []struct {
		UserID uuid.UUID `bun:"user_id"`
		Name   string    `bun:"name"`
	}

	err := r.db.NewSelect().
		TableExpr("users.user_roles ur").
		ColumnExpr("ur.user_id").
		ColumnExpr("ro.name").
		Join("JOIN users.roles ro ON ro.id = ur.role_id").
		Where("ur.user_id IN (?)", bun.In(userIDs)).
		Where("ur.enabled = TRUE").
		OrderExpr("ro.name ASC").
		Scan(ctx, &rows)
	if err != nil {
		slog.Error("staffUsers GetRolesForUserIDs failed", "error", err)
		return nil, errcustom.ErrUnexpectedError
	}

	for _, row := range rows {
		out[row.UserID] = append(out[row.UserID], row.Name)
	}
	for _, id := range userIDs {
		if _, ok := out[id]; !ok {
			out[id] = []string{}
		}
	}
	return out, nil
}
