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
		otlptracehttp.WithEndpoint("tempo:4318"))
}

func newTracerProvider(spanExporter trace.SpanExporter) (*trace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("spotify-search-proxy"),
		),
	)

	if err != nil {
		return nil, err
	}

	return trace.NewTracerProvider(
		trace.WithBatcher(spanExporter),
		trace.WithResource(r),
	), nil
}
