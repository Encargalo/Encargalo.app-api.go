package handler

import (
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/history/domain/dtos"
	"Encargalo.app-api.go/internal/history/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
)

type HistoryHandler interface {
	GetHistory(c echo.Context) error
}

type historyHandler struct {
	svc ports.HistoryApp
}

func NewHistoryHandler(svc ports.HistoryApp) HistoryHandler {
	return &historyHandler{svc: svc}
}

// GetHistory godoc
// @Summary      Historial combinado de ordenes, viajes y envíos
// @Description  Devuelve en un solo listado las órdenes, viajes y envíos del customer autenticado, ordenados de más reciente a más antiguo. Cada item trae "type" ("order", "trip" o "shipment") y solo el campo correspondiente a ese tipo, con el mismo shape que su endpoint de historial individual (/orders/history, /trips/history, /shipments/history).
// @Tags         History
// @Produce      json
// @Security     SessionCookie
// @Success      200  {array}   dtos.HistoryItem  "Historial combinado"
// @Failure      401  {string}  string            "No autorizado"
// @Router       /history [get]
func (h *historyHandler) GetHistory(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.GetHistory(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	// Conversión identidad: solo para que swag resuelva dtos.HistoryItem en @Success.
	return response.Send(c, http.StatusOK, dtos.HistoryResponse(resp))
}
