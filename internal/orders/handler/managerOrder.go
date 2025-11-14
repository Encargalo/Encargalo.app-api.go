package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/employee/domain/ports"
	"Encargalo.app-api.go/internal/orders/app"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/google/uuid"
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
	svc  app.AppManagerOrder
	svc2 ports.EmployeeApp
}

func New(svc app.AppManagerOrder, svc2 ports.EmployeeApp) ManagerOrder {
	return &managerOrder{svc, svc2}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *managerOrder) HandleWebSocket(c echo.Context) error {

	ctx := c.Request().Context()

	session, ok := ctx.Value("session").(models.ActiveSession)
	if !ok {
		slog.Error("No se encontró el usuario en el contexto")
		return errcustom.ErrUnexpectedError
	}

	if session.UserType != roleEmployee {
		slog.Error("Usuario no autorizado para acceder a esta ruta")
		return echo.NewHTTPError(http.StatusForbidden, "usuario no autorizado")
	}

	employee, err := h.svc2.SearchEmployeeByID(ctx, session.UserID)
	if err != nil {
		slog.Error("Error al buscar el empleado:", slog.String("error", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, errcustom.ErrUnexpectedError)
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		fmt.Println("Error al actualizar conexión:", err)
		return err
	}
	defer conn.Close()

	if err := h.svc.Register(ctx, employee.ShopID, conn); err != nil {
		fmt.Println("Error registrando conexión:", err)
		conn.WriteMessage(websocket.TextMessage, []byte("error al registrar conexión"))
		return nil
	}
	defer h.svc.Unregister(employee.ShopID)

	conn.WriteMessage(websocket.TextMessage, []byte("registrado correctamente"))

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if employee.ID != uuid.Nil {
				h.svc.Unregister(employee.ShopID)
			}
			break
		}
		continue
	}

	return nil
}
