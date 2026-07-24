package groups

import (
	"time"

	"Encargalo.app-api.go/internal/auth/domain/middleware/auth"
	requestinfo "Encargalo.app-api.go/internal/auth/domain/middleware/requestInfo"
	"Encargalo.app-api.go/internal/auth/domain/models"
	handAuth "Encargalo.app-api.go/internal/auth/handler"
	sharedmw "Encargalo.app-api.go/internal/shared/middleware"
	"github.com/labstack/echo/v4"
)

// signInRateLimit acota los intentos de sign-in por IP para mitigar fuerza
// bruta / credential stuffing sobre las cuentas (phone+password).
const (
	signInRateLimitCount  = 10
	signInRateLimitWindow = time.Minute
)

type AuthGroup interface {
	Resource(g *echo.Echo)
}

type authGroup struct {
	middleAuth  auth.AuthMiddleware
	requestInfo requestinfo.Request
	handAuth    handAuth.Auth
	rateLimiter sharedmw.RateLimiter
}

func NewAuthGroup(middleAuth auth.AuthMiddleware, requestInfo requestinfo.Request, handAuth handAuth.Auth, rateLimiter sharedmw.RateLimiter) AuthGroup {
	return &authGroup{middleAuth, requestInfo, handAuth, rateLimiter}
}

func (a *authGroup) Resource(g *echo.Echo) {
	group := g.Group("/auth")

	rateLimit := func(route string) echo.MiddlewareFunc {
		return a.rateLimiter.LimitByIP("auth:sign-in:"+route, signInRateLimitCount, signInRateLimitWindow)
	}

	group.POST("/sign-in/users", a.handAuth.SignInCustomer, rateLimit("users"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/employees", a.handAuth.SignInEmployee, rateLimit("employees"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/riders", a.handAuth.SignInRider, rateLimit("riders"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/staff", a.handAuth.SignInStaff, rateLimit("staff"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/google", a.handAuth.SignInGoogle, rateLimit("google"), a.requestInfo.GetRequestInfo)
	group.DELETE("/logout", a.handAuth.Logout, a.middleAuth.Auth, a.middleAuth.Require(models.Customer, models.OrderManager, models.Owner, models.Rider, models.Staff))
}
