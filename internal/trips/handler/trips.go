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
	EstimateFare(c echo.Context) error
}

type tripsHandler struct {
	svc ports.TripsApp
}

func NewTripsHandler(svc ports.TripsApp) TripsHandler {
	return &tripsHandler{svc: svc}
}

// RequestTrip godoc
// @Summary      Solicitar un viaje
// @Description  Crea un viaje en estado "payment_review" a partir de una cotización vigente (ver POST /trips/fare-estimate) y un comprobante de Pago Móvil. Pickup/dropoff y fare quedan fijados por esa cotización. El viaje no se oferta a conductores hasta que el staff apruebe el comprobante en la cola unificada de pagos.
// @Tags         Trips
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.CreateTripRequest  true  "quote_id, method_payment y comprobante"
// @Success      201  {object}  dtos.TripResponse
// @Failure      400  {string}  string  "payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Cotización no encontrada, expirada o de otro cliente"
// @Failure      422  {string}  string  "No se pudo subir el comprobante"
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

// EstimateFare godoc
// @Summary      Cotizar tarifa de un viaje
// @Description  Calcula distancia, tiempo estimado y tarifa (USD/Bs) entre pickup y destino, y la congela por 10 minutos en una cotización. El quote_id devuelto es obligatorio para crear el viaje (POST /trips): el precio mostrado acá es el que queda fijado, no se recalcula.
// @Tags         Trips
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.FareEstimateRequest  true  "Direcciones y coordenadas de pickup/destino"
// @Success      200  {object}  dtos.FareEstimateResponse
// @Failure      400  {string}  string  "payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      503  {string}  string  "Tasa BCV no disponible"
// @Router       /trips/fare-estimate [post]
func (h *tripsHandler) EstimateFare(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var body dtos.FareEstimateRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	body.CustomerID = session.UserID
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.EstimateFare(ctx, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
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
