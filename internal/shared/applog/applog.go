package applog

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strings"

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

	if status >= 500 {
		attrs = append(attrs, "stack", captureStack(6))
		slog.Error("solicitud fallida", attrs...)
		return
	}

	if status >= 400 {
		slog.Warn("solicitud fallida", attrs...)
	}
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
