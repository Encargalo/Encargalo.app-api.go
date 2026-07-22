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
}

type rateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) RateLimiter {
	return &rateLimiter{client}
}

func (r *rateLimiter) LimitByIP(keyPrefix string, limit int64, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			key := fmt.Sprintf("ratelimit:%s:%s", keyPrefix, c.RealIP())

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
