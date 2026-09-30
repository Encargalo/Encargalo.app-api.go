package groups

import (
	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	ridersHandler "Encargalo.app-api.go/internal/riders/handler"
	"github.com/labstack/echo/v4"
)

type RidersGroup interface {
	Resource(g *echo.Echo)
}

type ridersGroup struct {
	middle              auth.AuthMiddleware
	handlerRiders       ridersHandler.RiderHandler
	handlerProfile      ridersHandler.RiderProfileHandler
	handlerFinance      ridersHandler.FinanceHandler
	handlerStaffRiders  ridersHandler.StaffRiderHandler
	handlerStaffFinance ridersHandler.StaffFinanceHandler
}

func NewRidersGroup(
	middle auth.AuthMiddleware,
	handlerRiders ridersHandler.RiderHandler,
	handlerProfile ridersHandler.RiderProfileHandler,
	handlerFinance ridersHandler.FinanceHandler,
	handlerStaffRiders ridersHandler.StaffRiderHandler,
	handlerStaffFinance ridersHandler.StaffFinanceHandler,
) RidersGroup {
	return &ridersGroup{middle, handlerRiders, handlerProfile, handlerFinance, handlerStaffRiders, handlerStaffFinance}
}

func (r *ridersGroup) Resource(g *echo.Echo) {
	// Ruta histórica del WS de riders (spec rider-websocket); se registra aquí
	// para no mezclar el módulo riders dentro de OrdersGroup.
	g.GET("/orders/rider", r.handlerRiders.HandleWebSocket, r.middle.Auth, r.middle.Require(models.Rider))

	riders := g.Group("/riders", r.middle.Auth, r.middle.Require(models.Rider))
	riders.GET("/profile", r.handlerProfile.GetProfile)
	riders.GET("/trips/active", r.handlerRiders.GetActiveTrips)
	riders.GET("/balance", r.handlerFinance.GetBalance)
	riders.POST("/withdrawal", r.handlerFinance.RequestWithdrawal)
	riders.GET("/transactions", r.handlerFinance.ListTransactions)

	staffRiders := g.Group("/riders/staff", r.middle.Auth, r.middle.Require(models.Staff))
	staffRiders.GET("/candidates", r.handlerStaffRiders.FindCandidate)
	staffRiders.GET("/profiles", r.handlerStaffRiders.ListProfiles)
	staffRiders.GET("/profiles/:userId", r.handlerStaffRiders.GetProfile)
	staffRiders.PUT("/profiles/:userId", r.handlerStaffRiders.UpsertProfile)
	staffRiders.DELETE("/profiles/:userId/photos/:field", r.handlerStaffRiders.DeletePhoto)
	staffRiders.POST("/profiles/:userId/activate", r.handlerStaffRiders.Activate)
	staffRiders.POST("/profiles/:userId/suspend", r.handlerStaffRiders.Suspend)
	staffRiders.POST("/profiles/:userId/touch-contact", r.handlerStaffRiders.TouchContact)

	staffRiders.GET("/finance/payables", r.handlerStaffFinance.ListPayables)
	staffRiders.GET("/finance/:riderId/balance", r.handlerStaffFinance.GetBalance)
	staffRiders.GET("/finance/:riderId/transactions", r.handlerStaffFinance.ListTransactions)
	staffRiders.POST("/finance/:riderId/payouts", r.handlerStaffFinance.RecordPayout)
	staffRiders.POST("/finance/:riderId/adjustments", r.handlerStaffFinance.RecordAdjustment)
}
