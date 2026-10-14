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
	middleAuth    auth.AuthMiddleware
	requestInfo   requestinfo.Request
	handAuth      handAuth.Auth
	rateLimiter   sharedmw.RateLimiter
	gatewaySecret sharedmw.GatewaySecretAuth
}

func NewAuthGroup(middleAuth auth.AuthMiddleware, requestInfo requestinfo.Request, handAuth handAuth.Auth, rateLimiter sharedmw.RateLimiter, gatewaySecret sharedmw.GatewaySecretAuth) AuthGroup {
	return &authGroup{middleAuth, requestInfo, handAuth, rateLimiter, gatewaySecret}
}

func (a *authGroup) Resource(g *echo.Echo) {
	group := g.Group("/auth")

	rateLimit := func(route string) echo.MiddlewareFunc {
		return a.rateLimiter.LimitByIP("auth:sign-in:"+route, signInRateLimitCount, signInRateLimitWindow)
	}

	group.POST("/sign-in/users", a.handAuth.SignInCustomer, rateLimit("users"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/employees", a.handAuth.SignInEmployee, rateLimit("employees"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/riders", a.handAuth.SignInRider, rateLimit("riders"), a.requestInfo.GetRequestInfo)
	group.POST("/riders/identify", a.handAuth.IdentifyRiderPhone, rateLimit("riders-identify"))
	group.POST("/sign-in/staff", a.handAuth.SignInStaff, rateLimit("staff"), a.requestInfo.GetRequestInfo)
	group.POST("/sign-in/google", a.handAuth.SignInGoogle, rateLimit("google"), a.requestInfo.GetRequestInfo)
	group.DELETE("/logout", a.handAuth.Logout, a.middleAuth.Auth, a.middleAuth.Require(models.Customer, models.OrderManager, models.Owner, models.Rider, models.Staff))

	// Endpoint interno para el ForwardAuth del reverse proxy (Traefik). Debe
	// ser GET: por defecto ForwardAuth siempre llama al servidor de auth con
	// GET, sin importar el método original de la request (a menos que se
	// configure preserveRequestMethod, que no usamos) — un método distinto
	// aquí devuelve 405 y Traefik lo reenvía tal cual al cliente, rompiendo
	// todo el tráfico. No lleva middleAuth.Auth/Require: sería circular, ya
	// que Auth pasa a depender del header que este mismo endpoint produce.
	// Se registra fuera del prefijo /auth porque no es una ruta de negocio.
	g.GET("/internal/verify-session", a.handAuth.VerifySession, a.gatewaySecret.Validate)
}
