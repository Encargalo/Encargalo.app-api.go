package handler

import (
	"fmt"
	"net/http"

	"Encargalo.app-api.go/internal/orders/app"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type ManagerOrder interface {
	HandleWebSocket(c echo.Context) error
}

type managerOrder struct {
	svc app.AppManagerOrder
}

func New(svc app.AppManagerOrder) ManagerOrder {
	return &managerOrder{svc}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *managerOrder) HandleWebSocket(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		fmt.Println("Error al actualizar conexión:", err)
		return err
	}
	defer conn.Close()

	fmt.Println("Cliente conectado")

	shopID := uuid.New()

	if err := h.svc.Register(shopID, conn); err != nil {
		fmt.Println("Error registrando conexión:", err)
		conn.WriteMessage(websocket.TextMessage, []byte("error al registrar conexión"))
		return nil
	}
	defer h.svc.Unregister(shopID)

	conn.WriteMessage(websocket.TextMessage, []byte("registrado correctamente"))

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Cliente desconectado:", err)
			if shopID != uuid.Nil {
				h.svc.Unregister(shopID)
			}
			break
		}

		fmt.Println("Mensaje recibido:", string(msg))

		continue
	}

	return nil
}
