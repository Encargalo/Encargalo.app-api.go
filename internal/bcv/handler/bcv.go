package handler

import (
	"net/http"

	"Encargalo.app-api.go/internal/bcv/domain/models"
	"Encargalo.app-api.go/internal/bcv/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"github.com/labstack/echo/v4"
)

type BcvHandler interface {
	GetRate(e echo.Context) error
}

type bcvHandler struct {
	svc ports.BcvApp
}

func NewBcvHandler(svc ports.BcvApp) BcvHandler {
	return &bcvHandler{svc}
}

// GetRate godoc
// @Summary      Obtener tasa del dólar en bolívares
// @Description  Retorna la última tasa de cambio USD/Bs publicada por el BCV, cacheada por el servidor.
// @Tags         Bcv
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200 {object}  models.ExchangeRate
// @Failure      401 {string}  string  "token inválido"
// @Failure      500 {string}  string  "Error interno del servidor"
// @Router       /bcv/rate [get]
func (h *bcvHandler) GetRate(e echo.Context) error {
	var rate *models.ExchangeRate
	rate, err := h.svc.GetRate(e.Request().Context())
	if err != nil {
		return errcustom.ErrUnexpectedError
	}

	return response.Send(e, http.StatusOK, rate)
}
