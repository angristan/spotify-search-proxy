package main

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func newSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	return otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint("tempo:4318"), //TODO: Use env var
	)
}

func newTracerProvider(spanExporter trace.SpanExporter) (*trace.TracerProvider, error) {
	resource, err := resource.New(context.Background(),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(semconv.ServiceName("spotify-search-proxy")),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return nil, err
	}

	return trace.NewTracerProvider(
		trace.WithBatcher(spanExporter),
		trace.WithResource(resource),
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
	), nil
}
