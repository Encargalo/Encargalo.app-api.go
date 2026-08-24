package handler

import (
	"encoding/json"
	"errors"
	"log/slog"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
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
// ### Mensajes salientes
//   - `{"type":"connected"}` — confirmación de conexión
//   - `{"type":"new_order","order":{...}}` — nueva orden disponible (sin códigos)
//   - `{"type":"order_accepted","order":{...}}` — asignación exitosa (con pickup_code)
//   - `{"type":"order_taken","order_id":"<uuid>"}` — otro rider se adelantó
//   - `{"type":"order_update","order":{...}}` — actualización de orden asignada (con pickup_code)
//   - `{"type":"error","message":"..."}` — error procesando mensaje
//
// @Tags         Riders WebSocket
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /orders/rider [get]
func (h *riderHandler) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	riderID := session.UserID

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading websocket for rider", "error", err)
		return err
	}

	client := ws.NewClient(conn)
	go client.WritePump()

	if err := h.svc.Register(ctx, riderID, client); err != nil {
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

		default:
			slog.Warn("Tipo de mensaje rider desconocido", "type", base.Type)
		}
	}
}

func safeErrorMessage(err error) string {
	if errors.Is(err, errcustom.ErrNotFound) {
		return "orden no encontrada"
	}
	if errors.Is(err, errcustom.ErrInvalidTransition) {
		return "la orden no está disponible para aceptar"
	}
	return "error interno"
}
