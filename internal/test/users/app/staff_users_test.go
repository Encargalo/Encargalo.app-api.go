package app_test

import (
	"context"
	"testing"

	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaffUsersApp_BlockUser_DisablesAccount(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo)

	staffID := uuid.New()
	userID := uuid.New()
	acc := &models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(acc, nil).Once()
	repo.On("SetAccountEnabled", context.Background(), userID, false).Return(nil).Once()
	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(&models.Account{ID: userID, Name: "Ana", Enabled: false}, nil).Once()
	repo.On("GetRolesForUserIDs", context.Background(), []uuid.UUID{userID}).Return(map[uuid.UUID][]string{userID: {"Customer"}}, nil).Once()

	resp, err := svc.BlockUser(context.Background(), staffID, userID)
	require.NoError(t, err)
	assert.False(t, resp.Enabled)
}

func TestStaffUsersApp_UpdateUser_DuplicateEmail(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo)

	userID := uuid.New()
	email := "dup@example.com"
	current := &models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(current, nil).Once()
	repo.On("SearchCustomerByEmailAndNotIDEquals", context.Background(), userID, email).
		Return(&models.Account{ID: uuid.New()}, nil).Once()

	_, err := svc.UpdateUser(context.Background(), uuid.New(), userID, dto.StaffUpdateUserRequest{Email: &email})
	assert.ErrorIs(t, err, errcustom.ErrEmailAlreadyExist)
}
