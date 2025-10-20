package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/auth/handler/request"
	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Auth interface {
	SignInCustomer(e echo.Context) error
	Logout(e echo.Context) error
}

type auth struct {
	svc    ports.AuthApp
	jwt    jwt.Sessions
	cookie cookie.Cookie
}

func NewAuthHandler(svc ports.AuthApp, jwt jwt.Sessions, cookie cookie.Cookie) Auth {
	return &auth{svc, jwt, cookie}
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

	cookie := a.cookie.CreateCookieSession(jwtSession)

	e.SetCookie(cookie)

	return e.JSON(http.StatusCreated, "session created")
}

// DeleteSession godoc
// @Summary Cierra la sesión actual del cliente autenticado
// @Description Elimina la sesión activa identificada por session_id en la cookie de autenticación
// @Tags Sessions
// @Produce json
// @Success 200 {string} string "session deleted success"
// @Failure 404 {string} string "not found"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /customers/sessions [delete]
// @deprecatedrouter /api/sessions [delete]
func (s *auth) Logout(e echo.Context) error {

	ctx := e.Request().Context()

	sessionID, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("session_id"))))
	if err != nil {
		fmt.Println("Error al obtener el session_id")
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	if err := s.svc.DeleteSession(ctx, sessionID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	cookie := s.cookie.DeleteCookieSession()

	e.SetCookie(cookie)

	return e.JSON(http.StatusOK, "session deleted success")
}
