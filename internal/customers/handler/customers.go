package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"Encargalo.app-api.go/internal/pkg/cookie"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/jwt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CustomersHandler interface {
	RegisterCustomer(e echo.Context) error
	SearchCustomer(e echo.Context) error
	UpdateCustomer(e echo.Context) error
	UpdatePassword(e echo.Context) error
}

type customersHandler struct {
	customerApp ports.CustomersApp
	jwt         jwt.Sessions
	cookie      cookie.Cookie
}

func NewCustomersHandler(customerApp ports.CustomersApp, jwt jwt.Sessions, cookie cookie.Cookie) CustomersHandler {
	return &customersHandler{customerApp, jwt, cookie}
}

// RegisterCustomer godoc
// @Summary      Registrar un nuevo cliente
// @Description  Registrar un nuevo cliente en el sistema con los datos proporcionados. Valida campos obligatorios como nombre, teléfono y contraseña.
// @Tags         Customers
// @Accept       json
// @Param        customer  body  dto.RegisterCustomer  true  "Datos del cliente"
// @Success      201  {string}  string  "customer successfully registered"
// @Failure      400 {string} string "Se retorna cuando hay un campo que no cumple con los requisitos o directamente el body se envía vacío."
// @Failure      500 {string} string "Se retorna cuando ocurre un error inexperado en el servidor."
// @Router       /customers [post]
func (c *customersHandler) RegisterCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer := dto.RegisterCustomer{}

	if err := e.Bind(&customer); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := customer.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	sessionID, err := c.customerApp.RegisterCustomer(ctx, customer)
	if err != nil {
		if errors.Is(err, errcustom.ErrPhoneAlreadyExist) {
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jwtSession, err := c.jwt.CreateSession(sessionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	cookie := c.cookie.CreateCookieSession(jwtSession)

	e.SetCookie(cookie)

	return e.JSON(http.StatusCreated, "customer successfully registered")
}

// SearchCustomer godoc
// @Summary Obtiene la información del cliente autenticado
// @Description Retorna los datos del cliente identificado por el customer_id contenido en el token
// @Tags Customers
// @Produce json
// @Success 200 {object} dto.CustomerResponse "Datos del cliente"
// @Failure 404 {string} string "not found"
// @Failure 500 {string} string "unexpected error"
// @Router /customers [get]
func (c *customersHandler) SearchCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("customer_id"))))
	if err != nil {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	customerID := dto.SearchCustomerBy{
		ID: customer_id,
	}

	custo, err := c.customerApp.SearchCustomerBy(ctx, customerID)
	if err != nil {
		switch err.Error() {
		case "not found":
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
		}
	}

	return e.JSON(http.StatusOK, custo.ToDomainDTO())
}

// UpdateCustomer godoc
// @Summary Actualiza la información del cliente autenticado
// @Description Actualiza los datos del cliente usando la información enviada en el cuerpo de la solicitud
// @Tags Customers
// @Accept json
// @Produce json
// @Param customer body dto.UpdateCustomer true "Datos del cliente a actualizar"
// @Success 200 {string} string "customer updated success"
// @Failure 400 {string} string "error de validación o formato inválido"
// @Failure 409 {string} string "customer not found"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /customers [put]
func (c *customersHandler) UpdateCustomer(e echo.Context) error {

	ctx := e.Request().Context()

	customer := dto.UpdateCustomer{}

	if err := e.Bind(&customer); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := customer.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	customer_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("customer_id"))))
	if err != nil {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	if err := c.customerApp.UpdateCustomer(ctx, customer_id, customer); err != nil {
		switch err.Error() {
		case "not found.":
			return echo.NewHTTPError(http.StatusConflict, "customer not found")
		case "phone al ready exist":
			return echo.NewHTTPError(http.StatusConflict, "phone al ready exist")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
		}
	}

	return e.JSON(http.StatusOK, "customer updated success")
}

// UpdatePassword godoc
// @Summary Actualiza la contraseña del cliente autenticado
// @Description Permite al cliente autenticado actualizar su contraseña, validando el formato y los requisitos establecidos
// @Tags Customers
// @Accept json
// @Produce json
// @Param password body dto.UpdatePassword true "Datos para actualizar la contraseña"
// @Success 200 {string} string "password updated success"
// @Failure 400 {string} string "error de validación o formato inválido"
// @Failure 409 {string} string "customer not found"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /customers/change-password [put]
func (c *customersHandler) UpdatePassword(e echo.Context) error {

	ctx := e.Request().Context()

	pass := dto.UpdatePassword{}

	if err := e.Bind(&pass); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := pass.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	customer_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("customer_id"))))
	if err != nil {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	if err := c.customerApp.UpdatePassword(ctx, customer_id, pass); err != nil {
		switch err.Error() {
		case "not found.":
			return echo.NewHTTPError(http.StatusConflict, "customer not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
		}
	}

	return e.JSON(http.StatusOK, "password updated success")
}
