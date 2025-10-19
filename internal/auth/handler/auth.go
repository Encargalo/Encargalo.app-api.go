package handler

import (
	"errors"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/auth/handler/request"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/labstack/echo/v4"
)

type Auth interface {
	SignInCustomer(e echo.Context) error
}

type auth struct {
	svc ports.AuthApp
	jwt jwt.Sessions
}

func NewAuthHandler(svc ports.AuthApp, jwt jwt.Sessions) Auth {
	return &auth{svc, jwt}
}

func (a *auth) SignInCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	var signIn request.SignInRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInCustomer(ctx, signIn.Phone, signIn.Password)
	if err != nil {
		if errors.Is(err, errcustom.ErrIncorrectAccessData) {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}

		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())

	}

	jwtSession, err := a.jwt.CreateSession(sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	cookie := &http.Cookie{
		Name:     "encargalo_session",
		Value:    jwtSession,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour)}

	e.SetCookie(cookie)

	return e.JSON(http.StatusCreated, "session created")
}
