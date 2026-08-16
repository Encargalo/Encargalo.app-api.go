package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/auth/domain/models"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"Encargalo.app-api.go/internal/users/handler"
	"github.com/labstack/echo/v4"
)

const (
	staffUsersCandidatesRateLimitCount  = 30
	staffUsersCandidatesRateLimitWindow = time.Minute
)

type UsersGroup interface {
	Resource(g *echo.Echo)
}

type usersGroup struct {
	middle             requestinfo.Request
	middleAuth         auth.AuthMiddleware
	rateLimiter        sharedmw.RateLimiter
	handlerAddress     handler.UsersAddressHandler
	handlerUsers       handler.UsersHandler
	handlerFinance     handler.CustomerFinanceHandler
	handlerStaffRefund handler.StaffRefundHandler
	handlerStaffUsers  handler.StaffUsersHandler
}

func NewUsersGroup(
	middle requestinfo.Request,
	middleAuth auth.AuthMiddleware,
	rateLimiter sharedmw.RateLimiter,
	handlerAddress handler.UsersAddressHandler,
	handlerUsers handler.UsersHandler,
	handlerFinance handler.CustomerFinanceHandler,
	handlerStaffRefund handler.StaffRefundHandler,
	handlerStaffUsers handler.StaffUsersHandler,
) UsersGroup {
	return &usersGroup{
		middle,
		middleAuth,
		rateLimiter,
		handlerAddress,
		handlerUsers,
		handlerFinance,
		handlerStaffRefund,
		handlerStaffUsers,
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
	group.GET("/staff/candidates", o.handlerStaffUsers.FindCandidate,
		o.middle.GetRequestInfo,
		o.middleAuth.Auth,
		o.middleAuth.Require(models.Staff),
		o.rateLimiter.LimitByIP("users:staff:candidates", staffUsersCandidatesRateLimitCount, staffUsersCandidatesRateLimitWindow),
	)
	group.GET("/staff/refund-requests", o.handlerStaffRefund.HandleWebSocket, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.POST("/staff/refund-requests/:id/complete", o.handlerStaffRefund.CompleteRefund, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.GET("/staff", o.handlerStaffUsers.ListUsers, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.GET("/staff/:userId", o.handlerStaffUsers.GetUser, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.PATCH("/staff/:userId", o.handlerStaffUsers.UpdateUser, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.POST("/staff/:userId/block", o.handlerStaffUsers.BlockUser, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.POST("/staff/:userId/unblock", o.handlerStaffUsers.UnblockUser, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Staff))
	group.POST("/address", o.handlerAddress.RegisterAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.GET("/address", o.handlerAddress.SearchAllAdrress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))
	group.DELETE("/address", o.handlerAddress.DeleteAddress, o.middle.GetRequestInfo, o.middleAuth.Auth, o.middleAuth.Require(models.Customer, models.OrderManager, models.Owner))

}
