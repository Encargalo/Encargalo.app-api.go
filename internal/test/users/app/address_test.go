package app_test

import (
	"context"
	"testing"

	mockports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/app"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRegisterAddress_Success(t *testing.T) {
	repo := mockports.NewUsersAddressRepo(t)
	svc := app.NewUsersAddressApp(repo)

	customerID := uuid.New()
	addr := dto.Address{
		Type: "Casa", Label: "Casa", Address: "Calle 1", Latitude: 10.1, Longitude: -66.1,
	}

	repo.On("RegisterAddress", mock.Anything, mock.AnythingOfType("models.Address")).Return(nil)

	err := svc.RegisterAddress(context.Background(), customerID, addr)

	assert.NoError(t, err)
}

func TestSearchAllAddress_DelegatesToRepo(t *testing.T) {
	repo := mockports.NewUsersAddressRepo(t)
	svc := app.NewUsersAddressApp(repo)

	customerID := uuid.New()
	expected := models.Addresses{{ID: uuid.New(), CustomerID: customerID, Label: "Casa"}}
	repo.On("SearchAllAddress", mock.Anything, customerID).Return(expected, nil)

	got, err := svc.SearchAllAddress(context.Background(), customerID)

	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestDeleteAddress_RepoError(t *testing.T) {
	repo := mockports.NewUsersAddressRepo(t)
	svc := app.NewUsersAddressApp(repo)

	customerID := uuid.New()
	repo.On("DeleteAddress", mock.Anything, customerID, "addr-1").Return(errcustom.ErrNotFound)

	err := svc.DeleteAddress(context.Background(), customerID, "addr-1")

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}
