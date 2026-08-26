package app_test

import (
	"context"
	"testing"

	"Encargalo.app-api.go/internal/employee/app"
	"Encargalo.app-api.go/internal/employee/domain/dto"
	mocksemployee "Encargalo.app-api.go/internal/mocks/employee/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSearchEmployeeByID_DelegatesToRepo(t *testing.T) {
	repo := mocksemployee.NewEmployeeRepo(t)
	svc := app.New(repo)

	employeeID := uuid.New()
	repo.On("SearchEmployeeByID", context.Background(), employeeID).
		Return(dto.EmployeeResponse{ID: employeeID, Name: "Luigi"}, nil)

	resp, err := svc.SearchEmployeeByID(context.Background(), employeeID)

	require.NoError(t, err)
	require.Equal(t, "Luigi", resp.Name)
}

func TestSearchEmployeeByID_RepoError(t *testing.T) {
	repo := mocksemployee.NewEmployeeRepo(t)
	svc := app.New(repo)

	employeeID := uuid.New()
	repo.On("SearchEmployeeByID", context.Background(), employeeID).
		Return(dto.EmployeeResponse{}, errcustom.ErrNotFound)

	_, err := svc.SearchEmployeeByID(context.Background(), employeeID)

	require.ErrorIs(t, err, errcustom.ErrNotFound)
}
