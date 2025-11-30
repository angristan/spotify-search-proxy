package main

import (
	"context"
	"net/url"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func newSpanExporter(ctx context.Context, endpoint string) (trace.SpanExporter, error) {
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Scheme != "" {
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(parsed.Host),
		}
		if parsed.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if parsed.Path != "" && parsed.Path != "/" {
			opts = append(opts, otlptracehttp.WithURLPath(parsed.Path))
		}
		return otlptracehttp.New(ctx, opts...)
	}

	return otlptracehttp.New(ctx,
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
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
