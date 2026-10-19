package app_test

import (
	"context"
	"testing"

	authapp "Encargalo.app-api.go/internal/auth/app"
	"Encargalo.app-api.go/internal/auth/domain/models"
	mocksauthports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	usersmodels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestIdentifyRiderPhone_UnknownPhoneRegisters(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	phone := "+573001112233"
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(nil, errcustom.ErrNotFound)

	next, err := authSvc.IdentifyRiderPhone(context.Background(), phone)

	assert.NoError(t, err)
	assert.Equal(t, models.RiderPhoneNextRegister, next)
}

func TestIdentifyRiderPhone_CustomerWithoutRiderRoleOnboards(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	phone := "+573001112233"
	account := &usersmodels.Account{ID: uuid.New()}
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(account, nil)
	svc.On("GetUserRoles", mock.Anything, account.ID).Return([]string{string(models.Customer)}, nil)

	next, err := authSvc.IdentifyRiderPhone(context.Background(), phone)

	assert.NoError(t, err)
	assert.Equal(t, models.RiderPhoneNextOnboarding, next)
}

func TestIdentifyRiderPhone_RiderRoleSignsIn(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	phone := "+573001112233"
	account := &usersmodels.Account{ID: uuid.New()}
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(account, nil)
	svc.On("GetUserRoles", mock.Anything, account.ID).Return([]string{string(models.Customer), string(models.Rider)}, nil)

	next, err := authSvc.IdentifyRiderPhone(context.Background(), phone)

	assert.NoError(t, err)
	assert.Equal(t, models.RiderPhoneNextSignIn, next)
}

func TestIdentifyRiderPhone_SearchError(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	phone := "+573001112233"
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(nil, errcustom.ErrUnexpectedError)

	next, err := authSvc.IdentifyRiderPhone(context.Background(), phone)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	assert.Empty(t, next)
}

func TestIdentifyRiderPhone_RolesError(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	phone := "+573001112233"
	account := &usersmodels.Account{ID: uuid.New()}
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Phone: phone}).Return(account, nil)
	svc.On("GetUserRoles", mock.Anything, account.ID).Return(nil, errcustom.ErrUnexpectedError)

	next, err := authSvc.IdentifyRiderPhone(context.Background(), phone)

	assert.ErrorIs(t, err, errcustom.ErrUnexpectedError)
	assert.Empty(t, next)
}
