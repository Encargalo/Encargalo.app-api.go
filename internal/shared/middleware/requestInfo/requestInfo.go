package requestinfo

import (
	"context"

	"github.com/labstack/echo/v4"
)

type Request interface {
	GetRequestInfo(next echo.HandlerFunc) echo.HandlerFunc
}

type request struct {
}

type ctxKey string

const (
	keyIP        ctxKey = "ip"
	keyUserAgent ctxKey = "user-agent"
)

func NewRequestMiddleware() Request {
	return &request{}
}

func (r *request) GetRequestInfo(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		ctx := req.Context()

		ctx = context.WithValue(ctx, keyIP, c.RealIP())
		ctx = context.WithValue(ctx, keyUserAgent, req.Header.Get("User-Agent"))

		c.SetRequest(req.WithContext(ctx))
		return next(c)
	}
}
