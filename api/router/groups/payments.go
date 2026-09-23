package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/payments/handler"
	"github.com/labstack/echo/v4"
)

type PaymentsGroup interface {
	Resource(g *echo.Echo)
}

type paymentsGroup struct {
	middle              auth.AuthMiddleware
	handlerStaffReviews handler.StaffReviews
}

func NewPaymentsGroup(middle auth.AuthMiddleware, handlerStaffReviews handler.StaffReviews) PaymentsGroup {
	return &paymentsGroup{middle: middle, handlerStaffReviews: handlerStaffReviews}
}

func (p *paymentsGroup) Resource(g *echo.Echo) {
	group := g.Group("/payments")
	group.GET("/staff/reviews", p.handlerStaffReviews.HandleWebSocket, p.middle.Auth, p.middle.Require(models.Staff))
}
