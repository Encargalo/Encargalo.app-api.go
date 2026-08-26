package cookie

import (
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/shared/config"
)

type Role string

const (
	RoleCustomer Role = "customers"
	RoleBusiness Role = "employee"
)

type CookieHandler interface {
	CreateCookieSession(jwtSession string) *http.Cookie
	DeleteCookieSession() *http.Cookie
}

type cookie struct {
}

func NewCookieFactory() CookieHandler {
	// switch role {
	// case RoleCustomer:
	// 	return &cookie{
	// 		name: "customer-session",
	// 		path: "/customers",
	// 	}
	// case RoleBusiness:
	// 	return &cookie{
	// 		name: "employee-session",
	// 		path: "/employee",
	// 	}
	// default:
	// 	return &cookie{
	// 		name: "unknown-session",
	// 		path: "/",
	// 	}
	return &cookie{}
}

func (c *cookie) CreateCookieSession(jwtSession string) *http.Cookie {
	return &http.Cookie{
		Name:     "encargalo-session",
		Value:    jwtSession,
		Secure:   true,
		SameSite: setSameSite(),
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	}
}

func (c *cookie) DeleteCookieSession() *http.Cookie {
	return &http.Cookie{
		Name:     "encargalo-session",
		Value:    "",
		Secure:   true,
		SameSite: setSameSite(),
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func setSameSite() http.SameSite {
	if !config.Get().IsProduction() {
		return http.SameSiteNoneMode
	}
	return http.SameSiteStrictMode
}
