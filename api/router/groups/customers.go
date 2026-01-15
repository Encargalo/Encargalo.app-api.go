package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/auth/domain/models"
	handAuth "Encargalo.app-api.go/internal/auth/handler"
	"Encargalo.app-api.go/internal/customers/handler"
	"github.com/labstack/echo/v4"
)

type CustomersGroup interface {
	Resource(g *echo.Echo)
}

type customersGroup struct {
	middle           requestinfo.Request
	middleAuth       auth.AuthMiddleware
	handlerAddress   handler.CustomersAddressHandler
	handlerCustomers handler.CustomersHandler
	handAuth         handAuth.Auth
}

func NewCustomersGroup(
	middle requestinfo.Request,
	middleAuth auth.AuthMiddleware,
	handlerAddress handler.CustomersAddressHandler,
	handlerCustomers handler.CustomersHandler,
	handAuth handAuth.Auth,
) CustomersGroup {
	return &customersGroup{
		middle,
		middleAuth,
		handlerAddress,
		handlerCustomers,
		handAuth}
}

func (o *customersGroup) Resource(g *echo.Echo) {

	group := g.Group("/customers")

	group.POST("/sign-in", o.handAuth.SignInCustomer)
	group.DELETE("/logout", o.handAuth.Logout, o.middle.GetRequestInfo, o.middleAuth.Auth)
	group.POST("", o.handlerCustomers.RegisterCustomer, o.middle.GetRequestInfo)

	group.GET("", o.handlerCustomers.SearchCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.PUT("", o.handlerCustomers.UpdateCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.PUT("/change-password", o.handlerCustomers.UpdatePassword, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.POST("/address", o.handlerAddress.RegisterAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.GET("/address", o.handlerAddress.SearchAllAdrress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.DELETE("/address", o.handlerAddress.DeleteAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))

}
