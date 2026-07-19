package applog

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/labstack/echo/v4"
)

// SetupJSON configura slog como JSON en stderr (errores/warnings en producción).
func SetupJSON(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler).With("service", reqctx.ServiceName))
}

func attrsFromMeta(meta reqctx.RequestMeta) []any {
	attrs := []any{
		"request_id", meta.RequestID,
		"endpoint", meta.Endpoint,
		"method", meta.Method,
	}
	if meta.UserID != "" {
		attrs = append(attrs, "user_id", meta.UserID)
	}
	if meta.SessionID != "" {
		attrs = append(attrs, "session_id", meta.SessionID)
	}
	if meta.UserType != "" {
		attrs = append(attrs, "user_type", meta.UserType)
	}
	if meta.ShopID != "" {
		attrs = append(attrs, "shop_id", meta.ShopID)
	}
	return attrs
}

func Error(ctx context.Context, msg string, err error, extra ...any) {
	attrs := append([]any{}, extra...)
	if id := reqctx.RequestID(ctx); id != "" {
		attrs = append(attrs, "request_id", id)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	slog.Error(msg, attrs...)
}

func Warn(ctx context.Context, msg string, err error, extra ...any) {
	attrs := append([]any{}, extra...)
	if id := reqctx.RequestID(ctx); id != "" {
		attrs = append(attrs, "request_id", id)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	slog.Warn(msg, attrs...)
}

// HTTPFailure loguea un fallo de request: 4xx como warning, 5xx como error.
// Si el error trae causa interna (echo.HTTPError.Internal o wrap), la incluye en "cause".
func HTTPFailure(c echo.Context, status int, err error) {
	if c == nil || err == nil {
		return
	}

	meta := reqctx.MetaFromEcho(c)
	attrs := attrsFromMeta(meta)
	attrs = append(attrs,
		"status", status,
		"error", err.Error(),
	)
	if cause := errorCause(err); cause != "" && cause != err.Error() {
		attrs = append(attrs, "cause", cause)
	}

	if status >= 500 {
		attrs = append(attrs, "stack", captureStack(6))
		slog.Error("solicitud fallida", attrs...)
		return
	}

	if status >= 400 {
		slog.Warn("solicitud fallida", attrs...)
	}
}

// errorCause extrae el detalle real para logs sin confundirlo con el mensaje público.
func errorCause(err error) string {
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		if he.Internal != nil {
			return he.Internal.Error()
		}
		// Message ya es lo público; sin Internal no hay causa extra.
		return ""
	}
	return ""
}

// HTTPAccess loguea el resultado de toda solicitud HTTP (2xx–5xx).
func HTTPAccess(c echo.Context, status int, latency time.Duration) {
	if c == nil {
		return
	}

	meta := reqctx.MetaFromEcho(c)
	attrs := attrsFromMeta(meta)
	attrs = append(attrs,
		"status", status,
		"latency_ms", latency.Milliseconds(),
		"bytes_out", c.Response().Size,
	)
	if req := c.Request(); req != nil && req.URL != nil {
		if q := req.URL.RawQuery; q != "" {
			attrs = append(attrs, "query", q)
		}
	}

	slog.Info("solicitud", attrs...)
}

func captureStack(skip int) string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	var b strings.Builder
	first := true
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.File, "runtime/") {
			if !more {
				break
			}
			continue
		}
		if !first {
			b.WriteString(" => ")
		}
		first = false
		b.WriteString(frame.Function)
		b.WriteByte(':')
		// line number as decimal without fmt import overhead for hot path — use append
		b.Write(appendInt(nil, frame.Line))
		if !more {
			break
		}
	}
	return b.String()
}

func appendInt(dst []byte, n int) []byte {
	if n == 0 {
		return append(dst, '0')
	}
	var tmp [12]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte('0' + n%10)
		n /= 10
	}
	return append(dst, tmp[i:]...)
}
