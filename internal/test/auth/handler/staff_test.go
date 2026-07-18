package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	authhandler "Encargalo.app-api.go/internal/auth/handler"
	mocksports "Encargalo.app-api.go/internal/mocks/auth/domain/ports"
	mockscookie "Encargalo.app-api.go/internal/mocks/pkg/cookie"
	mocksjwt "Encargalo.app-api.go/internal/mocks/shared/jwt"
	"Encargalo.app-api.go/internal/shared/buildhttptest"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func buildBodyStaff(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewBuffer(b)
}

func httpCodeStaff(err error) int {
	if he, ok := err.(*echo.HTTPError); ok {
		return he.Code
	}
	return 0
}

func TestSignInStaff_Success(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	sessionID := uuid.New()
	body := buildBodyStaff(t, map[string]string{"phone_number": "+573001112233", "password": "claveSegura123"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/staff", http.MethodPost, body)

	svc.On("SignInStaff", tc.Req.Context(), "+573001112233", "claveSegura123").Return(sessionID, nil)
	jwtMock.On("CreateSession", sessionID).Return("jwt-token", nil)
	cookieMock.On("CreateCookieSession", "jwt-token").Return(&http.Cookie{Name: "encargalo-session", Value: "jwt-token"})

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInStaff(tc.Ctx)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tc.Res.Code)
}

func TestSignInStaff_IncorrectCredentials(t *testing.T) {
	svc := mocksports.NewAuthApp(t)
	jwtMock := mocksjwt.NewSessions(t)
	cookieMock := mockscookie.NewCookieHandler(t)

	body := buildBodyStaff(t, map[string]string{"phone_number": "+573001112233", "password": "wrong"})
	tc := buildhttptest.SetupControllerCase("/auth/sign-in/staff", http.MethodPost, body)

	svc.On("SignInStaff", tc.Req.Context(), "+573001112233", "wrong").Return(uuid.UUID{}, errcustom.ErrIncorrectAccessData)

	h := authhandler.NewAuthHandler(svc, jwtMock, cookieMock)
	err := h.SignInStaff(tc.Ctx)

	assert.Error(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, httpCodeStaff(err))
}
