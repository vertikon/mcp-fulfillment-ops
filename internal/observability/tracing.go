// Package observability provides OpenTelemetry tracing and Prometheus metrics
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerProvider wraps OpenTelemetry tracer provider
type TracerProvider struct {
	tp *tracesdk.TracerProvider
}

// InitTracing initializes OpenTelemetry tracing with Jaeger exporter
func InitTracing(serviceName, endpoint string) (*TracerProvider, error) {
	if endpoint == "" {
		endpoint = "http://localhost:4318/v1/traces"
	}

	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{tp: tp}, nil
}

// Shutdown shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	return tp.tp.Shutdown(ctx)
}

// Tracer returns a tracer for the given name
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
