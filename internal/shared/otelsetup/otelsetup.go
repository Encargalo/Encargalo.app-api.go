// Package otelsetup inicializa el envío de trazas y logs al OTel Collector vía OTLP.
package otelsetup

import (
	"context"
	"log/slog"

	"Encargalo.app-api.go/internal/shared/config"
	"Encargalo.app-api.go/internal/shared/reqctx"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

type Shutdown func(context.Context) error

func noopShutdown(context.Context) error { return nil }

// Setup arma trazas (TracerProvider global) y logs (slog.Handler vía otelslog) apuntando
// al OTLP endpoint configurado. Si cfg.Enabled() es false (sin collector, ej. en local),
// devuelve un handler nil y un shutdown no-op sin tocar los providers globales.
func Setup(ctx context.Context, cfg config.Observability, environment, serviceName string) (slog.Handler, Shutdown, error) {
	if !cfg.Enabled() {
		return nil, noopShutdown, nil
	}
	if serviceName == "" {
		serviceName = reqctx.ServiceName
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.DeploymentEnvironmentNameKey.String(environment),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	var headers map[string]string
	if cfg.OtlpToken != "" {
		headers = map[string]string{"Authorization": "Bearer " + cfg.OtlpToken}
	}

	// WithEndpointURL fija el path exacto de la URL (incluso vacío), pisando el default
	// de la señal (/v1/traces, /v1/logs) — hay que pasarlo explícito con WithURLPath.
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(cfg.OtlpEndpoint),
		otlptracehttp.WithURLPath("/v1/traces"),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(cfg.OtlpEndpoint),
		otlploghttp.WithURLPath("/v1/logs"),
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, nil, err
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	handler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(loggerProvider))

	shutdown := func(ctx context.Context) error {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			return err
		}
		return loggerProvider.Shutdown(ctx)
	}

	return handler, shutdown, nil
}
