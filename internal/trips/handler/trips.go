package handler

import (
	"net/http"
	"strconv"

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
	NearbyRiders(c echo.Context) error
	Live(c echo.Context) error
}

type tripsHandler struct {
	svc  ports.TripsApp
	live ports.CustomerTripLive
}

func NewTripsHandler(svc ports.TripsApp, live ports.CustomerTripLive) TripsHandler {
	return &tripsHandler{svc: svc, live: live}
}

// RequestTrip godoc
// @Summary      Solicitar un viaje
// @Description  Crea un viaje a partir de una cotización vigente (POST /trips/fare-estimate). Con method_payment=PagoMovil el comprobante es obligatorio y el viaje queda en payment_review hasta que el staff lo apruebe. Con method_payment=Balance se debita la tarifa completa en Bs y el viaje pasa directo a searching. Pickup/dropoff y fare quedan fijados por la cotización.
// @Tags         Trips
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.CreateTripRequest  true  "quote_id y method_payment (PagoMovil con comprobante, o Balance)"
// @Success      201  {object}  dtos.TripResponse
// @Failure      400  {string}  string  "payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Cotización no encontrada, expirada o de otro cliente"
// @Failure      422  {string}  string  "No se pudo subir el comprobante o saldo insuficiente"
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

// NearbyRiders godoc
// @Summary      Conductores cercanos para el mapa de cotización
// @Description  Devuelve hasta 30 pines de conductores disponibles en modo viajes, a unos 2 km del punto. Cada coordenada está redondeada a ~100 m. No incluye id, nombre ni datos del vehículo. Pensado para consultarse cada pocos segundos mientras el mapa está abierto.
// @Tags         Trips
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        lat  query  number  true  "Latitud del centro del mapa"
// @Param        lng  query  number  true  "Longitud del centro del mapa"
// @Success      200  {object}  dtos.NearbyRidersResponse
// @Failure      400  {string}  string  "coordenadas inválidas"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      429  {string}  string  "rate limit"
// @Router       /trips/nearby-riders [get]
func (h *tripsHandler) NearbyRiders(c echo.Context) error {
	lat, lng, err := parseMapPoint(c.QueryParam("lat"), c.QueryParam("lng"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "coordenadas inválidas")
	}

	resp, err := h.svc.NearbyRiders(lat, lng)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

func parseMapPoint(latRaw, lngRaw string) (float64, float64, error) {
	lat, err := strconv.ParseFloat(latRaw, 64)
	if err != nil {
		return 0, 0, err
	}
	lng, err := strconv.ParseFloat(lngRaw, 64)
	if err != nil {
		return 0, 0, err
	}
	return lat, lng, nil
}

// GetTrip godoc
// @Summary      Detalle de un viaje
// @Description  Si el viaje ya tiene rider asignado, `rider` trae nombre, teléfono, foto y vehículo. No incluye documentos KYC. pickup_code aparece desde accepted hasta que el pasajero sube; dropoff_code aparece en in_progress. Si un Pago Móvil aprobado se reintegró a la cartera, `payment` trae transaction_id, credited_bs y refund_eligible_until. Los cambios en vivo llegan por GET /trips/live.
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
// @Description  Cada viaje con rider asignado incluye `rider` (nombre, teléfono, foto y vehículo). No incluye documentos KYC.
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
// @Description  Solo permitido mientras el viaje esté en requested/searching/accepted/arriving. En payment_review responde 422 trip_payment_review_pending: hay que esperar a que el staff resuelva el comprobante. Si se pagó con balance, el débito se reintegra. Si el Pago Móvil ya estaba aprobado (searching, accepted o arriving), se acredita la tarifa en Bs a la cartera y la respuesta trae payment.transaction_id.
// @Tags         Trips
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Trip ID"
// @Success      200  {object}  dtos.TripResponse
// @Failure      400  {string}  string  "id inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "El viaje no pertenece a este cliente"
// @Failure      404  {string}  string  "No encontrado"
// @Failure      422  {string}  string  "Pago en revisión, o el viaje no se puede cancelar en su estado actual"
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
