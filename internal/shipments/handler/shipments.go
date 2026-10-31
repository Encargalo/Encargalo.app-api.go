package handler

import (
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"Encargalo.app-api.go/internal/shipments/domain/dtos"
	"Encargalo.app-api.go/internal/shipments/domain/ports"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ShipmentsHandler interface {
	RequestShipment(c echo.Context) error
	GetShipment(c echo.Context) error
	GetShipmentHistory(c echo.Context) error
	CancelShipment(c echo.Context) error
	EstimateFare(c echo.Context) error
	Live(c echo.Context) error
}

type shipmentsHandler struct {
	svc  ports.ShipmentsApp
	live ports.CustomerShipmentLive
}

func NewShipmentsHandler(svc ports.ShipmentsApp, live ports.CustomerShipmentLive) ShipmentsHandler {
	return &shipmentsHandler{svc: svc, live: live}
}

// RequestShipment godoc
// @Summary      Solicitar un envío
// @Description  Crea un envío de paquete (<5kg) a partir de una cotización vigente, con la misma tarifa que un viaje de pasajeros. method_payment solo acepta PagoMovil (comprobante, nace en payment_review) o Balance (debita la tarifa, nace en searching) — sin efectivo. sender/recipient indican quién retira y quién recibe el paquete: is_customer=true usa los datos de la cuenta; is_customer=false exige name y phone de un tercero. id (UUID, opcional) es la llave de idempotencia: un reintento con el mismo id devuelve el envío ya creado sin volver a cobrar.
// @Tags         Shipments
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.CreateShipmentRequest  true  "quote_id, method_payment (PagoMovil o Balance), sender, recipient y package_description"
// @Success      201  {object}  dtos.ShipmentResponse
// @Failure      400  {string}  string  "payload inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      404  {string}  string  "Cotización no encontrada, expirada o de otro cliente"
// @Failure      409  {string}  string  "El id ya lo usa un envío de otro cliente"
// @Failure      422  {string}  string  "No se pudo subir el comprobante o saldo insuficiente"
// @Router       /shipments [post]
func (h *shipmentsHandler) RequestShipment(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var body dtos.CreateShipmentRequest
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := body.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.RequestShipment(ctx, session.UserID, body)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusCreated, resp)
}

// EstimateFare godoc
// @Summary      Cotizar tarifa de un envío
// @Description  Calcula distancia, tiempo estimado y tarifa (USD/Bs) entre el punto de retiro y el de entrega, con la misma fórmula que los viajes de pasajeros, y la congela por 10 minutos en una cotización. El quote_id devuelto es obligatorio para crear el envío (POST /shipments): el precio mostrado acá es el que queda fijado, no se recalcula.
// @Tags         Shipments
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.FareEstimateRequest  true  "Direcciones y coordenadas de retiro/entrega"
// @Success      200  {object}  dtos.FareEstimateResponse
// @Failure      400  {string}  string  "payload inválido o coordenadas fuera del área de servicio"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      503  {string}  string  "Tasa BCV no disponible"
// @Router       /shipments/fare-estimate [post]
func (h *shipmentsHandler) EstimateFare(c echo.Context) error {
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

// GetShipment godoc
// @Summary      Detalle de un envío
// @Description  pickup_code y dropoff_code SIEMPRE vienen presentes en la respuesta, sin importar el estado del envío: el cliente los reenvía manualmente (WhatsApp, llamada) a quien esté físicamente presente en el retiro o la entrega, sea él mismo o un tercero. Si el envío ya tiene rider asignado, `rider` trae nombre, teléfono, foto y vehículo. En envíos cancelled o no_driver, `cancel_reason` dice por qué. Los cambios en vivo llegan por GET /shipments/live.
// @Tags         Shipments
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Shipment ID"
// @Success      200  {object}  dtos.ShipmentResponse
// @Failure      400  {string}  string  "id inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "El envío no pertenece a este cliente"
// @Failure      404  {string}  string  "No encontrado"
// @Router       /shipments/{id} [get]
func (h *shipmentsHandler) GetShipment(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	resp, err := h.svc.GetShipment(ctx, shipmentID, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// GetShipmentHistory godoc
// @Summary      Historial de envíos del cliente
// @Description  Cada envío con rider asignado incluye `rider` (nombre, teléfono, foto y vehículo). Los envíos cancelled y no_driver traen `cancel_reason` con el motivo.
// @Tags         Shipments
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.ShipmentHistoryResponse
// @Failure      401  {string}  string  "No autorizado"
// @Router       /shipments/history [get]
func (h *shipmentsHandler) GetShipmentHistory(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	resp, err := h.svc.GetShipmentHistory(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}

// CancelShipment godoc
// @Summary      Cancelar un envío
// @Description  Solo permitido mientras el envío esté en requested/searching/accepted. En payment_review responde 422 shipment_payment_review_pending: hay que esperar a que el staff resuelva el comprobante. Si se pagó con balance, el débito se reintegra. Si el Pago Móvil ya estaba aprobado, se acredita la tarifa en Bs a la cartera. La respuesta trae cancel_reason = customer_cancelled.
// @Tags         Shipments
// @Produce      json
// @Security     SessionCookie
// @Param        id  path  string  true  "Shipment ID"
// @Success      200  {object}  dtos.ShipmentResponse
// @Failure      400  {string}  string  "id inválido"
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "El envío no pertenece a este cliente"
// @Failure      404  {string}  string  "No encontrado"
// @Failure      422  {string}  string  "Pago en revisión, o el envío no se puede cancelar en su estado actual"
// @Router       /shipments/{id}/cancel [post]
func (h *shipmentsHandler) CancelShipment(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	shipmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "id inválido")
	}

	resp, err := h.svc.CancelShipment(ctx, shipmentID, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, resp)
}
