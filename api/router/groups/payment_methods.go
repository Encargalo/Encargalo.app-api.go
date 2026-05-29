package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/payment_methods/handler"
	"github.com/labstack/echo/v4"
)

type PaymentMethodsGroup interface {
	Resource(g *echo.Echo)
}

type paymentMethodsGroup struct {
	handler    handler.PaymentMethods
	middleAuth auth.AuthMiddleware
}

func NewPaymentMethodsGroup(h handler.PaymentMethods, middleAuth auth.AuthMiddleware) PaymentMethodsGroup {
	return &paymentMethodsGroup{h, middleAuth}
}

func (p *paymentMethodsGroup) Resource(g *echo.Echo) {
	g.GET("/payment-methods", p.handler.GetPaymentMethods, p.middleAuth.Auth)
}
