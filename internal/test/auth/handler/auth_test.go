package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	authhandler "Encargalo.app-api.go/internal/auth/handler"
	mocksports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mockscookie "Encargalo.app-api.go/internal/mocks/pkg/cookie"
	mocksjwt "Encargalo.app-api.go/internal/mocks/shared/jwt"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- SignInCustomer ---

func TestSignInCustomer_Success(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "claveSegura123"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/users", http.MethodPost, body)

	svc.On("SignInCustomer", tc.Req.Context(), "+573001112233", "claveSegura123").Return(sessionID, nil)
	jwtMock.On("CreateSession", sessionID).Return("jwt-token", nil)
	cookieMock.On("CreateCookieSession", "jwt-token").Return(&http.Cookie{Name: "encargalo-session", Value: "jwt-token"})

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInCustomer(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestSignInCustomer_InvalidBody(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := bytes.NewBufferString("{invalid-json}")
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/users", http.MethodPost, body)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSignInCustomer_ValidationFail(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	// phone_number inválido (no formato e164)
	body := buildBody(t, map[string]string{"phone_number": "123", "password": "pass"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/users", http.MethodPost, body)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSignInCustomer_IncorrectCredentials(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "wrong"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/users", http.MethodPost, body)

	svc.On("SignInCustomer", tc.Req.Context(), "+573001112233", "wrong").Return(uuid.UUID{}, errcustom.ErrIncorrectAccessData)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestSignInCustomer_UnexpectedError(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "pass"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/users", http.MethodPost, body)

	svc.On("SignInCustomer", tc.Req.Context(), "+573001112233", "pass").Return(uuid.UUID{}, errcustom.ErrUnexpectedError)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

func TestSignInCustomer_JWTError(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "pass"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/users", http.MethodPost, body)

	svc.On("SignInCustomer", tc.Req.Context(), "+573001112233", "pass").Return(sessionID, nil)
	jwtMock.On("CreateSession", sessionID).Return("", errcustom.ErrUnexpectedError)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInCustomer(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- SignInEmployee ---

func TestSignInEmployee_Success(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "claveSegura123"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/employees", http.MethodPost, body)

	svc.On("SignInEmployee", tc.Req.Context(), "+573001112233", "claveSegura123").Return(sessionID, nil)
	jwtMock.On("CreateSession", sessionID).Return("jwt-token", nil)
	cookieMock.On("CreateCookieSession", "jwt-token").Return(&http.Cookie{Name: "encargalo-session", Value: "jwt-token"})

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInEmployee(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestSignInEmployee_IncorrectCredentials(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "wrong"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/employees", http.MethodPost, body)

	svc.On("SignInEmployee", tc.Req.Context(), "+573001112233", "wrong").Return(uuid.UUID{}, errcustom.ErrIncorrectAccessData)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInEmployee(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCode(err))
}

func TestSignInEmployee_UnexpectedError(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBody(t, map[string]string{"phone_number": "+573001112233", "password": "pass"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/employees", http.MethodPost, body)

	svc.On("SignInEmployee", tc.Req.Context(), "+573001112233", "pass").Return(uuid.UUID{}, errcustom.ErrUnexpectedError)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInEmployee(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- SignInGoogle ---

func TestSignInGoogle_SuccessExistingUser(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	body := buildBody(t, map[string]string{"id_token": "valid-token"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/google", http.MethodPost, body)

	svc.On("SignInGoogle", tc.Req.Context(), "valid-token").Return(sessionID, nil)
	jwtMock.On("CreateSession", sessionID).Return("jwt-token", nil)
	cookieMock.On("CreateCookieSession", "jwt-token").Return(&http.Cookie{Name: "encargalo-session", Value: "jwt-token"})

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInGoogle(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestSignInGoogle_InvalidBody(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := bytes.NewBufferString("{invalid-json}")
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/google", http.MethodPost, body)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInGoogle(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSignInGoogle_ValidationFail(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	// id_token vacío
	body := buildBody(t, map[string]string{"id_token": ""})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/google", http.MethodPost, body)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInGoogle(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, httpCode(err))
}

func TestSignInGoogle_InvalidToken(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBody(t, map[string]string{"id_token": "expired-token"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/google", http.MethodPost, body)

	svc.On("SignInGoogle", tc.Req.Context(), "expired-token").Return(uuid.UUID{}, errcustom.ErrUnautorized)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInGoogle(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestSignInGoogle_UnexpectedError(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBody(t, map[string]string{"id_token": "valid-token"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/google", http.MethodPost, body)

	svc.On("SignInGoogle", tc.Req.Context(), "valid-token").Return(uuid.UUID{}, errcustom.ErrUnexpectedError)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInGoogle(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- Logout ---

func TestLogout_Success(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	session := models.ActiveSession{ID: sessionID}

	tc := buildhttptest.SetupControllerCase("/auth/logout", http.MethodDelete, nil)
	ctx := context.WithValue(tc.Req.Context(), models.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)

	svc.On("DeleteSession", ctx, sessionID).Return(nil)
	cookieMock.On("DeleteCookieSession").Return(&http.Cookie{Name: "encargalo-session", Value: "", MaxAge: -1})

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.Logout(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var body authhandler.LogoutResponse
	require.NoError(t, json.Unmarshal(tc.Res.Body.Bytes(), &body))
	assert.Equal(t, "session deleted success", body.Message)
}

func TestLogout_DeleteSessionError(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	session := models.ActiveSession{ID: sessionID}

	tc := buildhttptest.SetupControllerCase("/auth/logout", http.MethodDelete, nil)
	ctx := context.WithValue(tc.Req.Context(), models.SessionKey, session)
	tc.Req = tc.Req.WithContext(ctx)
	tc.Ctx.SetRequest(tc.Req)

	svc.On("DeleteSession", ctx, sessionID).Return(errcustom.ErrUnexpectedError)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.Logout(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusInternalServerError, httpCode(err))
}

// --- VerifySession ---

func TestVerifySession_Success(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	session := &models.ActiveSession{
		ID:        sessionID,
		UserID:    uuid.New(),
		Roles:     []models.Role{models.Customer},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodGet, nil)
	tc.Req.AddCookie(&http.Cookie{Name: "encargalo-session", Value: "valid-jwt"})
	tc.Ctx.SetRequest(tc.Req)

	jwtMock.On("ValidateToken", "valid-jwt").Return(jwt.MapClaims{"session_id": sessionID.String()}, nil)
	svc.On("SearchSessions", tc.Req.Context(), sessionID).Return(session, nil)
	svc.On("EnsureAccountEnabled", tc.Req.Context(), session.UserID).Return(nil)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.VerifySession(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tc.Res.Code)

	var got models.ActiveSession
	require.NoError(t, json.Unmarshal([]byte(tc.Res.Header().Get(sharedmw.SessionHeader)), &got))
	assert.Equal(t, sessionID, got.ID)
	assert.Equal(t, session.Roles, got.Roles)
}

func TestVerifySession_MissingCookie_Unauthorized(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodGet, nil)

	jwtMock.On("ValidateToken", "").Return(jwt.MapClaims{}, errcustom.ErrUnautorized)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.VerifySession(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestVerifySession_InvalidToken_Unauthorized(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodGet, nil)
	tc.Req.AddCookie(&http.Cookie{Name: "encargalo-session", Value: "invalid-jwt"})
	tc.Ctx.SetRequest(tc.Req)

	jwtMock.On("ValidateToken", "invalid-jwt").Return(jwt.MapClaims{}, errcustom.ErrUnautorized)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.VerifySession(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestVerifySession_SessionNotFound_Unauthorized(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodGet, nil)
	tc.Req.AddCookie(&http.Cookie{Name: "encargalo-session", Value: "valid-jwt"})
	tc.Ctx.SetRequest(tc.Req)

	jwtMock.On("ValidateToken", "valid-jwt").Return(jwt.MapClaims{"session_id": sessionID.String()}, nil)
	svc.On("SearchSessions", tc.Req.Context(), sessionID).Return(nil, errcustom.ErrNotFound)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.VerifySession(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}

func TestVerifySession_SessionExpired_Unauthorized(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	session := &models.ActiveSession{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	tc := buildhttptest.SetupControllerCase("/internal/verify-session", http.MethodGet, nil)
	tc.Req.AddCookie(&http.Cookie{Name: "encargalo-session", Value: "valid-jwt"})
	tc.Ctx.SetRequest(tc.Req)

	jwtMock.On("ValidateToken", "valid-jwt").Return(jwt.MapClaims{"session_id": sessionID.String()}, nil)
	svc.On("SearchSessions", tc.Req.Context(), sessionID).Return(session, nil)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.VerifySession(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpCode(err))
}
