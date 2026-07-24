package handler

import (
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/users/domain/dto"
	"Encargalo.app-api.go/internal/users/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UsersAddressHandler interface {
	RegisterAddress(e echo.Context) error
	SearchAllAdrress(e echo.Context) error
	DeleteAddress(e echo.Context) error
}

type customersAddrssHandler struct {
	svc ports.UsersAddressApp
}

func NewUsersAddressHandler(svc ports.UsersAddressApp) UsersAddressHandler {
	return &customersAddrssHandler{svc}
}

// RegisterAddress godoc
// @Summary      Registrar una nueva dirección
// @Description  Registra una dirección asociada al cliente autenticado, identificado por el token de sesión.
// @Tags         Users Address
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        address  body      dto.Address  true  "Datos de la dirección"
// @Success      201      {string}  string       "address registred"
// @Failure      400      {string}  string       "Error de validación o parseo"
// @Failure      401      {string}  string       "No autorizado"
// @Failure      500      {string}  string       "Error interno del servidor"
// @Router       /users/address [post]
func (c *customersAddrssHandler) RegisterAddress(e echo.Context) error {

	ctx := e.Request().Context()

	var address dto.Address

	if err := e.Bind(&address); err != nil {
		slog.Error("Bind error:", "err", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := address.Validate(); err != nil {
		slog.Error("Validation error:", "err", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if err := c.svc.RegisterAddress(ctx, session.UserID, address); err != nil {
		slog.Error("Error registrando dirección:", "err", err)
		return errcustom.ToHTTPError(err)
	}

	return e.JSON(http.StatusCreated, "address registred")
}

// SearchAllAdrress godoc
// @Summary      Obtener todas las direcciones del cliente autenticado
// @Description  Retorna un arreglo con todas las direcciones asociadas al cliente identificado en el token de sesión.
// @Tags         Users Address
// @Produce      json
// @Security     SessionCookie
// @Success      200  {array}   dto.Address  "Lista de direcciones"
// @Failure      401  {string}  string       "No autorizado"
// @Failure      404  {string}  string       "No se encontraron direcciones"
// @Failure      500  {string}  string       "Error interno del servidor"
// @Router       /users/address [get]
func (c *customersAddrssHandler) SearchAllAdrress(e echo.Context) error {

	ctx := e.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	addresses, err := c.svc.SearchAllAddress(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}

	return e.JSON(http.StatusOK, addresses)
}

// DeleteAddress godoc
// @Summary      Eliminar una dirección del cliente autenticado
// @Description  Elimina la dirección especificada por su ID, siempre que pertenezca al cliente autenticado.
// @Tags         Users Address
// @Produce      json
// @Security     SessionCookie
// @Param        address_id  query     string  true  "ID de la dirección (UUID)"
// @Success      200         {string}  string  "address deleted success"
// @Failure      400         {string}  string  "address_id es requerido o inválido"
// @Failure      401         {string}  string  "No autorizado"
// @Failure      500         {string}  string  "Error interno del servidor"
// @Router       /users/address [delete]
func (c *customersAddrssHandler) DeleteAddress(e echo.Context) error {

	ctx := e.Request().Context()

	address_id := e.QueryParam("address_id")

	if address_id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "address_id es requerido")
	}

	if _, err := uuid.Parse(address_id); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "address_id inválido")
	}

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if err := c.svc.DeleteAddress(ctx, session.UserID, address_id); err != nil {
		return errcustom.ToHTTPError(err)
	}

	return e.JSON(http.StatusOK, "address deleted success")
}
