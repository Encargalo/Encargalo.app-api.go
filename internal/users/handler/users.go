package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cookie"

	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/labstack/echo/v4"

	"Encargalo.app-api.go/internal/auth/domain/models"
	portsAuth "Encargalo.app-api.go/internal/auth/domain/ports"
)

type UsersHandler interface {
	RegisterCustomer(e echo.Context) error
	SearchCustomer(e echo.Context) error
	UpdateCustomer(e echo.Context) error
	UpdatePassword(e echo.Context) error
}

type customersHandler struct {
	customerApp ports.UsersApp
	auth        portsAuth.AuthApp
	jwt         jwt.Sessions
	cookie      cookie.CookieHandler
}

func NewUsersHandler(customerApp ports.UsersApp, auth portsAuth.AuthApp, jwt jwt.Sessions, cookie cookie.CookieHandler) UsersHandler {
	return &customersHandler{customerApp, auth, jwt, cookie}
}

// RegisterCustomer godoc
// @Summary      Registrar un nuevo cliente
// @Description  Registra un nuevo cliente en el sistema. Al completarse, crea automáticamente la sesión y establece la cookie de autenticación.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        customer  body      dto.RegisterCustomer  true  "Datos del cliente"
// @Success      201       {string}  string                "customer successfully registered"
// @Failure      400       {string}  string                "Campo inválido o body vacío"
// @Failure      409       {string}  string                "El teléfono ya está registrado"
// @Failure      500       {string}  string                "Error interno del servidor"
// @Router       /users [post]
func (c *customersHandler) RegisterCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer := dto.RegisterCustomer{}

	if err := e.Bind(&customer); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := customer.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.customerApp.RegisterCustomer(ctx, customer); err != nil {
		return errcustom.ToHTTPError(err)
	}

	sessionId, err := c.auth.SignInCustomer(ctx, customer.Phone, customer.Password)
	if err != nil {
		fmt.Println()
	}

	jwtSession, err := c.jwt.CreateSession(sessionId)
	if err != nil {
		return errcustom.ErrUnexpectedError
	}

	e.SetCookie(c.cookie.CreateCookieSession(jwtSession))

	return e.JSON(http.StatusCreated, "customer successfully registered")
}

// SearchCustomer godoc
// @Summary      Obtener datos del cliente autenticado
// @Description  Retorna los datos del cliente identificado por el customer_id contenido en el token de sesión.
// @Tags         Users
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dto.CustomerResponse  "Datos del cliente"
// @Failure      401  {string}  string                "No autorizado"
// @Failure      404  {string}  string                "Cliente no encontrado"
// @Failure      500  {string}  string                "Error interno del servidor"
// @Router       /users [get]
func (c *customersHandler) SearchCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	customerID := dto.SearchCustomerBy{
		ID: session.UserID,
	}

	custo, err := c.customerApp.SearchCustomerBy(ctx, customerID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return e.JSON(http.StatusOK, custo.ToDomainDTO())
}

// UpdateCustomer godoc
// @Summary Actualiza la información del cliente autenticado
// @Description Actualiza los datos del cliente usando la información enviada en el cuerpo de la solicitud
// @Tags Users
// @Accept json
// @Produce json
// @Param customer body dto.UpdateCustomer true "Datos del cliente a actualizar"
// @Success 200 {string} string "customer updated success"
// @Failure 400 {string} string "error de validación o formato inválido"
// @Failure 404 {string} string "customer not found"
// @Failure 409 {string} string "phone already exist"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /users [put]
func (c *customersHandler) UpdateCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer := dto.UpdateCustomer{}

	if err := e.Bind(&customer); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := customer.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if err := c.customerApp.UpdateCustomer(ctx, session.UserID, customer); err != nil {
		return errcustom.ToHTTPError(err)
	}

	return e.JSON(http.StatusOK, "customer updated success")
}

// UpdatePassword godoc
// @Summary Actualiza la contraseña del cliente autenticado
// @Description Permite al cliente autenticado actualizar su contraseña, validando el formato y los requisitos establecidos
// @Tags Users
// @Accept json
// @Produce json
// @Param password body dto.UpdatePassword true "Datos para actualizar la contraseña"
// @Success 200 {string} string "password updated success"
// @Failure 400 {string} string "error de validación o formato inválido"
// @Failure 404 {string} string "customer not found"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /users/change-password [put]
func (c *customersHandler) UpdatePassword(e echo.Context) error {

	ctx := e.Request().Context()

	pass := dto.UpdatePassword{}

	if err := e.Bind(&pass); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := pass.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if err := c.customerApp.UpdatePassword(ctx, session.UserID, pass); err != nil {
		return errcustom.ToHTTPError(err)
	}

	return e.JSON(http.StatusOK, "password updated success")
}
