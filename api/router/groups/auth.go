package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	handAuth "Encargalo.app-api.go/internal/auth/handler"
	"github.com/labstack/echo/v4"
)

type AuthGroup interface {
	Resource(g *echo.Echo)
}

type authGroup struct {
	middleAuth auth.AuthMiddleware
	handAuth   handAuth.Auth
}

func NewAuthGroup(middleAuth auth.AuthMiddleware, handAuth handAuth.Auth) AuthGroup {
	return &authGroup{middleAuth, handAuth}
}

func (a *authGroup) Resource(g *echo.Echo) {
	group := g.Group("/auth")

	group.POST("/sign-in/users", a.handAuth.SignInCustomer)
	group.POST("/sign-in/employees", a.handAuth.SignInEmployee)
	group.POST("/sign-in/riders", a.handAuth.SignInRider)
	group.DELETE("/logout", a.handAuth.Logout, a.middleAuth.Auth, a.middleAuth.Require(models.Customer, models.OrderManager, models.Owner, models.Rider))
}
