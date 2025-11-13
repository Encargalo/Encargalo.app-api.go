package groups

import (
	"Encargalo.app-api.go/internal/orders/handler"
	customerauth "Encargalo.app-api.go/internal/shared/middleware/customerAuth"
	employeeauth "Encargalo.app-api.go/internal/shared/middleware/employeeAuth"
	"github.com/labstack/echo/v4"
)

type OrdersGroup interface {
	Resource(g *echo.Echo)
}

type ordersGroup struct {
	middle               customerauth.AuthMiddleware
	midleEmployee        employeeauth.AuthMiddleware
	handlerOrders        handler.OrderHandler
	hanlderManagerOrders handler.ManagerOrder
}

func NewOrdersGroup(middle customerauth.AuthMiddleware, midleEmployee employeeauth.AuthMiddleware, handlerOrders handler.OrderHandler, hanlderManagerOrders handler.ManagerOrder) OrdersGroup {
	return &ordersGroup{middle, midleEmployee, handlerOrders, hanlderManagerOrders}
}

func (o *ordersGroup) Resource(g *echo.Echo) {
	group := g.Group("/orders")
	group.POST("", o.handlerOrders.CreateOrder, o.middle.Auth)
	group.GET("/ws", o.hanlderManagerOrders.HandleWebSocket, o.midleEmployee.Auth)
}
