package cookie

import (
	"net/http"
	"time"
)

const (
	cookieName = "encargalo_session"
	cookiePath = "/"
)

type Cookie interface {
	CreateCookieSession(jwtSession string) *http.Cookie
	DeleteCookieSession() *http.Cookie
}

type cookie struct{}

func NewCookie() Cookie {
	return &cookie{}
}

func (c *cookie) CreateCookieSession(jwtSession string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    jwtSession,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Path:     cookiePath,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
	}
}

func (c *cookie) DeleteCookieSession() *http.Cookie {
	return &http.Cookie{
		Name:     "cookieName",
		Value:    "",
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
		Path:     cookiePath,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}
