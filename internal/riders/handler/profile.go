package handler

import (
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	"Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
)

type RiderProfileHandler interface {
	GetProfile(c echo.Context) error
	UpsertDeviceToken(c echo.Context) error
}

type riderProfileHandler struct {
	svc ports.RiderProfileApp
}

func NewRiderProfileHandler(svc ports.RiderProfileApp) RiderProfileHandler {
	return &riderProfileHandler{svc: svc}
}

// GetProfile godoc
//
// @Summary      Perfil del conductor autenticado
// @Description  Devuelve los datos del propio conductor (`driver`: id, nombre, apellido y foto de perfil) y su vehículo (`vehicle`: tipo, placa, marca, modelo y color). Solo incluye lo necesario para el panel de conductor activo: no expone contacto, notas ni fotos de documentos. Los campos sin dato se omiten; si el rider aún no tiene perfil, `vehicle` es un objeto vacío.
// @Tags         Riders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.RiderSelfProfileResponse
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      404  {string}  string  "Cuenta no encontrada"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /riders/profile [get]
func (h *riderProfileHandler) GetProfile(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var resp dtos.RiderSelfProfileResponse
	resp, err := h.svc.GetOwnProfile(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// UpsertDeviceToken godoc
//
// @Summary      Registrar token de push (FCM)
// @Description  Registra o reemplaza el token FCM del dispositivo del rider autenticado. La app lo llama al iniciar sesión y cada vez que el token rota. Un rider tiene un solo token activo — un login nuevo pisa el anterior. Se usa para mandarle push cuando se le oferta un viaje o un pedido, en paralelo al WebSocket.
// @Tags         Riders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.UpsertDeviceTokenRequest  true  "Token FCM y plataforma"
// @Success      204  "Sin contenido"
// @Failure      400  {string}  string  "Datos inválidos"
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /riders/device-token [put]
func (h *riderProfileHandler) UpsertDeviceToken(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var req dtos.UpsertDeviceTokenRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}

	if err := h.svc.UpsertDeviceToken(ctx, session.UserID, req); err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
