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
// @Summary      WebSocket del cliente para seguir sus viajes
// @Description  El cliente recibe el estado del viaje y la ubicación del rider sin volver a consultar. Al conectar se envían los viajes activos.
//
// ### Mensajes salientes
//   - `{"type":"connected"}`
//   - `{"type":"trip_update","trip":{...}}` — incluye pickup_code o dropoff_code cuando el cliente debe mostrarlos, y rider público si ya hay conductor
//   - `{"type":"rider_location","trip_id":"<uuid>","latitude":0,"longitude":0}` — posición del rider asignado
//
// @Tags         Trips
// @Security     SessionCookie
// @Failure      401  {string}  string  "No autorizado"
// @Failure      403  {string}  string  "Acceso denegado"
// @Router       /trips/live [get]
func (h *tripsHandler) Live(c echo.Context) error {
	ctx := c.Request().Context()
	session := ctx.Value(authModels.SessionKey).(authModels.ActiveSession)

	cfg := config.Get()
	upgrader := ws.BuildUpgrader(cfg.IsProduction(), cfg.Server.AllowedOrigins)
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("Error upgrading websocket for customer trip", "error", err)
		return err
	}

	client := ws.NewClient(conn)
	go client.WritePump()
	client.Send([]byte(`{"type":"connected"}`))

	if h.live != nil {
		if err := h.live.Register(ctx, session.UserID, client); err != nil {
			slog.Error("Error registrando canal de viaje", "customerID", session.UserID, "error", err)
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
				slog.Info("WebSocket de viaje cerrado", "customerID", session.UserID)
			} else {
				slog.Error("Error leyendo websocket de viaje", "error", err)
			}
			return nil
		}
	}
}
