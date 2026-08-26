package app_test

import (
	"context"
	"testing"

	authapp "Encargalo.app-api.go/internal/auth/app"
	"Encargalo.app-api.go/internal/auth/domain/models"
	mocksauthports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mocksriders "Encargalo.app-api.go/internal/mocks/riders/domain/ports"
	mocksgoogleauth "Encargalo.app-api.go/internal/mocks/shared/adapters/googleauth"
	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/pkg/bycript"
	"Encargalo.app-api.go/internal/shared/adapters/googleauth"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	usersmodels "Encargalo.app-api.go/internal/users/domain/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSignInEmployee_Success(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	employeeID, shopID := uuid.New(), uuid.New()
	phone, password := "0414", "claveSegura123"
	account := &usersmodels.Account{ID: employeeID, Password: hashedPassword(t, password), Enabled: true, Role: "OrderManager", ShopID: shopID}

	svc.On("SearchAccountByPhone", mock.Anything, phone, []string{"Order Manager", "Owner"}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: employeeID}).Return(account, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, employeeID).Return(nil)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInEmployee(context.Background(), phone, password)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, sessionID)
}

func TestSignInEmployee_NotFound(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	svc.On("SearchAccountByPhone", mock.Anything, "0414", []string{"Order Manager", "Owner"}).
		Return(nil, errcustom.ErrNotFound)

	_, err := authSvc.SignInEmployee(context.Background(), "0414", "x")

	require.ErrorIs(t, err, errcustom.ErrIncorrectAccessData)
}

func TestSignInRider_Success(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	riderProfiles := mocksriders.NewRiderProfileApp(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, riderProfiles)

	riderID := uuid.New()
	phone, password := "0414", "claveSegura123"
	account := &usersmodels.Account{ID: riderID, Password: hashedPassword(t, password), Enabled: true}

	svc.On("SearchAccountByPhone", mock.Anything, phone, []string{string(models.Rider)}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: riderID}).Return(account, nil)
	riderProfiles.On("IsActiveRider", mock.Anything, riderID).Return(true, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, riderID).Return(nil)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInRider(context.Background(), phone, password)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, sessionID)
}

func TestSignInRider_NotActiveRider(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	riderProfiles := mocksriders.NewRiderProfileApp(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, riderProfiles)

	riderID := uuid.New()
	phone, password := "0414", "claveSegura123"
	account := &usersmodels.Account{ID: riderID, Password: hashedPassword(t, password), Enabled: true}

	svc.On("SearchAccountByPhone", mock.Anything, phone, []string{string(models.Rider)}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: riderID}).Return(account, nil)
	riderProfiles.On("IsActiveRider", mock.Anything, riderID).Return(false, nil)

	_, err := authSvc.SignInRider(context.Background(), phone, password)

	require.ErrorIs(t, err, errcustom.ErrRiderNotActive)
	repo.AssertNotCalled(t, "SaveSession", mock.Anything, mock.Anything)
}

func TestSignInStaff_Success(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	staffID := uuid.New()
	phone, password := "0414", "claveSegura123"
	account := &usersmodels.Account{ID: staffID, Password: hashedPassword(t, password), Enabled: true}

	svc.On("SearchAccountByPhone", mock.Anything, phone, []string{string(models.Staff)}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: staffID}).Return(account, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, staffID).Return(nil)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInStaff(context.Background(), phone, password)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, sessionID)
}

func TestSignInStaff_WrongPassword(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, nil, nil)

	account := &usersmodels.Account{ID: uuid.New(), Password: hashedPassword(t, "correcta"), Enabled: true}
	svc.On("SearchAccountByPhone", mock.Anything, "0414", []string{string(models.Staff)}).Return(account, nil)

	_, err := authSvc.SignInStaff(context.Background(), "0414", "incorrecta")

	require.ErrorIs(t, err, errcustom.ErrIncorrectAccessData)
}

func TestSignInGoogle_ExistingCustomer(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	verifier := mocksgoogleauth.NewVerifier(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, verifier, nil)

	customerID := uuid.New()
	account := &usersmodels.Account{ID: customerID, Enabled: true}

	verifier.On("Verify", mock.Anything, "id-token").Return(&googleauth.UserInfo{Email: "luigi@example.com"}, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Email: "luigi@example.com"}).Return(account, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: customerID}).Return(account, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, customerID).Return(nil)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInGoogle(context.Background(), "id-token")

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, sessionID)
}

