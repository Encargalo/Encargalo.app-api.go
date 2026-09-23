package postgres

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"Encargalo.app-api.go/internal/shared/reqctx"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace"
)

// slowQueryThreshold es a partir de cuánto una query se considera "pesada" y
// vale la pena loguearla. Deliberadamente NO logueamos cada query (a
// diferencia del bundebug.QueryHook anterior, cuyo output verbose nunca
// llegaba a ningún lado consultable): a este volumen, loguear cada SELECT
// satura a Loki igual que ya lo hizo el dashboard de 6h con auto-refresh.
const slowQueryThreshold = 150 * time.Millisecond

const maxQueryTemplateLen = 300

// queryLogHook manda a slog (y de ahí a Loki, mismo pipeline que
// applog.HTTPAccess) solo las queries lentas o que fallan, con su forma
// (QueryTemplate) en vez del SQL final interpolado — QueryTemplate nunca
// trae los valores reales, así que no arriesga filtrar PII (teléfonos,
// coordenadas, etc.) en los logs.
type queryLogHook struct{}

func (queryLogHook) BeforeQuery(ctx context.Context, _ *bun.QueryEvent) context.Context {
	return ctx
}

func (queryLogHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	duration := time.Since(event.StartTime)
	if event.Err == nil && duration < slowQueryThreshold {
		return
	}

	attrs := []any{
		"db_operation", event.Operation(),
		"duration_ms", duration.Milliseconds(),
		"query_template", truncateQueryTemplate(event.QueryTemplate),
	}
	if id := reqctx.RequestID(ctx); id != "" {
		attrs = append(attrs, "request_id", id)
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		attrs = append(attrs, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	}

	if event.Err != nil {
		attrs = append(attrs, "error", event.Err.Error())
		slog.Error("db query failed", attrs...)
		return
	}

	slog.Warn("slow db query", attrs...)
}

func truncateQueryTemplate(q string) string {
	q = strings.Join(strings.Fields(q), " ")
	if len(q) > maxQueryTemplateLen {
		return q[:maxQueryTemplateLen] + "…"
	}
	return q
}
