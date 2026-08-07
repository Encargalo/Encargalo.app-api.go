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
)

func TestRegisterCustomer_Success(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	req := dto.RegisterCustomer{
		Name: "Carlos", SurName: "Ramírez", Phone: "+573001112233",
		Email: "carlos@example.com", Password: "claveSegura123",
	}

	repo.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: req.Phone}).
		Return(nil, errcustom.ErrNotFound)
	repo.On("RegisterCustomer", mock.Anything, mock.AnythingOfType("*models.Account")).
		Return(nil)

	err := svc.RegisterCustomer(context.Background(), req)

	assert.NoError(t, err)
}

func TestRegisterCustomer_PhoneAlreadyExists(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	req := dto.RegisterCustomer{
		Name: "Carlos", SurName: "Ramírez", Phone: "+573001112233",
		Email: "carlos@example.com", Password: "claveSegura123",
	}

	repo.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: req.Phone}).
		Return(&models.Account{ID: uuid.New()}, nil)

	err := svc.RegisterCustomer(context.Background(), req)

	assert.ErrorIs(t, err, errcustom.ErrPhoneAlreadyExist)
}

func TestUpdateCustomer_PhoneAlreadyExists(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()
	newPhone := "+573009998877"
	update := dto.UpdateCustomer{Phone: &newPhone}

	repo.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).
		Return(&models.Account{ID: customerID}, nil)
	repo.On("SearchCustomerByPhoneAndNotIDEquals", mock.Anything, customerID, newPhone).
		Return(&models.Account{ID: uuid.New()}, nil)

	err := svc.UpdateCustomer(context.Background(), customerID, update)

	assert.ErrorIs(t, err, errcustom.ErrPhoneAlreadyExist)
}

func TestUpdateCustomer_NotFound(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()

	repo.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).
		Return(nil, errcustom.ErrNotFound)

	err := svc.UpdateCustomer(context.Background(), customerID, dto.UpdateCustomer{})

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdatePassword_NotFound(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()

	repo.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).
		Return(nil, nil)

	err := svc.UpdatePassword(context.Background(), customerID, dto.UpdatePassword{Password: "claveSegura123"})

	assert.ErrorIs(t, err, errcustom.ErrNotFound)
}

func TestUpdatePassword_Success(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()

	repo.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).
		Return(&models.Account{ID: customerID}, nil)
	repo.On("UpdatePassword", mock.Anything, customerID, mock.AnythingOfType("*models.Account")).
		Return(nil)

	err := svc.UpdatePassword(context.Background(), customerID, dto.UpdatePassword{Password: "claveSegura123"})

	assert.NoError(t, err)
}

func TestRequestAccountDeletion_Success(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()

	repo.On("RequestAccountDeletion", mock.Anything, customerID).Return(nil)

	err := svc.RequestAccountDeletion(context.Background(), customerID)

	assert.NoError(t, err)
}

func TestRequestAccountDeletion_RepoError(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()

	repo.On("RequestAccountDeletion", mock.Anything, customerID).Return(errcustom.ErrUnexpectedError)

	err := svc.RequestAccountDeletion(context.Background(), customerID)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestCancelPendingDeletionIfAny_Success(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	customerID := uuid.New()

	repo.On("CancelAccountDeletion", mock.Anything, customerID).Return(nil)

	err := svc.CancelPendingDeletionIfAny(context.Background(), customerID)

	assert.NoError(t, err)
}

func TestFinalizePendingDeletions_NoEligibleAccounts(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	repo.On("FinalizePendingDeletions", mock.Anything, mock.AnythingOfType("time.Time")).
		Return(int64(0), nil)

	n, err := svc.FinalizePendingDeletions(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestFinalizePendingDeletions_RepoError(t *testing.T) {
	repo := mockports.NewUsersRepo(t)
	svc := app.NewUserApp(repo, bycript.NewHashPassword())

	repo.On("FinalizePendingDeletions", mock.Anything, mock.AnythingOfType("time.Time")).
		Return(int64(0), errcustom.ErrUnexpectedError)

	n, err := svc.FinalizePendingDeletions(context.Background())

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	assert.Equal(t, int64(0), n)
}
