package middleware

import (
	"fmt"
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/shared/applog"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	LimitByIP(keyPrefix string, limit int64, window time.Duration) echo.MiddlewareFunc
	// LimitByKey acota, por la key que devuelva keyFunc, cuántas requests
	// puede hacer un cliente en window. Si keyFunc devuelve ok=false (ej. no
	// hay sesión en contexto todavía), la request pasa sin contar — pensado
	// para encadenarse después de un middleware que sí garantice la key
	// (ej. Auth, que carga la sesión en el contexto del request).
	LimitByKey(keyFunc func(c echo.Context) (string, bool), keyPrefix string, limit int64, window time.Duration) echo.MiddlewareFunc
}

type rateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) RateLimiter {
	return &rateLimiter{client}
}

func (r *rateLimiter) LimitByIP(keyPrefix string, limit int64, window time.Duration) echo.MiddlewareFunc {
	return r.LimitByKey(func(c echo.Context) (string, bool) {
		return c.RealIP(), true
	}, keyPrefix, limit, window)
}

func (r *rateLimiter) LimitByKey(keyFunc func(c echo.Context) (string, bool), keyPrefix string, limit int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id, ok := keyFunc(c)
			if !ok {
				return next(c)
			}

			ctx := c.Request().Context()
			key := fmt.Sprintf("ratelimit:%s:%s", keyPrefix, id)

			count, err := r.client.Incr(ctx, key).Result()
			if err != nil {
				applog.Error(ctx, "rate limiter: fallo de redis, se permite la solicitud", err, "key", key)
				return next(c)
			}
			if count == 1 {
				r.client.Expire(ctx, key, window)
			}
			if count > limit {
				return echo.NewHTTPError(http.StatusTooManyRequests, "demasiados intentos, intenta de nuevo más tarde")
			}

			return next(c)
		}
	}
}
