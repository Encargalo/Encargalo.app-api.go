package groups

import (
	"Encargalo.app-api.go/internal/customers/handler"
	customerauth "Encargalo.app-api.go/internal/shared/middleware/customerAuth"
	requestinfo "Encargalo.app-api.go/internal/shared/middleware/requestInfo"
	"github.com/labstack/echo/v4"
)

type CustomersGroup interface {
	Resource(g *echo.Echo)
}

type customersGroup struct {
	middle           requestinfo.Request
	middleAuth       customerauth.AuthMiddleware
	handlerAddress   handler.CustomersAddressHandler
	handlerCustomers handler.CustomersHandler
}

func NewCustomersGroup(
	middle requestinfo.Request,
	middleAuth customerauth.AuthMiddleware,
	handlerAddress handler.CustomersAddressHandler,
	handlerCustomers handler.CustomersHandler,
) CustomersGroup {
	return &customersGroup{
		middle,
		middleAuth,
		handlerAddress,
		handlerCustomers}
}

func (o *customersGroup) Resource(g *echo.Echo) {

	group := g.Group("/customers")

	group.POST("", o.handlerCustomers.RegisterCustomer, o.middle.GetRequestInfo)
	group.GET("", o.handlerCustomers.SearchCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth)
	group.PUT("", o.handlerCustomers.UpdateCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth)
	group.PUT("/change-password", o.handlerCustomers.UpdatePassword, o.middle.GetRequestInfo, o.middleAuth.Auth)
	group.POST("/address", o.handlerAddress.RegisterAddress, o.middle.GetRequestInfo, o.middleAuth.Auth)
	group.GET("/address", o.handlerAddress.SearchAllAdrress, o.middle.GetRequestInfo, o.middleAuth.Auth)
	group.DELETE("/address/:address", o.handlerAddress.DeleteAddress, o.middle.GetRequestInfo, o.middleAuth.Auth)

}
