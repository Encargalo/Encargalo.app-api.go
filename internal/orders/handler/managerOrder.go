package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	portsOrder "Encargalo.app-api.go/internal/orders/domain/ports"
	"Encargalo.app-api.go/internal/shared/errcustom"

	"Encargalo.app-api.go/internal/orders/app"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	modelsUpdateOrders "Encargalo.app-api.go/internal/orders/domain/models"
)

type ManagerOrder interface {
	HandleWebSocket(c echo.Context) error
}

type managerOrder struct {
	svc      app.AppManagerOrder
	svc2     ports.EmployeeApp
	svcOrder portsOrder.OrdersApp
}

func New(
	svc app.AppManagerOrder,
	svc2 ports.EmployeeApp,
	svcOrder portsOrder.OrdersApp,
) ManagerOrder {
	return &managerOrder{svc, svc2, svcOrder}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *managerOrder) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró la sesión en el contexto")
		return errcustom.ErrUnexpectedError
	}

	var connectionID uuid.UUID
	isShop := session.UserType == models.OrderManager

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

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading websocket", "error", err)
		return err
	}

	client, err := h.svc.Register(ctx, connectionID, conn, isShop)
	if err != nil {
		conn.Close()
		return err
	}

	defer func() {
		h.svc.Unregister(connectionID, client, isShop)
		conn.Close()
	}()

	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

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
			var payload modelsUpdateOrders.UpdateStatusOrder
			if err := json.Unmarshal(msg, &payload); err != nil {
				slog.Warn("Payload inválido update_status", "error", err)
				continue
			}

			order, err := h.svcOrder.UpdateOrderStatus(
				ctx,
				payload.OrderID,
				payload.NewStatus,
			)
			if err != nil {
				slog.Error("Error actualizando orden", "error", err)
				continue
			}

			h.svc.BroadcastOrder(order)

		default:
		}
	}
}
