package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/tracking/domain/dtos"
	"Encargalo.app-api.go/internal/tracking/domain/ports"
	"github.com/labstack/echo/v4"
)

type TrackingHandler interface {
	Track(c echo.Context) error
}

type trackingHandler struct {
	svc ports.TrackingApp
}

func NewTrackingHandler(svc ports.TrackingApp) TrackingHandler {
	return &trackingHandler{svc: svc}
}

// Track godoc
// @Summary      Registrar evento de tracking
// @Description  Persiste un evento genérico (pixel, cta u otro type). El body `params` se guarda tal cual en JSONB; el servidor inyecta user_id y session_id desde la sesión. Idempotente por `id`.
// @Tags         Tracking
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        event  body      dtos.TrackEvent  true  "Evento de tracking"
// @Success      204    {string}  string           "Registrado o ya existía"
// @Failure      400    {string}  string           "Body inválido, type inválido o params demasiado grandes"
// @Failure      401    {string}  string           "No autorizado"
// @Failure      429    {string}  string           "Demasiadas solicitudes"
// @Failure      500    {string}  string           "Error interno del servidor"
// @Router       /tracking [post]
func (h *trackingHandler) Track(c echo.Context) error {
	ctx := c.Request().Context()

	session, ok := ctx.Value(models.SessionKey).(models.ActiveSession)
	if !ok {
		return errcustom.ErrUnautorized
	}

	var event dtos.TrackEvent
	if err := c.Bind(&event); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := event.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.svc.Track(ctx, session.UserID, session.ID, event); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
