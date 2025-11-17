package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	portsOrder "Encargalo.app-api.go/internal/orders/domain/ports"

	"Encargalo.app-api.go/internal/orders/app"
	modelsOrder "Encargalo.app-api.go/internal/orders/domain/models"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

const (
	roleEmployee = "Order Manager"
)

type ManagerOrder interface {
	HandleWebSocket(c echo.Context) error
}

type managerOrder struct {
	svc      app.AppManagerOrder
	svc2     ports.EmployeeApp
	svcOrder portsOrder.OrdersApp
}

func New(svc app.AppManagerOrder, svc2 ports.EmployeeApp, svcOrder portsOrder.OrdersApp) ManagerOrder {
	return &managerOrder{svc, svc2, svcOrder}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *managerOrder) HandleWebSocket(c echo.Context) error {
	ctx := c.Request().Context()

	// Obtener sesión desde el contexto
	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	// Validar tipo de usuario
	if session.UserType != roleEmployee {
		slog.Error("Usuario no autorizado para acceder a esta ruta")
		return echo.NewHTTPError(http.StatusForbidden, "usuario no autorizado")
	}

	// Obtener empleado por ID
	employee, err := h.svc2.SearchEmployeeByID(ctx, session.UserID)
	if err != nil {
		slog.Error("Error al buscar empleado:", slog.String("error", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	// Actualizar HTTP → WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error al actualizar conexión websocket:", slog.String("error", err.Error()))
		return err
	}

	// Asegurar cierre del WebSocket
	defer func() {
		conn.Close()
		h.svc.Unregister(employee.ShopID)
		slog.Info("WebSocket desconectado y limpiado", "shopID", employee.ShopID)
	}()

	// Registrar conexión del local
	if err := h.svc.Register(ctx, employee.ShopID, conn); err != nil {
		slog.Error("Error registrando conexión:", slog.String("error", err.Error()))
		conn.WriteMessage(websocket.TextMessage, []byte("error al registrar conexión"))
		return nil
	}

	conn.WriteMessage(websocket.TextMessage, []byte("registrado correctamente"))

	// LOOP PRINCIPAL DE LECTURA
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {

			// Detecta cierre normal o cierre inesperado
			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Warn("Cliente cerró la conexión websocket", "shopID", employee.ShopID)
			} else {
				slog.Error("Error leyendo mensaje websocket:", slog.String("error", err.Error()))
			}

			return nil // se ejecutará el defer limpiando todo
		}

		// Parsear mensaje base
		var base map[string]any
		if err := json.Unmarshal(msg, &base); err != nil {
			conn.WriteMessage(msgType, []byte("Error: formato inválido"))
			continue
		}

		msgTypeStr, _ := base["type"].(string)

		switch msgTypeStr {

		case "update_status":
			var payload modelsOrder.UpdateStatusOrder
			if err := json.Unmarshal(msg, &payload); err != nil {
				slog.Error("Error parseando update_status:", "error", err.Error())
				conn.WriteMessage(msgType, []byte("Error: formato inválido en update_status"))
				continue
			}

			order, err := h.svcOrder.UpdateOrderStatus(ctx, payload.OrderID, payload.NewStatus)
			if err != nil {
				slog.Error("Error actualizando estado de orden:", "error", err.Error())
				conn.WriteMessage(msgType, []byte("Error: error al actualizar estado de orden"))
				continue
			}

			conn.WriteMessage(msgType, []byte("Estado de orden actualizado correctamente"))

			h.svc.BroadcastOrder(order)

			continue

		default:
			conn.WriteMessage(msgType, []byte("Error: tipo de evento no reconocido"))
			continue
		}
	}
}
