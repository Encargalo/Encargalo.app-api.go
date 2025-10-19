package groups

import (
	"Encargalo.app-api.go/internal/auth/handler"
	"github.com/labstack/echo/v4"
)

type AuthGroup interface {
	Resource(g *echo.Echo)
}

type authGroup struct {
	authHand handler.Auth
}

func NewAuthGroup(authHand handler.Auth) AuthGroup {
	return &authGroup{authHand}
}

func (a *authGroup) Resource(g *echo.Echo) {
	group := g.Group("/sign-in")

	group.POST("/customers", a.authHand.SignInCustomer)
}
