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
func Setup(ctx context.Context, cfg config.Observability, environment string) (slog.Handler, Shutdown, error) {
	if !cfg.Enabled() {
		return nil, noopShutdown, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(reqctx.ServiceName),
			semconv.DeploymentEnvironmentNameKey.String(environment),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(cfg.OtlpEndpoint))
	if err != nil {
		return nil, nil, err
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithEndpointURL(cfg.OtlpEndpoint))
	if err != nil {
		return nil, nil, err
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	handler := otelslog.NewHandler(reqctx.ServiceName, otelslog.WithLoggerProvider(loggerProvider))

	shutdown := func(ctx context.Context) error {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			return err
		}
		return loggerProvider.Shutdown(ctx)
	}

	return handler, shutdown, nil
}
