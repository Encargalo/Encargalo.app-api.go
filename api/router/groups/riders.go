package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	"Encargalo.app-api.go/internal/auth/domain/models"
	ridersHandler "Encargalo.app-api.go/internal/riders/handler"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// saveApplicationBodyLimit cabe el formulario completo con fotos comprimidas; cada foto se limita aparte a 7MB.
const saveApplicationBodyLimit = "16M"

// saveApplicationRateLimit acota cuántas veces una cuenta sube fotos a Cloudinary.
const (
	saveApplicationRateLimitCount  = 30
	saveApplicationRateLimitWindow = time.Hour
)

type RidersGroup interface {
	Resource(g *echo.Echo)
}

type ridersGroup struct {
	middle              auth.AuthMiddleware
	handlerRiders       ridersHandler.RiderHandler
	handlerProfile      ridersHandler.RiderProfileHandler
	handlerApplication  ridersHandler.RiderApplicationHandler
	handlerFinance      ridersHandler.FinanceHandler
	handlerStaffRiders  ridersHandler.StaffRiderHandler
	handlerStaffFinance ridersHandler.StaffFinanceHandler
	rateLimiter         sharedmw.RateLimiter
}

func NewRidersGroup(
	middle auth.AuthMiddleware,
	handlerRiders ridersHandler.RiderHandler,
	handlerProfile ridersHandler.RiderProfileHandler,
	handlerApplication ridersHandler.RiderApplicationHandler,
	handlerFinance ridersHandler.FinanceHandler,
	handlerStaffRiders ridersHandler.StaffRiderHandler,
	handlerStaffFinance ridersHandler.StaffFinanceHandler,
	rateLimiter sharedmw.RateLimiter,
) RidersGroup {
	return &ridersGroup{middle, handlerRiders, handlerProfile, handlerApplication, handlerFinance, handlerStaffRiders, handlerStaffFinance, rateLimiter}
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

	// Quien aún no es rider entra con su sesión de customer (identify respondió onboarding).
	application := g.Group("/riders/application")
	application.GET("", r.handlerApplication.GetApplication, r.middle.Auth, r.middle.Require(models.Customer))
	application.PUT("", r.handlerApplication.SaveApplication,
		middleware.BodyLimit(saveApplicationBodyLimit), r.middle.Auth, r.middle.Require(models.Customer),
		r.rateLimiter.LimitByKey(sessionUserIDKey, "riders:application:save", saveApplicationRateLimitCount, saveApplicationRateLimitWindow))
	application.POST("/submit", r.handlerApplication.SubmitApplication, r.middle.Auth, r.middle.Require(models.Customer))

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
