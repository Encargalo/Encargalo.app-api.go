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

const (
	roleEmployee = "Order Manager"
	roleCustomer = "customer"
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

	// 1. Obtener sesión y validar autenticación
	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if session.UserType != roleEmployee && session.UserType != roleCustomer {
		slog.Error("Usuario no autorizado para acceder a esta ruta")
		return echo.NewHTTPError(http.StatusForbidden, "usuario no autorizado")
	}

	// 2. Determinar ID de conexión y Rol
	var conectionID uuid.UUID
	isShop := session.UserType == roleEmployee

	if isShop {
		employee, err := h.svc2.SearchEmployeeByID(ctx, session.UserID)
		if err != nil {
			slog.Error("Error al buscar empleado:", slog.String("error", err.Error()))
			return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
		}
		conectionID = employee.ShopID
	} else {
		conectionID = session.UserID
	}

	// 3. Upgrade HTTP a WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error al actualizar conexión websocket:", slog.String("error", err.Error()))
		return err
	}

	// 4. Defer para limpieza: Se ejecuta cuando se rompe el bucle for
	defer func() {
		// IMPORTANTE: Pasamos 'conn' para borrar SOLO esta conexión específica y no todas las del usuario
		h.svc.Unregister(conectionID, conn, isShop)
		conn.Close()
		slog.Info("WebSocket desconectado y limpiado", "ID", conectionID)
	}()

	// 5. Registrar conexión en el servicio (Esto inicia el WritePump/Ping en el servicio)
	if err := h.svc.Register(ctx, conectionID, conn, isShop); err != nil {
		slog.Error("Error registrando conexión:", slog.String("error", err.Error()))
		conn.WriteMessage(websocket.TextMessage, []byte("error al registrar conexión"))
		return nil
	}

	// -----------------------------------------------------------------------
	// CONFIGURACIÓN CRÍTICA PARA EVITAR DESCONEXIONES (Read Pump Config)
	// -----------------------------------------------------------------------

	// Limite de tamaño para evitar ataques de memoria
	conn.SetReadLimit(4096)

	// Establecemos el primer deadline de lectura
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Handler para PONG: Cuando el navegador responde al Ping del servidor,
	// extendemos el deadline 60 segundos más.
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Mensaje inicial de éxito
	conn.WriteMessage(websocket.TextMessage, []byte("registrado correctamente"))

	// 6. Bucle de Lectura (Read Loop)
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Warn("Cliente cerró la conexión websocket", "ID", conectionID)
			} else {
				slog.Error("Error leyendo mensaje websocket:", slog.String("error", err.Error()))
			}
			// Al salir del bucle (break o return), se ejecuta el defer y limpia la conexión
			return nil
		}

		// Parsear mensaje base
		var base map[string]any
		if err := json.Unmarshal(msg, &base); err != nil {
			slog.Error("Error parseando JSON base", "error", err)
			// No desconectamos, solo avisamos y seguimos escuchando
			conn.WriteMessage(msgType, []byte("Error: formato JSON inválido"))
			continue
		}

		msgTypeStr, _ := base["type"].(string)

		switch msgTypeStr {

		case "update_status":
			var payload modelsUpdateOrders.UpdateStatusOrder
			if err := json.Unmarshal(msg, &payload); err != nil {
				slog.Error("Error parseando payload update_status:", "error", err.Error())
				conn.WriteMessage(msgType, []byte("Error: formato inválido en update_status"))
				continue
			}

			// Lógica de negocio
			order, err := h.svcOrder.UpdateOrderStatus(ctx, payload.OrderID, payload.NewStatus)
			if err != nil {
				slog.Error("Error actualizando estado de orden:", "error", err.Error())
				conn.WriteMessage(msgType, []byte("Error: error al actualizar estado de orden"))
				continue
			}

			// Confirmación al emisor
			conn.WriteMessage(msgType, []byte("Estado de orden actualizado correctamente"))

			// Broadcast a todos los interesados (tienda y cliente)
			h.svc.BroadcastOrder(order)

		default:
			// Log opcional para debug, pero evita saturar
			// slog.Debug("Mensaje recibido tipo desconocido", "type", msgTypeStr)
		}
	}
}
