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
	name string
	path string
}

func NewCookieFactory(role Role) CookieHandler {
	switch role {
	case RoleCustomer:
		return &cookie{
			name: "customer-session",
			path: "/customers",
		}
	case RoleBusiness:
		return &cookie{
			name: "employee-session",
			path: "/employee",
		}
	default:
		return &cookie{
			name: "unknown-session",
			path: "/",
		}
	}
}

func (c *cookie) CreateCookieSession(jwtSession string) *http.Cookie {
	return &http.Cookie{
		Name:     c.name, // Usamos la configuración inyectada por la Factory
		Value:    jwtSession,
		Secure:   true,
		SameSite: setSameSite(),
		HttpOnly: true,
		Path:     c.path, // Usamos la configuración inyectada por la Factory
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	}
}

func (c *cookie) DeleteCookieSession() *http.Cookie {
	return &http.Cookie{
		Name:     c.name, // Importante: Para borrar una cookie, el Nombre y Path deben coincidir
		Value:    "",
		Secure:   true,
		SameSite: setSameSite(),
		HttpOnly: true,
		Path:     c.path, // Usamos la configuración inyectada por la Factory
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}

func setSameSite() http.SameSite {
	if config.Get().Environment.Current != "production" {
		return http.SameSiteNoneMode
	}
	return http.SameSiteStrictMode
}
