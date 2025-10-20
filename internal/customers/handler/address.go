package handler

import (
	"fmt"
	"net/http"
	"strings"

	"Encargalo.app-api.go/internal/customers/domain/dto"
	"Encargalo.app-api.go/internal/customers/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CustomersAddressHandler interface {
	RegisterAddress(e echo.Context) error
	SearchAllAdrress(e echo.Context) error
	DeleteAddress(e echo.Context) error
}

type customersAddrssHandler struct {
	svc ports.CustomersAddressApp
}

func NewCustomersAddressHandler(svc ports.CustomersAddressApp) CustomersAddressHandler {
	return &customersAddrssHandler{svc}
}

// RegisterAddress godoc
// @Summary Registra una nueva dirección para el cliente autenticado
// @Description Registra una dirección asociada al customer_id obtenido del contexto
// @Tags Customers Address
// @Accept json
// @Produce json
// @Param address body dto.Address true "Datos de la dirección"
// @Success 201 {string} string "address registred"
// @Failure 400 {string} string "error de validación o parseo"
// @Failure 500 {string} string "unexpected error"
// @Router /customers/address [post]
func (c *customersAddrssHandler) RegisterAddress(e echo.Context) error {

	ctx := e.Request().Context()

	address := dto.Address{}

	if err := e.Bind(&address); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := address.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	customer_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("customer_id"))))
	if err != nil {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	if err := c.svc.RegisterAddress(ctx, customer_id, address); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	return e.JSON(http.StatusCreated, "address registred")
}

// SearchAllAdrress godoc
// @Summary Obtiene todas las direcciones del cliente autenticado
// @Description Retorna un arreglo con todas las direcciones asociadas al cliente identificado en el token
// @Tags Customers Address
// @Produce json
// @Success 200 {array} dto.Address "Lista de direcciones"
// @Failure 404 {string} string "not found"
// @Failure 500 {string} string "unexpected error"
// @Router /customers/address [get]
func (c *customersAddrssHandler) SearchAllAdrress(e echo.Context) error {

	ctx := e.Request().Context()

	customer_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("customer_id"))))
	if err != nil {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	addresses, err := c.svc.SearchAllAddress(ctx, customer_id)
	if err != nil {
		switch err.Error() {
		case "not found":
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
		}
	}

	return e.JSON(http.StatusOK, addresses)
}

// DeleteAddress godoc
// @Summary Elimina una dirección del cliente autenticado
// @Description Elimina la dirección especificada por su ID, siempre que pertenezca al cliente autenticado
// @Tags Customers Address
// @Param address path string true "ID de la dirección (UUID)"
// @Produce json
// @Success 200 {string} string "address deleted success"
// @Failure 500 {string} string "unexpected error"
// @Security SessionCookie
// @Router /customers/address/{address} [delete]
func (c *customersAddrssHandler) DeleteAddress(e echo.Context) error {

	ctx := e.Request().Context()

	address_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(e.Param("address"))))
	if err != nil {
		fmt.Println("Error al obtener el address_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	customer_id, err := uuid.Parse(strings.TrimSpace(fmt.Sprintln(ctx.Value("customer_id"))))
	if err != nil {
		fmt.Println("Error al obtener el customer_id")
		return echo.NewHTTPError(http.StatusInternalServerError, "unexpected error")
	}

	if err := c.svc.DeleteAddress(ctx, customer_id, address_id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return e.JSON(http.StatusOK, "address deleted success")
}
