package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/auth/domain/models"
	handAuth "Encargalo.app-api.go/internal/auth/handler"
	"Encargalo.app-api.go/internal/employee/handler"
	"github.com/labstack/echo/v4"
)

type EmployeeGroup interface {
	Resource(g *echo.Echo)
}

type employeeGroup struct {
	hand       handler.EmployyeHandler
	handAuth   handAuth.Auth
	middle     requestinfo.Request
	middleAuth auth.AuthMiddleware
}

func NewEmployeeGroup(hand handler.EmployyeHandler, handAuth handAuth.Auth, middle requestinfo.Request, middleAuth auth.AuthMiddleware) EmployeeGroup {
	return &employeeGroup{hand, handAuth, middle, middleAuth}
}

func (e *employeeGroup) Resource(g *echo.Echo) {
	group := g.Group("/employee")

	group.POST("/sign-in", e.handAuth.SignInEmployee)
	group.DELETE("/logout", e.handAuth.LogoutEmployee, e.middle.GetRequestInfo, e.middleAuth.Auth)

	group.GET("", e.hand.SearchEmployee, e.middle.GetRequestInfo, e.middleAuth.Auth, e.middleAuth.Require(models.OrderManager))
}
