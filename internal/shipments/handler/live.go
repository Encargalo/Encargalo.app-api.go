package handler

import (
	"log/slog"

	authModels "Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/ws"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// Live godoc
// @Summary      WebSocket del cliente para seguir sus envíos
// @Description  El cliente recibe el estado del envío sin volver a consultar. Al conectar se envían los envíos activos.
//
// ### Mensajes salientes
//   - `{"type":"connected"}`
//   - `{"type":"shipment_update","shipment":{...}}` — incluye pickup_code y dropoff_code siempre, y rider público si ya hay conductor. En cancelled y no_driver trae cancel_reason y, si hubo revisión de pago, el bloque payment
//
// @Tags         Shipments
// @Security     SessionCookie
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /shipments/live [get]
func (h *shipmentsHandler) Live(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	cfg := config.Get()
	upgrader := ws.BuildUpgrader(cfg.IsProduction(), cfg.Server.AllowedOrigins)
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading websocket for customer shipment", "error", err)
		return err
	}

	client := ws.NewClient(conn)
	go client.WritePump()
	client.Send([]byte(`{"type":"connected"}`))

	if h.live != nil {
		if err := h.live.Register(ctx, session.UserID, client); err != nil {
			slog.Error("Error registrando canal de envío", "customerID", session.UserID, "error", err)
			client.Close()
			_ = conn.Close()
			return nil
		}
	}

	defer func() {
		if h.live != nil {
			h.live.Unregister(session.UserID, client)
		}
		client.Close()
		_ = conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Info("WebSocket de envío cerrado", "customerID", session.UserID)
			} else {
				slog.Error("Error leyendo websocket de envío", "error", err)
			}
			return nil
		}
	}
}
