package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/auth/domain/ports"
	"Encargalo.app-api.go/internal/auth/handler/request"
	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/labstack/echo/v4"
)

type Auth interface {
	SignInCustomer(e echo.Context) error
	LogoutCustomer(e echo.Context) error
	SignInEmployee(e echo.Context) error
	LogoutEmployee(e echo.Context) error
}

type auth struct {
	svc ports.AuthApp
	jwt jwt.Sessions
}

func NewAuthHandler(svc ports.AuthApp, jwt jwt.Sessions) Auth {
	return &auth{svc, jwt}
}

// SignInCustomer godoc
// @Summary      Iniciar sesión como cliente
// @Description  Autentica a un cliente usando su número de teléfono y contraseña.
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInRequest  true  "Datos de inicio de sesión"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      422    {object}  map[string]string  "Credenciales incorrectas"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/customers [post]
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

	cookie := cookie.NewCookieFactory("customers")

	e.SetCookie(cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")
}

// SignInEmployee godoc
// @Summary      Iniciar sesión como empleado
// @Description  Autentica a un empleado usando su número de teléfono y contraseña.
// @Description  Si las credenciales son correctas, se crea una sesión y se devuelve una cookie con un JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  request.SignInRequest  true  "Datos de inicio de sesión"
// @Success      201    {string}  string  "session created"
// @Failure      400    {object}  map[string]string  "Solicitud inválida (validación o JSON incorrecto)"
// @Failure      422    {object}  map[string]string  "Credenciales incorrectas"
// @Failure      500    {object}  map[string]string  "Error inesperado del servidor"
// @Router       /auth/sign-in/employee [post]
func (a *auth) SignInEmployee(e echo.Context) error {

	ctx := e.Request().Context()

	var signIn request.SignInRequest

	if err := e.Bind(&signIn); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := signIn.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := a.svc.SignInEmployee(ctx, signIn.Phone, signIn.Password)
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

	cookie := cookie.NewCookieFactory("employee")

	e.SetCookie(cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "session created")

}

// DeleteSession godoc
// @Summary Cierra la sesión actual del cliente o empleado autenticado
// @Description Elimina la sesión activa identificada por session_id en la cookie de autenticación
// @Tags Auth
// @Produce json
// @Success 200 {string} string "session deleted success"
// @Failure 404 {string} string "not found"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /auth/logout [delete]
func (s *auth) LogoutCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if err := s.svc.DeleteSession(ctx, session.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	cookie := cookie.NewCookieFactory("customers")
	e.SetCookie(cookie.DeleteCookieSession())

	return e.JSON(http.StatusOK, "session deleted success")
}

// DeleteSession godoc
// @Summary Cierra la sesión actual del cliente o empleado autenticado
// @Description Elimina la sesión activa identificada por session_id en la cookie de autenticación
// @Tags Auth
// @Produce json
// @Success 200 {string} string "session deleted success"
// @Failure 404 {string} string "not found"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /auth/logout [delete]
func (s *auth) LogoutEmployee(e echo.Context) error {

	ctx := e.Request().Context()

	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if err := s.svc.DeleteSession(ctx, session.ID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	cookie := cookie.NewCookieFactory("employee")
	e.SetCookie(cookie.DeleteCookieSession())

	return e.JSON(http.StatusOK, "session deleted success")
}
