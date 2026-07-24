package reqctx

import (
	"context"

	authmodels "Encargalo.app-api.go/internal/auth/domain/models"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

const ServiceName = "encargalo-api"
const ServiceNameETL = "encargalo-etl"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func RequestIDFromEcho(c echo.Context) string {
	if c == nil {
		return ""
	}
	if id, ok := c.Get(string(RequestIDKey)).(string); ok && id != "" {
		return id
	}
	return RequestID(c.Request().Context())
}

type RequestMeta struct {
	RequestID string
	Method    string
	Endpoint  string
	UserID    string
	SessionID string
	UserType  string
	ShopID    string
}

func MetaFromEcho(c echo.Context) RequestMeta {
	meta := RequestMeta{
		RequestID: RequestIDFromEcho(c),
	}
	if c == nil {
		return meta
	}
	if req := c.Request(); req != nil {
		meta.Method = req.Method
		meta.Endpoint = c.Path()
		if meta.Endpoint == "" {
			meta.Endpoint = req.URL.Path
		}
		if session, ok := req.Context().Value(authmodels.SessionKey).(authmodels.ActiveSession); ok {
			meta.UserID = session.UserID.String()
			meta.SessionID = session.ID.String()
			meta.UserType = string(session.UserType)
			if session.ShopID != nil && *session.ShopID != uuid.Nil {
				meta.ShopID = session.ShopID.String()
			}
		}
	}
	return meta
}
