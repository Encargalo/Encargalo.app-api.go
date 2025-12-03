package groups

import (
	"Encargalo.app-api.go/internal/auth/handler"
	"Encargalo.app-api.go/internal/shared/middleware/auth"
	"github.com/labstack/echo/v4"
)

type AuthGroup interface {
	Resource(g *echo.Echo)
}

type authGroup struct {
	authHand   handler.Auth
	middleAuth auth.AuthMiddleware
}

func NewAuthGroup(authHand handler.Auth, middleAuth auth.AuthMiddleware) AuthGroup {
	return &authGroup{authHand, middleAuth}
}

func (a *authGroup) Resource(g *echo.Echo) {
	group := g.Group("/auth")

	group.POST("/sign-in/customers", a.authHand.SignInCustomer)
	group.POST("/sign-in/employee", a.authHand.SignInEmployee)
	group.DELETE("/logout", a.authHand.Logout, a.middleAuth.Auth)
}
