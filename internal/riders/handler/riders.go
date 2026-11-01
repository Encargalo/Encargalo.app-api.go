package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/riders/domain/dtos"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/response"
	"Encargalo.app-api.go/internal/shared/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type RiderHandler interface {
	HandleWebSocket(c echo.Context) error
	GetActiveTrips(c echo.Context) error
	ReportLocation(c echo.Context) error
	SetAvailability(c echo.Context) error
}

type riderHandler struct {
	svc      ridersPorts.RidersApp
	upgrader websocket.Upgrader
}

func New(svc ridersPorts.RidersApp) RiderHandler {
	cfg := config.Get()
	return &riderHandler{
		svc:      svc,
		upgrader: ws.BuildUpgrader(cfg.IsProduction(), cfg.Server.AllowedOrigins),
	}
}

// HandleWebSocket godoc
//
// @Summary      WebSocket de gestión de órdenes para riders
// @Description  Establece una conexión WebSocket exclusiva para el rol Rider.
// Al conectar el rider entra al pool de disponibles y recibe sus órdenes activas con `pickup_code`.
//
// ### Mensajes entrantes soportados
// **accept_order**
//
//	{"type":"accept_order","order_id":"<uuid>"}
//
// → Intenta asignar la orden al rider. Primer rider que acepta gana. Un rider en bici
// no puede tomar pedidos de negocios Turbo ni de supermercado: responde
// `{"type":"error","message":"tu vehículo no puede llevar este pedido"}`.
//
// **update_location**
//
//	{"type":"update_location","latitude":10.48,"longitude":-66.90}
//
// → Guarda la ubicación en memoria (solo relevante en modo trips). En modo trips,
// la primera ubicación de la conexión (o un cambio de celda H3) reenvía como
// `trip_offer` los viajes que siguen buscando conductor dentro del radio de su
// ronda actual, salvo los que el rider canceló o si ya tiene un viaje en curso.
//
// **accept_trip**
//
//	{"type":"accept_trip","trip_id":"<uuid>"}
//
// → Intenta asignar el viaje al rider. Primer rider que acepta gana. Si el rider ya
// tiene un viaje en accepted, arriving o in_progress responde
// `{"type":"error","message":"ya tienes un viaje en curso"}`.
//
// **advance_trip**
//
//	{"type":"advance_trip","trip_id":"<uuid>","code":"1234"}
//
// → Solo el rider asignado. accepted→arriving no pide código. arriving→in_progress
// exige el pickup_code que ve el cliente. in_progress→completed exige el dropoff_code.
// El viaje que recibe el rider no incluye esos códigos. Al completar acredita la ganancia.
//
// **cancel_trip**
//
//	{"type":"cancel_trip","trip_id":"<uuid>"}
//
// → Solo el rider asignado, y solo en accepted o arriving. Libera el viaje
// (searching, sin rider) y lo vuelve a ofertar a otros riders cercanos.
// El rider que canceló no vuelve a recibir esa oferta.
//
// **accept_shipment**
//
//	{"type":"accept_shipment","shipment_id":"<uuid>"}
//
// → Intenta asignar el envío al rider (modo orders, sin filtro de vehículo:
// cualquiera lleva <5kg). Primero que acepta gana. No cuenta como "ocupado":
// un rider puede llevar varios envíos y pedidos a la vez, igual que hoy con orders.
//
// **advance_shipment**
//
//	{"type":"advance_shipment","shipment_id":"<uuid>","code":"1234"}
//
// → Solo el rider asignado. accepted→in_transit exige el pickup_code que ve
// el cliente. in_transit→delivered exige el dropoff_code y acredita la
// ganancia del rider. El envío que recibe el rider no incluye esos códigos:
// el cliente los reenvía de palabra a quien esté presente en retiro o entrega.
//
// **cancel_shipment**
//
//	{"type":"cancel_shipment","shipment_id":"<uuid>"}
//
// → Solo el rider asignado, y solo en accepted. Libera el envío (searching,
// sin rider) y lo vuelve a ofertar a todo el pool en modo orders.
//
// ### Mensajes salientes
//   - `{"type":"connected"}` — confirmación de conexión. En modo trips le siguen
//     los viajes que el rider ya tiene asignados (accepted, arriving o in_progress),
//     como `trip_status` sin códigos.
//   - `{"type":"new_order","order":{...}}` — nueva orden disponible (sin códigos). Un rider
//     en bici no recibe las de negocios Turbo ni de supermercado, ni en el broadcast ni
//     entre las disponibles que llegan al conectar.
//   - `{"type":"order_accepted","order":{...}}` — asignación exitosa (con pickup_code)
//   - `{"type":"order_taken","order_id":"<uuid>"}` — otro rider se adelantó
//   - `{"type":"order_update","order":{...}}` — actualización de orden asignada (con pickup_code)
//   - `{"type":"trip_offer","trip":{...}}` — viaje ofrecido a hasta 3 riders cercanos. `fare_usd`/`fare_bs` son la ganancia del rider (90% de la tarifa), igual en todos los mensajes de viaje; `collect_usd`/`collect_bs` son el total a cobrar en efectivo. Los montos en Bs usan siempre la tasa BCV congelada en la cotización del viaje. Incluye `customer` con el nombre y el apellido del pasajero, sin teléfono. Si nadie acepta en 10 min, otra ronda ofrece a otros 3 con un anillo H3 más de radio, hasta 4 rondas. Una ronda sin riders elegibles espera 5 min y también cuenta como ronda. Al terminar la ronda 4 el viaje pasa a no_driver.
//   - `{"type":"trip_accepted","trip":{...}}` — asignación de viaje exitosa. Solo el rider que aceptó recibe `trip.customer` con `name` y `phone` (sin apellido).
//   - `{"type":"trip_taken","trip_id":"<uuid>"}` — otro rider se adelantó
//   - `{"type":"trip_status","trip":{...}}` — trip avanzó de estado (respuesta a advance_trip) o quedó liberado (respuesta a cancel_trip). `trip.customer` va solo al rider asignado mientras el viaje está en accepted, arriving o in_progress. Al completar o liberar el viaje, el bloque no se envía.
//   - `{"type":"trip_cancelled","trip_id":"<uuid>"}` — el cliente canceló el viaje. Lo reciben el rider asignado (accepted o arriving) y los riders que tenían la oferta viva. No dice quién canceló ni el motivo. El rider asignado deja de reenviar su ubicación.
//   - `{"type":"new_shipment","shipment":{...}}` — nuevo envío disponible, mandado a todo rider conectado en modo orders (sin filtro de vehículo). Sin pickup_code/dropoff_code. `fare_usd`/`fare_bs` son la ganancia del rider (90% de la tarifa), igual en todos los mensajes de envío; los Bs usan la tasa BCV congelada en la cotización del envío. Se reenvía también al conectar, junto con los envíos que sigan en searching en ese momento.
//   - `{"type":"shipment_accepted","shipment":{...}}` — asignación exitosa (sin códigos)
//   - `{"type":"shipment_taken","shipment_id":"<uuid>"}` — otro rider se adelantó
//   - `{"type":"shipment_status","shipment":{...}}` — envío avanzó de estado (respuesta a advance_shipment) o quedó liberado (respuesta a cancel_shipment). También se reenvía al conectar para los envíos ya asignados al rider.
//   - `{"type":"shipment_cancelled","shipment_id":"<uuid>"}` — el cliente canceló el envío. Solo lo recibe el rider ya asignado, si lo hay.
//   - `{"type":"error","message":"..."}` — error procesando mensaje
//
// @Tags         Riders WebSocket
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        mode  query  string  false  "orders|trips (default orders)"
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /orders/rider [get]
func (h *riderHandler) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	riderID := session.UserID

	mode := riderModels.RiderMode(c.QueryParam("mode"))

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading websocket for rider", "error", err)
		return err
	}

	client := ws.NewClient(conn)
	go client.WritePump()

	if err := h.svc.Register(ctx, riderID, client, mode); err != nil {
		_ = conn.Close()
		return err
	}

	slog.Info("rider WebSocket connected", "riderID", riderID)

	defer func() {
		slog.Info("rider WebSocket disconnected", "riderID", riderID)
		h.svc.Unregister(riderID, client)
		_ = conn.Close()
	}()

	client.Send([]byte(`{"type":"connected"}`))

	if mode == riderModels.RiderModeTrips {
		if err := h.svc.SendActiveTrips(ctx, riderID, client); err != nil {
			slog.Error("Error reenviando viajes activos al rider", "riderID", riderID, "error", err)
		}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Info("WebSocket rider cerrado", "riderID", riderID)
			} else {
				slog.Error("Error leyendo websocket rider", "error", err)
			}
			return nil
		}

		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg, &base); err != nil {
			slog.Warn("Mensaje rider inválido", "error", err)
			continue
		}

		switch base.Type {
		case "accept_order":
			var payload struct {
				Type    string `json:"type"`
				OrderID string `json:"order_id"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.OrderID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			orderID, err := uuid.Parse(payload.OrderID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "order_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.AcceptOrder(ctx, riderID, orderID, client); err != nil {
				slog.Error("Error aceptando orden", "riderID", riderID, "orderID", orderID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "accept_trip":
			var payload struct {
				Type   string `json:"type"`
				TripID string `json:"trip_id"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.TripID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			tripID, err := uuid.Parse(payload.TripID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "trip_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.AcceptTrip(ctx, riderID, tripID, client); err != nil {
				slog.Error("Error aceptando viaje", "riderID", riderID, "tripID", tripID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "advance_trip":
			var payload struct {
				Type   string `json:"type"`
				TripID string `json:"trip_id"`
				Code   string `json:"code"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.TripID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			tripID, err := uuid.Parse(payload.TripID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "trip_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.AdvanceTrip(ctx, riderID, tripID, payload.Code, client); err != nil {
				slog.Error("Error avanzando viaje", "riderID", riderID, "tripID", tripID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "cancel_trip":
			var payload struct {
				Type   string `json:"type"`
				TripID string `json:"trip_id"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.TripID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			tripID, err := uuid.Parse(payload.TripID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "trip_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.CancelAcceptedTrip(ctx, riderID, tripID, client); err != nil {
				slog.Error("Error cancelando viaje", "riderID", riderID, "tripID", tripID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "accept_shipment":
			var payload struct {
				Type       string `json:"type"`
				ShipmentID string `json:"shipment_id"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.ShipmentID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			shipmentID, err := uuid.Parse(payload.ShipmentID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "shipment_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.AcceptShipment(ctx, riderID, shipmentID, client); err != nil {
				slog.Error("Error aceptando envío", "riderID", riderID, "shipmentID", shipmentID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "advance_shipment":
			var payload struct {
				Type       string `json:"type"`
				ShipmentID string `json:"shipment_id"`
				Code       string `json:"code"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.ShipmentID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			shipmentID, err := uuid.Parse(payload.ShipmentID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "shipment_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.AdvanceShipment(ctx, riderID, shipmentID, payload.Code, client); err != nil {
				slog.Error("Error avanzando envío", "riderID", riderID, "shipmentID", shipmentID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "cancel_shipment":
			var payload struct {
				Type       string `json:"type"`
				ShipmentID string `json:"shipment_id"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil || payload.ShipmentID == "" {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			shipmentID, err := uuid.Parse(payload.ShipmentID)
			if err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "shipment_id inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.CancelAcceptedShipment(ctx, riderID, shipmentID, client); err != nil {
				slog.Error("Error cancelando envío", "riderID", riderID, "shipmentID", shipmentID, "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		case "update_location":
			var payload struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}
			if err := json.Unmarshal(msg, &payload); err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": "payload inválido"})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.UpdateLocation(ctx, riderID, payload.Latitude, payload.Longitude); err != nil {
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": safeErrorMessage(err)})
				client.Send(errMsg)
			}

		default:
			slog.Warn("Tipo de mensaje rider desconocido", "type", base.Type)
		}
	}
}

// GetActiveTrips godoc
//
// @Summary      Viajes activos del rider
// @Description  Devuelve los viajes que el rider ya tiene asignados y siguen en curso (accepted, arriving o in_progress), sin códigos de subida/bajada. `fare_usd`/`fare_bs` son la ganancia del rider (90% de la tarifa), no lo que paga el pasajero; `collect_usd`/`collect_bs` son el total a cobrar en efectivo. Los Bs usan la tasa BCV congelada en la cotización del viaje. Cada viaje incluye `customer` con el nombre, el apellido y el teléfono del pasajero. En la oferta solo van nombre y apellido, y cuando el viaje ya finalizó se quita el teléfono. Sirve para recuperar el viaje al abrir la app sin depender del WebSocket. Lista vacía si no tiene ninguno.
// @Tags         Riders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Success      200  {object}  dtos.ActiveTripsResponse
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /riders/trips/active [get]
func (h *riderHandler) GetActiveTrips(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	trips, err := h.svc.ActiveTrips(ctx, session.UserID)
	if err != nil {
		return errcustom.ToHTTPError(err)
	}
	return response.Send(c, http.StatusOK, dtos.ActiveTripsResponse{Trips: trips})
}

// ReportLocation godoc
//
// @Summary      Reportar ubicación en background
// @Description  Equivalente REST de update_location, para cuando la app está en background y el WebSocket no está vivo. `mode` es obligatorio (`orders` o `trips`): sin conexión el backend no tiene de dónde inferirlo. Publica la misma presencia (Redis, TTL 45s) y última ubicación conocida (Postgres) que el WebSocket, y notifica al cliente si el rider tiene un viaje o pedido asignado en curso. No mete al rider en rondas de oferta nuevas: eso sigue exigiendo un socket conectado. Requiere cookie `encargalo-session`. Rol: Rider.
// @Tags         Riders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.ReportLocationRequest  true  "Ubicación y modo actual"
// @Success      204  "Sin contenido"
// @Failure      400  {string}  string  "Datos inválidos o coordenadas fuera del área de servicio"
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /riders/location [post]
func (h *riderHandler) ReportLocation(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var req dtos.ReportLocationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}

	if err := h.svc.ReportLocation(ctx, session.UserID, req.Latitude, req.Longitude, riderModels.RiderMode(req.Mode)); err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// SetAvailability godoc
//
// @Summary      Pausar o reanudar nuevas asignaciones
// @Description  El rider sigue conectado y conserva su orden o viaje activo, pero mientras accepting_new_work sea false deja de recibir nuevas órdenes (new_order) y ofertas de viaje (trip_offer). No afecta lo ya asignado ni ofertas que le hayan llegado antes de pausar.
// @Tags         Riders
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Param        body  body  dtos.SetAvailabilityRequest  true  "Estado de disponibilidad"
// @Success      204  "Sin contenido"
// @Failure      400  {string}  string  "Datos inválidos"
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /riders/availability [put]
func (h *riderHandler) SetAvailability(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	var req dtos.SetAvailabilityRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}
	if err := req.Validate(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payload inválido")
	}

	if err := h.svc.SetAvailability(ctx, session.UserID, req.AcceptingNewWork); err != nil {
		return errcustom.ToHTTPError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func safeErrorMessage(err error) string {
	if errors.Is(err, errcustom.ErrNotFound) {
		return "no encontrado"
	}
	if errors.Is(err, errcustom.ErrInvalidTransition) {
		return "esta transición no está disponible"
	}
	if errors.Is(err, errcustom.ErrInvalidCoordinates) {
		return "coordenadas inválidas"
	}
	if errors.Is(err, errcustom.ErrInvalidCode) {
		return "código de confirmación inválido"
	}
	if errors.Is(err, errcustom.ErrRiderBusy) {
		return "ya tienes un viaje en curso"
	}
	if errors.Is(err, errcustom.ErrVehicleCannotCarryOrder) {
		return "tu vehículo no puede llevar este pedido"
	}
	return "error interno"
}
