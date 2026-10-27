package app_test

import (
	"context"
	"testing"

	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRegisterCustomerFromGoogle_Success(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	stream := mockports.NewStreamPublisher(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword(), stream)

	req := dto.RegisterCustomerFromGoogle{Name: "Carlos", SurName: "Ramírez", Email: "carlos@example.com"}
	var registeredID uuid.UUID
	repo.On("RegisterCustomerFromGoogle", mock.Anything, mock.AnythingOfType("*models.Account")).
		Run(func(args mock.Arguments) {
			acc := args.Get(1).(*models.Account)
			acc.ID = uuid.New()
			registeredID = acc.ID
		}).Return(nil)
	stream.On("PublisherEvent", mock.Anything, mock.MatchedBy(func(event models.DataEventUserUpdated) bool {
		return event.MessageType == "user_updated" && event.UserID == registeredID.String()
	})).Return(nil)

	acc, err := svc.RegisterCustomerFromGoogle(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, acc)
}

func TestRegisterCustomerFromGoogle_RepoError(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword(), nil)

	req := dto.RegisterCustomerFromGoogle{Name: "Carlos", Email: "carlos@example.com"}
	repo.On("RegisterCustomerFromGoogle", mock.Anything, mock.AnythingOfType("*models.Account")).
		Return(errcustom.ErrUnexpectedError)

	_, err := svc.RegisterCustomerFromGoogle(context.Background(), req)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestSearchCustomerProfileByID_DelegatesToRepo(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword(), nil)

	userID := uuid.New()
	repo.On("SearchCustomerProfileByID", mock.Anything, userID).Return(&models.Account{ID: userID}, nil)

	acc, err := svc.SearchCustomerProfileByID(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, userID, acc.ID)
}

func TestGetUserRoles_DelegatesToRepo(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword(), nil)

	userID := uuid.New()
	repo.On("GetUserRoles", mock.Anything, userID).Return([]string{"Customer"}, nil)

	roles, err := svc.GetUserRoles(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, []string{"Customer"}, roles)
}

func TestSearchAccountByPhone_DelegatesToRepo(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword(), nil)

	roles := []string{"OrderManager"}
	repo.On("SearchAccountByPhone", mock.Anything, "0414", roles).Return(&models.Account{Phone: new("0414")}, nil)

	acc, err := svc.SearchAccountByPhone(context.Background(), "0414", roles)

	require.NoError(t, err)
	require.NotNil(t, acc.Phone)
	assert.Equal(t, "0414", *acc.Phone)
}
