package middleware

import (
	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequestID genera un request_id por petición y lo guarda en el context de Echo
// y en el context de Go para que los logs lo lean.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = uuid.NewString()
			}

			c.Set(string(reqctx.RequestIDKey), requestID)
			c.Response().Header().Set(echo.HeaderXRequestID, requestID)

			ctx := reqctx.WithRequestID(c.Request().Context(), requestID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
