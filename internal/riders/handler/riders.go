package handler

import (
	"encoding/json"
	"errors"
	"log/slog"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	riderModels "Encargalo.app-api.go/internal/riders/domain/models"
	ridersPorts "Encargalo.app-api.go/internal/riders/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type RiderHandler interface {
	HandleWebSocket(c echo.Context) error
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
// → Intenta asignar la orden al rider. Primer rider que acepta gana.
//
// **update_location**
//
//	{"type":"update_location","latitude":10.48,"longitude":-66.90}
//
// → Guarda la ubicación en memoria (solo relevante en modo trips).
//
// **accept_trip**
//
//	{"type":"accept_trip","trip_id":"<uuid>"}
//
// → Intenta asignar el viaje al rider. Primer rider que acepta gana.
//
// **advance_trip**
//
//	{"type":"advance_trip","trip_id":"<uuid>"}
//
// → Solo el rider asignado. Mueve el trip al siguiente estado
// (accepted→arriving→in_progress→completed). Al llegar a completed calcula
// la tarifa (distancia vial pickup→dropoff) y acredita la ganancia del rider.
//
// **cancel_trip**
//
//	{"type":"cancel_trip","trip_id":"<uuid>"}
//
// → Solo el rider asignado, y solo en accepted o arriving. Libera el viaje
// (searching, sin rider) y lo vuelve a ofertar a otros riders cercanos.
// El rider que canceló no vuelve a recibir esa oferta.
//
// ### Mensajes salientes
//   - `{"type":"connected"}` — confirmación de conexión
//   - `{"type":"new_order","order":{...}}` — nueva orden disponible (sin códigos)
//   - `{"type":"order_accepted","order":{...}}` — asignación exitosa (con pickup_code)
//   - `{"type":"order_taken","order_id":"<uuid>"}` — otro rider se adelantó
//   - `{"type":"order_update","order":{...}}` — actualización de orden asignada (con pickup_code)
//   - `{"type":"trip_offer","trip":{...}}` — viaje ofrecido a hasta 3 riders cercanos. Si nadie acepta en 2 min, otra ronda ofrece a otros 3 con un anillo H3 más de radio, hasta 4 rondas o hasta agotar candidatos. Después el viaje pasa a no_driver.
//   - `{"type":"trip_accepted","trip":{...}}` — asignación de viaje exitosa
//   - `{"type":"trip_taken","trip_id":"<uuid>"}` — otro rider se adelantó
//   - `{"type":"trip_status","trip":{...}}` — trip avanzó de estado (respuesta a advance_trip) o quedó liberado (respuesta a cancel_trip)
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

			if err := h.svc.AdvanceTrip(ctx, riderID, tripID, client); err != nil {
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
	return "error interno"
}
