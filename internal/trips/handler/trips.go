package handler

import (
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"Encargalo.app-api.go/internal/trips/domain/dtos"
	"Encargalo.app-api.go/internal/trips/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type TripsHandler interface {
	RequestTrip(c echo.Context) error
	GetTrip(c echo.Context) error
	GetTripHistory(c echo.Context) error
	CancelTrip(c echo.Context) error
}

type tripsHandler struct {
	svc ports.TripsApp
}

func NewTripsHandler(svc ports.TripsApp) TripsHandler {
	return &tripsHandler{svc: svc}
}

// RequestTrip godoc
// @Summary      Solicitar un viaje
// @Description  Crea un viaje en estado "requested" con punto de recogida y destino. El matching con conductores cercanos no está incluido en este endpoint.
// @Tags         Trips
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.CreateTripRequest  true  "Datos del viaje"
// @Success      201  {object}  dtos.TripResponse
// @Failure      400  {string}  string  "payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Router       /trips [post]
func (h *tripsHandler) RequestTrip(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var body dtos.CreateTripRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.RequestTrip(ctx, session.UserID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusCreated, resp)
}

// GetTrip godoc
// @Summary      Detalle de un viaje
// @Tags         Trips
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Trip ID"
// @Success      200  {object}  dtos.TripResponse
// @Failure      400  {string}  string  "id inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "El viaje no pertenece a este cliente"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /trips/{id} [get]
func (h *tripsHandler) GetTrip(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	resp, err := h.svc.GetTrip(ctx, tripID, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// GetTripHistory godoc
// @Summary      Historial de viajes del cliente
// @Tags         Trips
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.TripHistoryResponse
// @Failure      401  {string}  string  "No autorizado"
// @Router       /trips/history [get]
func (h *tripsHandler) GetTripHistory(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.GetTripHistory(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// CancelTrip godoc
// @Summary      Cancelar un viaje
// @Description  Solo permitido mientras el viaje esté en requested/searching/accepted/arriving.
// @Tags         Trips
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Trip ID"
// @Success      200  {object}  dtos.TripResponse
// @Failure      400  {string}  string  "id inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "El viaje no pertenece a este cliente"
// @Failure      404  {string}  string  "No encontrado"
// @Failure      422  {string}  string  "El viaje no se puede cancelar en su estado actual"
// @Router       /trips/{id}/cancel [post]
func (h *tripsHandler) CancelTrip(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	resp, err := h.svc.CancelTrip(ctx, tripID, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}
