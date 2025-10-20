package groups

import (
	"Encargalo.app-api.go/internal/auth/handler"
	customerauth "Encargalo.app-api.go/internal/shared/middleware/customerAuth"
	"github.com/labstack/echo/v4"
)

type AuthGroup interface {
	Resource(g *echo.Echo)
}

type authGroup struct {
	authHand   handler.Auth
	middleAuth customerauth.AuthMiddleware
}

func NewAuthGroup(authHand handler.Auth, middleAuth customerauth.AuthMiddleware) AuthGroup {
	return &authGroup{authHand, middleAuth}
}

func (a *authGroup) Resource(g *echo.Echo) {
	group := g.Group("/auth")

	group.POST("/sign-in/customers", a.authHand.SignInCustomer)
	group.DELETE("/logout", a.authHand.Logout, a.middleAuth.Auth)
}
