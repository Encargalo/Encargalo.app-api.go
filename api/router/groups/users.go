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
	middle             requestinfo.Request
	middleAuth         auth.AuthMiddleware
	handlerAddress     handler.UsersAddressHandler
	handlerUsers       handler.UsersHandler
	handlerFinance     handler.CustomerFinanceHandler
	handlerStaffRefund handler.StaffRefundHandler
}

func NewUsersGroup(
	middle requestinfo.Request,
	middleAuth auth.AuthMiddleware,
	handlerAddress handler.UsersAddressHandler,
	handlerUsers handler.UsersHandler,
	handlerFinance handler.CustomerFinanceHandler,
	handlerStaffRefund handler.StaffRefundHandler,
) UsersGroup {
	return &usersGroup{
		middle,
		middleAuth,
		handlerAddress,
		handlerUsers,
		handlerFinance,
		handlerStaffRefund,
	}
}

func (o *usersGroup) Resource(g *echo.Echo) {

	group := g.Group("/users")

	group.POST("", o.handlerUsers.RegisterCustomer, o.middle.GetRequestInfo)

	group.GET("", o.handlerUsers.SearchCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner, models.Staff))
	group.PUT("", o.handlerUsers.UpdateCustomer, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.PUT("/change-password", o.handlerUsers.UpdatePassword, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.DELETE("", o.handlerUsers.DeleteAccount, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.GET("/balance", o.handlerFinance.GetBalance, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.GET("/pago-movil", o.handlerFinance.GetPagoMovil, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.PUT("/pago-movil", o.handlerFinance.UpsertPagoMovil, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.POST("/balance/refund-requests", o.handlerFinance.CreateRefundRequest, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.GET("/balance/refund-requests", o.handlerFinance.ListRefundRequests, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.GET("/balance/refund-requests/:id", o.handlerFinance.GetRefundRequest, o.middleAuth.Auth, o.middleAuth.Require(models.Customer))
	group.GET("/staff/refund-requests", o.handlerStaffRefund.HandleWebSocket, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.POST("/staff/refund-requests/:id/complete", o.handlerStaffRefund.CompleteRefund, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.POST("/address", o.handlerAddress.RegisterAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.GET("/address", o.handlerAddress.SearchAllAdrress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.DELETE("/address", o.handlerAddress.DeleteAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))

}