func TestSignInGoogle_RegistersNewCustomer(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	repo := mocksauthports.NewAuthRepo(t)
	verifier := mocksgoogleauth.NewVerifier(t)
	authSvc := authapp.NewAuthApp(svc, bycript.NewHashPassword(), repo, verifier, nil)

	newID := uuid.New()
	registered := &usersmodels.Account{ID: newID, Enabled: true}

	verifier.On("Verify", mock.Anything, "id-token").Return(&googleauth.UserInfo{Email: "nuevo@example.com", Name: "Luigi"}, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{Email: "nuevo@example.com"}).Return(nil, errcustom.ErrNotFound)
	svc.On("RegisterCustomerFromGoogle", mock.Anything, dto.RegisterCustomerFromGoogle{Name: "Luigi", Email: "nuevo@example.com"}).Return(registered, nil)
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: newID}).Return(registered, nil)
	svc.On("CancelPendingDeletionIfAny", mock.Anything, newID).Return(nil)
	repo.On("SaveSession", mock.Anything, mock.AnythingOfType("*models.ActiveSession")).Return(nil)

	sessionID, err := authSvc.SignInGoogle(context.Background(), "id-token")

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, sessionID)
}

func TestSignInGoogle_InvalidToken(t *testing.T) {
	verifier := mocksgoogleauth.NewVerifier(t)
	authSvc := authapp.NewAuthApp(nil, nil, nil, verifier, nil)

	verifier.On("Verify", mock.Anything, "bad-token").Return(nil, errcustom.ErrUnautorized)

	_, err := authSvc.SignInGoogle(context.Background(), "bad-token")

	require.ErrorIs(t, err, errcustom.ErrUnautorized)
}

func TestSearchSessions_DelegatesToRepo(t *testing.T) {
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(nil, nil, repo, nil, nil)

	sessionID := uuid.New()
	session := &models.ActiveSession{ID: sessionID}
	repo.On("SearchSession", mock.Anything, sessionID).Return(session, nil)

	got, err := authSvc.SearchSessions(context.Background(), sessionID)

	require.NoError(t, err)
	require.Equal(t, session, got)
}

func TestDeleteSession_Success(t *testing.T) {
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(nil, nil, repo, nil, nil)

	sessionID := uuid.New()
	repo.On("SearchSession", mock.Anything, sessionID).Return(&models.ActiveSession{ID: sessionID}, nil)
	repo.On("DeleteSession", mock.Anything, sessionID).Return(nil)

	err := authSvc.DeleteSession(context.Background(), sessionID)

	require.NoError(t, err)
}

// DeleteSession sobre una sesión inexistente es un no-op silencioso (ya
// expiró/fue borrada por otro request), no un error.
func TestDeleteSession_AlreadyGoneIsNoop(t *testing.T) {
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(nil, nil, repo, nil, nil)

	sessionID := uuid.New()
	repo.On("SearchSession", mock.Anything, sessionID).Return(nil, nil)

	err := authSvc.DeleteSession(context.Background(), sessionID)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "DeleteSession", mock.Anything, mock.Anything)
}

func TestDeleteSession_SearchError(t *testing.T) {
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(nil, nil, repo, nil, nil)

	sessionID := uuid.New()
	repo.On("SearchSession", mock.Anything, sessionID).Return(nil, errcustom.ErrUnexpectedError)

	err := authSvc.DeleteSession(context.Background(), sessionID)

	require.ErrorIs(t, err, errcustom.ErrUnexpectedError)
}

func TestDeleteAllSessionsForUser_DelegatesToRepo(t *testing.T) {
	repo := mocksauthports.NewAuthRepo(t)
	authSvc := authapp.NewAuthApp(nil, nil, repo, nil, nil)

	userID := uuid.New()
	repo.On("DeleteAllSessionsForUser", mock.Anything, userID).Return(nil)

	err := authSvc.DeleteAllSessionsForUser(context.Background(), userID)

	require.NoError(t, err)
}

func TestEnsureAccountEnabled_Enabled(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	authSvc := authapp.NewAuthApp(svc, nil, nil, nil, nil)

	userID := uuid.New()
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: userID}).
		Return(&usersmodels.Account{ID: userID, Enabled: true}, nil)

	err := authSvc.EnsureAccountEnabled(context.Background(), userID)

	require.NoError(t, err)
}

func TestEnsureAccountEnabled_Disabled(t *testing.T) {
	svc := mocksusersports.NewUsersApp(t)
	authSvc := authapp.NewAuthApp(svc, nil, nil, nil, nil)

	userID := uuid.New()
	svc.On("SearchCustomerBy", mock.Anything, dto.SearchCustomerBy{ID: userID}).
		Return(&usersmodels.Account{ID: userID, Enabled: false}, nil)

	err := authSvc.EnsureAccountEnabled(context.Background(), userID)

	require.ErrorIs(t, err, errcustom.ErrAccountDisabled)
}
