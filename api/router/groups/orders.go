package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/orders/handler"
	ridersHandler "Encargalo.app-api.go/internal/riders/handler"
	"github.com/labstack/echo/v4"
)

type OrdersGroup interface {
	Resource(g *echo.Echo)
}

type ordersGroup struct {
	middle               auth.AuthMiddleware
	handlerOrders        handler.OrderHandler
	hanlderManagerOrders handler.ManagerOrder
	handlerStaffReviews  handler.StaffReviews
	handlerRiders        ridersHandler.RiderHandler
	handlerFinance       ridersHandler.FinanceHandler
	handlerStaffRiders   ridersHandler.StaffRiderHandler
}

func NewOrdersGroup(
	middle auth.AuthMiddleware,
	handlerOrders handler.OrderHandler,
	hanlderManagerOrders handler.ManagerOrder,
	handlerStaffReviews handler.StaffReviews,
	handlerRiders ridersHandler.RiderHandler,
	handlerFinance ridersHandler.FinanceHandler,
	handlerStaffRiders ridersHandler.StaffRiderHandler,
) OrdersGroup {
	return &ordersGroup{middle, handlerOrders, hanlderManagerOrders, handlerStaffReviews, handlerRiders, handlerFinance, handlerStaffRiders}
}

func (o *ordersGroup) Resource(g *echo.Echo) {
	group := g.Group("/orders")

	group.POST("", o.handlerOrders.CreateOrder, o.middle.Auth, o.middle.Require(models.Customer))
	group.POST("/quote", o.handlerOrders.CreateQuote, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/history", o.handlerOrders.GetOrderHistory, o.middle.Auth, o.middle.Require(models.Customer))
	group.GET("/manager", o.hanlderManagerOrders.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Customer, models.OrderManager, models.Owner))
	group.GET("/staff/reviews", o.handlerStaffReviews.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Staff))
	group.GET("/rider", o.handlerRiders.HandleWebSocket, o.middle.Auth, o.middle.Require(models.Rider))
	group.POST("/:id/confirm-pickup", o.handlerOrders.ConfirmPickup, o.middle.Auth, o.middle.Require(models.OrderManager, models.Owner))
	group.POST("/:id/confirm-delivery", o.handlerOrders.ConfirmDelivery, o.middle.Auth, o.middle.Require(models.Rider))

	riders := g.Group("/riders", o.middle.Auth, o.middle.Require(models.Rider))
	riders.GET("/balance", o.handlerFinance.GetBalance)
	riders.POST("/withdrawal", o.handlerFinance.RequestWithdrawal)
	riders.GET("/transactions", o.handlerFinance.ListTransactions)

	staffRiders := g.Group("/riders/staff", o.middle.Auth, o.middle.Require(models.Staff))
	staffRiders.GET("/candidates", o.handlerStaffRiders.FindCandidate)
	staffRiders.GET("/profiles", o.handlerStaffRiders.ListProfiles)
	staffRiders.GET("/profiles/:userId", o.handlerStaffRiders.GetProfile)
	staffRiders.PUT("/profiles/:userId", o.handlerStaffRiders.UpsertProfile)
	staffRiders.DELETE("/profiles/:userId/photos/:field", o.handlerStaffRiders.DeletePhoto)
	staffRiders.POST("/profiles/:userId/activate", o.handlerStaffRiders.Activate)
	staffRiders.POST("/profiles/:userId/suspend", o.handlerStaffRiders.Suspend)
	staffRiders.POST("/profiles/:userId/touch-contact", o.handlerStaffRiders.TouchContact)
}
