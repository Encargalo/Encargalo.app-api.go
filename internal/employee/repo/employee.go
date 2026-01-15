package repo

import (
	"context"
	"database/sql"
	"log/slog"

	"Encargalo.app-api.go/internal/employee/domain/dto"
	"Encargalo.app-api.go/internal/employee/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type employeeRepo struct {
	db *bun.DB
}

func New(db *bun.DB) ports.EmployeeRepo {
	return &employeeRepo{db}
}

func (e *employeeRepo) SearchEmployeeByPhone(ctx context.Context, phone string) (models.Account, error) {

	var account models.Account
	err := e.db.NewSelect().Model(&account).Where("phone = ?", phone).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Account{}, errcustom.ErrNotFound
		}
		return models.Account{}, errcustom.ErrUnexpectedError
	}

	return account, nil
}

func (e *employeeRepo) SearchEmployeeByID(ctx context.Context, employeeID uuid.UUID) (dto.EmployeeResponse, error) {
	var account models.Account
	err := e.db.NewSelect().
		Model(&account).
		Relation("Shop").
		Where("?TableAlias.id = ?", employeeID).
		Scan(ctx)

	if err != nil {
		if err == sql.ErrNoRows {
			return dto.EmployeeResponse{}, errcustom.ErrNotFound
		}
		slog.Error("Error searching employee by ID", "error", err, "employeeID", employeeID)
		return dto.EmployeeResponse{}, errcustom.ErrUnexpectedError
	}
	return account.ToDomainDTO(), nil
}
