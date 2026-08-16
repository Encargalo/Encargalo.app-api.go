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

func TestStaffUsersApp_GrantStaff(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo)

	staffID := uuid.New()
	userID := uuid.New()
	acc := &models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(acc, nil).Once()
	repo.On("HasStaffRole", context.Background(), userID).Return(false, nil).Once()
	repo.On("EnsureStaffRole", context.Background(), userID, staffID).Return(nil).Once()
	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(acc, nil).Once()
	repo.On("GetRolesForUserIDs", context.Background(), []uuid.UUID{userID}).Return(map[uuid.UUID][]string{userID: {"Customer", "Staff"}}, nil).Once()

	resp, err := svc.GrantStaff(context.Background(), staffID, userID)
	require.NoError(t, err)
	assert.Contains(t, resp.Roles, "Staff")
}

func TestStaffUsersApp_RevokeStaff_CannotRevokeSelf(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo)

	staffID := uuid.New()
	_, revoked, err := svc.RevokeStaff(context.Background(), staffID, staffID)
	assert.ErrorIs(t, err, errcustom.ErrCannotRevokeOwnStaff)
	assert.False(t, revoked)
}

func TestStaffUsersApp_RevokeStaff_LastStaffMember(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo)

	staffID := uuid.New()
	userID := uuid.New()
	acc := &models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(acc, nil).Once()
	repo.On("HasStaffRole", context.Background(), userID).Return(true, nil).Once()
	repo.On("CountEnabledStaff", context.Background()).Return(1, nil).Once()

	_, revoked, err := svc.RevokeStaff(context.Background(), staffID, userID)
	assert.ErrorIs(t, err, errcustom.ErrLastStaffMember)
	assert.False(t, revoked)
}
