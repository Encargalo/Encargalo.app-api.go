package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/auth/domain/models"
	"Encargalo.app-api.go/internal/users/handler"
	"github.com/labstack/echo/v4"
)

type UsersGroup interface {
	Resource(g *echo.Echo)
}

type usersGroup struct {
	middle         requestinfo.Request
	middleAuth     auth.AuthMiddleware
	handlerAddress handler.UsersAddressHandler
	handlerUsers   handler.UsersHandler
}

func NewUsersGroup(
	middle requestinfo.Request,
	middleAuth auth.AuthMiddleware,
	handlerAddress handler.UsersAddressHandler,
	handlerUsers handler.UsersHandler,
) UsersGroup {
	return &usersGroup{
		middle,
		middleAuth,
		handlerAddress,
		handlerUsers,
	}
}

func (o *usersGroup) Resource(g *echo.Echo) {

	group := g.Group("/users")

	group.POST("", o.handlerUsers.RegisterCustomer, o.middle.GetRequestInfo)

	group.GET("", o.handlerUsers.SearchCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.PUT("", o.handlerUsers.UpdateCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.PUT("/change-password", o.handlerUsers.UpdatePassword, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.POST("/address", o.handlerAddress.RegisterAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.GET("/address", o.handlerAddress.SearchAllAdrress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.DELETE("/address", o.handlerAddress.DeleteAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))

}
