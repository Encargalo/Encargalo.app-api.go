package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	modelsOrder "Encargalo.app-api.go/internal/orders/domain/models"
	portsOrder "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"Encargalo.app-api.go/internal/shared/ws"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type ManagerOrder interface {
	HandleWebSocket(c echo.Context) error
}

type managerOrder struct {
	svc      portsOrder.AppManagerOrder
	svc2     ports.EmployeeApp
	svcOrder portsOrder.OrdersApp
	upgrader websocket.Upgrader
}

func New(
	svc portsOrder.AppManagerOrder,
	svc2 ports.EmployeeApp,
	svcOrder portsOrder.OrdersApp,
) ManagerOrder {
	cfg := config.Get()
	return &managerOrder{
		svc:      svc,
		svc2:     svc2,
		svcOrder: svcOrder,
		upgrader: ws.BuildUpgrader(cfg.IsProduction(), cfg.Server.AllowedOrigins),
	}
}

// HandleWebSocket godoc
//
// @Summary WebSocket de gestión de órdenes
// @Description
// Establece una conexión WebSocket para la gestión en tiempo real de órdenes.
//
// ### Autenticación
// Requiere una sesión válida inyectada en el contexto (middleware de auth).
//
// ### Comportamiento
// - Si el usuario es **OrderManager**, la conexión se asocia al `shop_id` del empleado.
// - Si es otro tipo de usuario, se asocia directamente al `user_id`.
// - Al conectarse, el servidor envía:
// ```json
// { "type": "connected" }
// ```
//
// ### Mensajes entrantes soportados
// **update_status**
// ```json
//
//	{
//	  "type": "update_status",
//	  "order_id": "uuid",
//	  "new_status": "string"
//	}
//
// ```
// Un OrderManager/Owner solo puede actualizar órdenes de su propio shop_id
// (validado server-side); intentarlo sobre una orden de otra tienda responde
// `{"type":"error"}` sin aplicar el cambio.
//
// ### Mensajes salientes
// - Eventos de órdenes actualizadas (broadcast).
//
// @Tags         Orders WebSocket
// @Accept       json
// @Produce      json
// @Security     SessionCookie
// @Failure      401  {string}  string  "Sesión no encontrada o inválida"
// @Failure      403  {string}  string  "Acceso denegado"
// @Failure      500  {string}  string  "Error interno del servidor"
// @Router       /orders/manager [get]
func (h *managerOrder) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	session := ctx.Value(models.SessionKey).(models.ActiveSession)

	var connectionID uuid.UUID
	isShop := session.HasRole(models.OrderManager) || session.HasRole(models.Owner)

	if isShop {
		employee, err := h.svc2.SearchEmployeeByID(ctx, session.UserID)
		if err != nil {
			slog.Error("Error buscando empleado", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
		}
		connectionID = employee.ShopID
	} else {
		connectionID = session.UserID
	}

	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading websocket", "error", err)
		return err
	}

	client := ws.NewClient(conn)
	go client.WritePump()

	if err := h.svc.Register(ctx, connectionID, client, isShop); err != nil {
		_ = conn.Close()
		return err
	}

	slog.Info("manager WebSocket connected", "connectionID", connectionID, "isShop", isShop)

	defer func() {
		slog.Info("manager WebSocket disconnected", "connectionID", connectionID, "isShop", isShop)
		h.svc.Unregister(connectionID, client, isShop)
		_ = conn.Close()
	}()

	client.Send([]byte(`{"type":"connected"}`))

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Info("WebSocket cerrado por el cliente", "id", connectionID)
			} else {
				slog.Error("Error leyendo websocket", "error", err)
			}
			return nil
		}

		var base struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(msg, &base); err != nil {
			slog.Warn("Mensaje inválido", "error", err)
			continue
		}

		switch base.Type {

		case "update_status":
			var payload modelsOrder.UpdateStatusOrder
			if err := json.Unmarshal(msg, &payload); err != nil {
				slog.Warn("Payload inválido update_status", "error", err)
				client.Send([]byte(`{"type":"error","message":"payload inválido"}`))
				continue
			}

			order, err := h.svcOrder.UpdateOrderStatus(
				ctx,
				payload.OrderID,
				payload.NewStatus,
				session.UserID,
				isShop,
				connectionID,
			)
			if err != nil {
				slog.Error("Error actualizando orden", "error", err)
				errMsg, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
				client.Send(errMsg)
				continue
			}

			if err := h.svc.BroadcastOrder(order); err != nil {
				slog.Error("Error broadcasting order", "error", err)
			}

		default:
		}
	}
}
