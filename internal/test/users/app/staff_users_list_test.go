package app_test

import (
	"context"
	"testing"

	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cursor"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaffUsersApp_ListUsers_Success(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	userID := uuid.New()
	acc := models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("ListManagedCustomers", context.Background(), "", dto.StaffUserStatusAll, 21, (*cursor.Key)(nil)).
		Return([]models.Account{acc}, nil)
	repo.On("GetRolesForUserIDs", context.Background(), []uuid.UUID{userID}).
		Return(map[uuid.UUID][]string{userID: {"Customer"}}, nil)

	resp, err := svc.ListUsers(context.Background(), dto.ListStaffUsersQuery{})

	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.False(t, resp.HasMore)
}

func TestStaffUsersApp_ListUsers_InvalidStatus(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	_, err := svc.ListUsers(context.Background(), dto.ListStaffUsersQuery{Status: "banned"})

	assert.ErrorIs(t, err, errcustom.ErrInvalidFilter)
}

func TestStaffUsersApp_ListUsers_QueryTooShort(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	_, err := svc.ListUsers(context.Background(), dto.ListStaffUsersQuery{Query: "a"})

	assert.ErrorIs(t, err, errcustom.ErrInvalidSearchQuery)
}

func TestStaffUsersApp_ListUsers_InvalidCursor(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	_, err := svc.ListUsers(context.Background(), dto.ListStaffUsersQuery{Cursor: "not-a-valid-cursor"})

	assert.ErrorIs(t, err, errcustom.ErrInvalidCursor)
}

func TestStaffUsersApp_FindCandidate_Success(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	userID := uuid.New()
	acc := &models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("FindManagedCustomerByPhone", context.Background(), "04141234567").Return(acc, nil)
	repo.On("GetRolesForUserIDs", context.Background(), []uuid.UUID{userID}).
		Return(map[uuid.UUID][]string{userID: {"Customer"}}, nil)

	resp, err := svc.FindCandidate(context.Background(), "04141234567")

	require.NoError(t, err)
	assert.Equal(t, userID, resp.UserID)
}

func TestStaffUsersApp_FindCandidate_EmptyPhone(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	_, err := svc.FindCandidate(context.Background(), "   ")

	assert.ErrorIs(t, err, errcustom.ErrInvalidSearchQuery)
}

func TestStaffUsersApp_FindCandidate_NotFound(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	repo.On("FindManagedCustomerByPhone", context.Background(), "04141234567").Return(nil, errcustom.ErrNotFound)

	_, err := svc.FindCandidate(context.Background(), "04141234567")

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestStaffUsersApp_UnblockUser_EnablesAccount(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	stream := mocksusersports.NewStreamPublisher(t)
	svc := app.NewStaffUsersApp(repo, stream)

	staffID := uuid.New()
	userID := uuid.New()

	repo.On("GetManagedCustomerByID", context.Background(), userID).
		Return(&models.Account{ID: userID, Name: "Ana", Enabled: false}, nil).Once()
	repo.On("SetAccountEnabled", context.Background(), userID, true).Return(nil).Once()
	repo.On("GetManagedCustomerByID", context.Background(), userID).
		Return(&models.Account{ID: userID, Name: "Ana", Enabled: true}, nil).Once()
	repo.On("GetRolesForUserIDs", context.Background(), []uuid.UUID{userID}).
		Return(map[uuid.UUID][]string{userID: {"Customer"}}, nil).Once()
	// El ETL resincroniza DimUser por evento, no por el batch horario.
	stream.On("PublisherEvent", context.Background(), models.DataEventUserUpdated{
		MessageType: "user_updated",
		UserID:      userID.String(),
	}).Return(nil)

	resp, err := svc.UnblockUser(context.Background(), staffID, userID)

	require.NoError(t, err)
	assert.True(t, resp.Enabled)
}

func TestStaffUsersApp_UnblockUser_AlreadyEnabledIsNoop(t *testing.T) {
	repo := mocksusersports.NewStaffUsersRepo(t)
	svc := app.NewStaffUsersApp(repo, nil)

	staffID := uuid.New()
	userID := uuid.New()
	acc := &models.Account{ID: userID, Name: "Ana", Enabled: true}

	repo.On("GetManagedCustomerByID", context.Background(), userID).Return(acc, nil).Once()
	repo.On("GetRolesForUserIDs", context.Background(), []uuid.UUID{userID}).
		Return(map[uuid.UUID][]string{userID: {"Customer"}}, nil).Once()

	resp, err := svc.UnblockUser(context.Background(), staffID, userID)

	require.NoError(t, err)
	assert.True(t, resp.Enabled)
	repo.AssertNotCalled(t, "SetAccountEnabled", context.Background(), userID, true)
}
