package middleware

import (
	"net/http"
	"time"

	"Encargalo.app-api.go/internal/shared/applog"
	"Encargalo.app-api.go/internal/shared/errcustom"
	"github.com/labstack/echo/v4"
)

// RequestLog registra todas las solicitudes HTTP (éxito y fallo) al terminar.
func RequestLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)

			status := c.Response().Status
			if err != nil {
				status = errcustom.HTTPStatus(err)
			}
			if status == 0 {
				status = http.StatusOK
			}

			applog.HTTPAccess(c, status, time.Since(start))
			return err
		}
	}
}
