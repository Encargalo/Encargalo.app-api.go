package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/handler"
	"github.com/labstack/echo/v4"
)

type OrdersGroup interface {
	Resource(g *echo.Echo)
}

type ordersGroup struct {
	middle               auth.AuthMiddleware
	handlerOrders        handler.OrderHandler
	hanlderManagerOrders handler.ManagerOrder
}

func NewOrdersGroup(middle auth.AuthMiddleware, handlerOrders handler.OrderHandler, hanlderManagerOrders handler.ManagerOrder) OrdersGroup {
	return &ordersGroup{middle, handlerOrders, hanlderManagerOrders}
}

func (o *ordersGroup) Resource(g *echo.Echo) {
	group := g.Group("/orders")

	group.POST("", o.handlerOrders.CreateOrder, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/manager", o.hanlderManagerOrders.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Customer, models.OrderManager))
}
