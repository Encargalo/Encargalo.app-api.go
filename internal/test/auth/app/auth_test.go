package app_test

import (
	"context"
	"testing"

	authapp "Encargalo.app-api.go/internal/auth/app"
	mocksauthports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	pass := plain
	if err := bycript.NewHashPassword().HashPassword(&pass); err != nil {
		t.Fatal(err)
	}
	return pass
}

func TestSignInCustomer_CancelsPendingDeletion(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	customerID := uuid.New()
	phone := "+573001112233"
	password := "claveSegura123"
	account := &models.Account{ID: customerID, Password: hashedPassword(t, password), Enabled: true}

	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).Return(account, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, customerID).Return(nil)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInCustomer(context.Background(), phone, password)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, sessionID)
	svc.AssertCalled(t, "CancelPendingDeletionIfAny", mock.Anything, customerID)
}

func TestSignInCustomer_CancelPendingDeletionFails_LoginStillSucceeds(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	customerID := uuid.New()
	phone := "+573001112233"
	password := "claveSegura123"
	account := &models.Account{ID: customerID, Password: hashedPassword(t, password), Enabled: true}

	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).Return(account, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, customerID).Return(errcustom.ErrUnexpectedError)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInCustomer(context.Background(), phone, password)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, sessionID)
}

// TestSignInCustomer_AccountAlreadyFinalized_NotFound documenta la garantía
// gratuita del soft delete: una cuenta con deleted_at seteado (eliminación ya
// finalizada) es invisible para SearchCustomerBy, así que el login falla como
// cualquier credencial inválida — no requiere código nuevo en appAuth.
func TestSignInCustomer_AccountAlreadyFinalized_NotFound(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	phone := "+573001112233"

	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).
		Return(nil, errcustom.ErrNotFound)

	sessionID, err := authSvc.SignInCustomer(context.Background(), phone, "claveSegura123")

	assert.ErrorIs(t, err, errcustom.ErrIncorrectAccessData)
	assert.Equal(t, uuid.Nil, sessionID)
}

func TestSignInCustomer_DisabledAccount(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	customerID := uuid.New()
	phone := "+573001112233"
	password := "claveSegura123"
	account := &models.Account{ID: customerID, Password: hashedPassword(t, password), Enabled: false}

	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).Return(account, nil)

	sessionID, err := authSvc.SignInCustomer(context.Background(), phone, password)

	assert.ErrorIs(t, err, errcustom.ErrAccountDisabled)
	assert.Equal(t, uuid.Nil, sessionID)
}
