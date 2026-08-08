package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	mocksauthports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mockscookie "Encargalo.app-api.go/internal/mocks/pkg/cookie"
	mocksjwt "Encargalo.app-api.go/internal/mocks/shared/jwt"
	mocksusersports "Encargalo.app-api.go/internal/mocks/users/domain/ports"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/models"
	usershandler "Encargalo.app-api.go/internal/users/handler"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func buildBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func httpCode(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func newUsersHandler(t *testing.T) (usershandler.UsersHandler, *mocksusersports.UsersApp, *mocksauthports.AuthApp, *mocksjwt.Sessions, *mockscookie.CookieHandler) {
	t.Helper()
	svc := mocksusersports.NewUsersApp(t)
	auth := mocksauthports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)
	h := usershandler.NewUsersHandler(svc, auth, jwtMock, cookieMock)
	return h, svc, auth, jwtMock, cookieMock
}

func sessionCtx(t *testing.T, tc buildhttptest.ControllerCase, userID uuid.UUID) (context.Context, buildhttptest.ControllerCase) {
	t.Helper()
	session := authmodels.ActiveSession{ID: uuid.New(), UserID: userID}
	ctx := context.WithValue(tc.Req.Context(), authmodels.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)
	return ctx, tc
}

// --- RegisterCustomer ---

