package groups

import (
	"Encargalo.app-api.go/internal/employee/handler"
	"Encargalo.app-api.go/internal/shared/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/shared/middleware/requestInfo"
	"github.com/labstack/echo/v4"
)

type EmployeeGroup interface {
	Resource(g *echo.Echo)
}

type employeeGroup struct {
	hand       handler.EmployyeHandler
	middle     requestinfo.Request
	middleAuth auth.AuthMiddleware
}

func NewEmployeeGroup(hand handler.EmployyeHandler, middle requestinfo.Request, middleAuth auth.AuthMiddleware) EmployeeGroup {
	return &employeeGroup{hand, middle, middleAuth}
}

func (e *employeeGroup) Resource(g *echo.Echo) {
	group := g.Group("/employee")

	group.GET("", e.hand.SearchEmployee, e.middle.GetRequestInfo, e.middleAuth.Auth)
}
