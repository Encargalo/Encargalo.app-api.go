package groups

import (
	"Encargalo.app-api.go/internal/orders/handler"
	"Encargalo.app-api.go/internal/shared/middleware/auth"
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
	groupCustomer := g.Group("customers/orders")
	groupBusiness := g.Group("employee/orders")

	groupCustomer.POST("", o.handlerOrders.CreateOrder, o.middle.Auth)
	groupCustomer.GET("/manager", o.hanlderManagerOrders.HandleWebSocket, o.middle.Auth)

	groupBusiness.GET("/manager", o.hanlderManagerOrders.HandleWebSocket, o.middle.Auth)
}
