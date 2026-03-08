package repo

import (
	"context"
	"database/sql"
	"log/slog"

	"Encargalo.app-api.go/internal/employee/domain/dto"
	"Encargalo.app-api.go/internal/employee/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	usersModels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type employeeRepo struct {
	db *bun.DB
}

func New(db *bun.DB) ports.EmployeeRepo {
	return &employeeRepo{db}
}

type accountWithRole struct {
	usersModels.Account
	ShopID   uuid.UUID `bun:"shop_id"`
	RoleName string    `bun:"role_name"`
}

func safeDerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (e *employeeRepo) SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (dto.EmployeeResponse, error) {
	var result accountWithRole

	err := e.db.NewSelect().
		TableExpr("users.accounts AS ua").
		ColumnExpr("ua.*").
		ColumnExpr("ur.shop_id AS shop_id").
		ColumnExpr("r.name AS role_name").
		Join("JOIN users.user_roles ur ON ur.user_id = ua.id").
		Join("JOIN users.roles r ON r.id = ur.role_id").
		Where("ua.id = ?", employeeID).
		Where("r.name IN (?)", bun.In([]string{"Order Manager", "Owner"})).
		Where("ua.deleted_at IS NULL").
		Where("ur.enabled = TRUE").
		Scan(ctx, &result)

	if err != nil {
		if err == sql.ErrNoRows {
			return dto.EmployeeResponse{}, errcustom.ErrNotFound
		}
		slog.Error("Error searching employee by ID", "error", err, "employeeID", employeeID)
		return dto.EmployeeResponse{}, errcustom.ErrUnexpectedError
	}

	result.Account.ShopID = result.ShopID
	result.Role = result.RoleName

	var shop models.EmployeeShop
	if err := e.db.NewSelect().
		Model(&shop).
		Where("id = ?", result.ShopID).
		Scan(ctx); err != nil && err != sql.ErrNoRows {
		slog.Error("Error searching shop for employee", "error", err)
	}

	return dto.EmployeeResponse{
		ID:     result.ID,
		ShopID: result.Account.ShopID,
		Name:   result.Name,
		Email:  safeDerefString(result.Email),
		Phone:  result.Phone,
		Role:   result.Role,
		Shop:   shop.ToDomainDTO(),
	}, nil
}