func TestRegisterCustomer_Success(t *testing.T) {
	h, svc, auth, jwtMock, cookieMock := newUsersHandler(t)

	sessionID := uuid.New()
	body := buildBody(t, map[string]string{
		"name": "Carlos", "sur_name": "Ramírez",
		"phone": "+573001112233", "password": "claveSegura123",
		"birthday_date": "1990-05-20",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPost, body)

	svc.On("RegisterCustomer", tc.Req.Context(), dto.RegisterCustomer{
		Name: "Carlos", SurName: "Ramírez", Phone: "+573001112233",
		Password: "claveSegura123", BirthdayDate: "1990-05-20",
	}).Return(nil)
	auth.On("SignInCustomer", tc.Req.Context(), "+573001112233", "claveSegura123").Return(sessionID, nil)
	jwtMock.On("CreateSession", sessionID).Return("jwt-token", nil)
	cookieMock.On("CreateCookieSession", "jwt-token").Return(&http.Cookie{Name: "encargalo-session", Value: "jwt-token"})

	err := h.RegisterCustomer(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestRegisterCustomer_InvalidBody(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	body := bytes.NewBufferString("{invalid-json}")
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPost, body)

	err := h.RegisterCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterCustomer_ValidationFail(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	// name muy corto, sin phone e164
	body := buildBody(t, map[string]string{
		"name": "A", "sur_name": "Ramírez",
		"phone": "123", "password": "claveSegura123",
		"birthday_date": "1990-05-20",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPost, body)

	err := h.RegisterCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestRegisterCustomer_PhoneAlreadyExists(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	body := buildBody(t, map[string]string{
		"name": "Carlos", "sur_name": "Ramírez",
		"phone": "+573001112233", "password": "claveSegura123",
		"birthday_date": "1990-05-20",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPost, body)

	svc.On("RegisterCustomer", tc.Req.Context(), dto.RegisterCustomer{
		Name: "Carlos", SurName: "Ramírez", Phone: "+573001112233",
		Password: "claveSegura123", BirthdayDate: "1990-05-20",
	}).Return(errcustom.ErrPhoneAlreadyExist)

	err := h.RegisterCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusConflict, httpCode(err))
}

func TestRegisterCustomer_UnexpectedError(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	body := buildBody(t, map[string]string{
		"name": "Carlos", "sur_name": "Ramírez",
		"phone": "+573001112233", "password": "claveSegura123",
		"birthday_date": "1990-05-20",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPost, body)

	svc.On("RegisterCustomer", tc.Req.Context(), dto.RegisterCustomer{
		Name: "Carlos", SurName: "Ramírez", Phone: "+573001112233",
		Password: "claveSegura123", BirthdayDate: "1990-05-20",
	}).Return(errcustom.ErrUnexpectedError)

	err := h.RegisterCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- SearchCustomer ---

func TestSearchCustomer_Success(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users", http.MethodGet, nil)
	ctx, tc := sessionCtx(t, tc, userID)

	phone := "+573001112233"
	account := &models.Account{ID: userID, Name: "Carlos", Phone: &phone}
	svc.On("SearchCustomerProfileByID", ctx, userID).Return(account, nil)

	err := h.SearchCustomer(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestSearchCustomer_NoSession(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	tc := buildhttptest.SetupControllerCase("/users", http.MethodGet, nil)

	err := h.SearchCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestSearchCustomer_NotFound(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users", http.MethodGet, nil)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("SearchCustomerProfileByID", ctx, userID).Return(nil, errcustom.ErrNotFound)

	err := h.SearchCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, httpCode(err))
}

func TestSearchCustomer_UnexpectedError(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users", http.MethodGet, nil)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("SearchCustomerProfileByID", ctx, userID).Return(nil, errcustom.ErrUnexpectedError)

	err := h.SearchCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- UpdateCustomer ---

func TestUpdateCustomer_Success(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]string{
		"name": "Carlos", "sur_name": "Ramírez", "phone": "+573001112233",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPut, body)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("UpdateCustomer", ctx, userID, dto.UpdateCustomer{
		Name: new("Carlos"), SurName: new("Ramírez"), Phone: new("+573001112233"),
	}).Return(nil)

	err := h.UpdateCustomer(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateCustomer_InvalidBody(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	body := bytes.NewBufferString("{invalid}")
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPut, body)

	err := h.UpdateCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateCustomer_ValidationFail(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	body := buildBody(t, map[string]string{"name": "A", "sur_name": "R", "phone": "123"})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPut, body)

	err := h.UpdateCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdateCustomer_NoSession(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	body := buildBody(t, map[string]string{
		"name": "Carlos", "sur_name": "Ramírez", "phone": "+573001112233",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPut, body)

	err := h.UpdateCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestUpdateCustomer_PartialUpdate(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]string{"phone": "+573001112233"})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPut, body)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("UpdateCustomer", ctx, userID, dto.UpdateCustomer{
		Phone: new("+573001112233"),
	}).Return(nil)

	err := h.UpdateCustomer(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdateCustomer_NotFound(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]string{
		"name": "Carlos", "sur_name": "Ramírez", "phone": "+573001112233",
	})
	tc := buildhttptest.SetupControllerCase("/users", http.MethodPut, body)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("UpdateCustomer", ctx, userID, dto.UpdateCustomer{
		Name: new("Carlos"), SurName: new("Ramírez"), Phone: new("+573001112233"),
	}).Return(errcustom.ErrNotFound)

	err := h.UpdateCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, httpCode(err))
}

// --- UpdatePassword ---

func TestUpdatePassword_Success(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]string{"password": "nuevaClave123"})
	tc := buildhttptest.SetupControllerCase("/users/change-password", http.MethodPut, body)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("UpdatePassword", ctx, userID, dto.UpdatePassword{Password: "nuevaClave123"}).Return(nil)

	err := h.UpdatePassword(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestUpdatePassword_InvalidBody(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	body := bytes.NewBufferString("{invalid}")
	tc := buildhttptest.SetupControllerCase("/users/change-password", http.MethodPut, body)

	err := h.UpdatePassword(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdatePassword_ValidationFail(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	// password muy corta (min=8)
	body := buildBody(t, map[string]string{"password": "123"})
	tc := buildhttptest.SetupControllerCase("/users/change-password", http.MethodPut, body)

	err := h.UpdatePassword(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestUpdatePassword_NoSession(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	body := buildBody(t, map[string]string{"password": "nuevaClave123"})
	tc := buildhttptest.SetupControllerCase("/users/change-password", http.MethodPut, body)

	err := h.UpdatePassword(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestUpdatePassword_NotFound(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]string{"password": "nuevaClave123"})
	tc := buildhttptest.SetupControllerCase("/users/change-password", http.MethodPut, body)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("UpdatePassword", ctx, userID, dto.UpdatePassword{Password: "nuevaClave123"}).Return(errcustom.ErrNotFound)

	err := h.UpdatePassword(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusNotFound, httpCode(err))
}

func TestUpdatePassword_UnexpectedError(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	body := buildBody(t, map[string]string{"password": "nuevaClave123"})
	tc := buildhttptest.SetupControllerCase("/users/change-password", http.MethodPut, body)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("UpdatePassword", ctx, userID, dto.UpdatePassword{Password: "nuevaClave123"}).Return(errcustom.ErrUnexpectedError)

	err := h.UpdatePassword(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- DeleteAccount ---

func TestDeleteAccount_Success(t *testing.T) {
	h, svc, auth, _, cookieMock := newUsersHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users", http.MethodDelete, nil)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("RequestAccountDeletion", ctx, userID).Return(nil)
	auth.On("DeleteAllSessionsForUser", ctx, userID).Return(nil)
	cookieMock.On("DeleteCookieSession").Return(&http.Cookie{Name: "encargalo-session", Value: ""})

	err := h.DeleteAccount(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}

func TestDeleteAccount_NoSession(t *testing.T) {
	h, _, _, _, _ := newUsersHandler(t)

	tc := buildhttptest.SetupControllerCase("/users", http.MethodDelete, nil)

	err := h.DeleteAccount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, errcustom.ErrUnexpectedError, err)
}

func TestDeleteAccount_RequestFails(t *testing.T) {
	h, svc, _, _, _ := newUsersHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users", http.MethodDelete, nil)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("RequestAccountDeletion", ctx, userID).Return(errcustom.ErrUnexpectedError)

	err := h.DeleteAccount(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// DeleteAllSessionsForUser falla en modo best-effort: no debe romper la
// respuesta de éxito, la solicitud de eliminación ya quedó registrada.
func TestDeleteAccount_SessionInvalidationFailsButRequestSucceeds(t *testing.T) {
	h, svc, auth, _, cookieMock := newUsersHandler(t)

	userID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/users", http.MethodDelete, nil)
	ctx, tc := sessionCtx(t, tc, userID)

	svc.On("RequestAccountDeletion", ctx, userID).Return(nil)
	auth.On("DeleteAllSessionsForUser", ctx, userID).Return(errcustom.ErrUnexpectedError)
	cookieMock.On("DeleteCookieSession").Return(&http.Cookie{Name: "encargalo-session", Value: ""})

	err := h.DeleteAccount(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)
}
