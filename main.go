package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"

	spotifyService "github.com/angristan/spotify-search-proxy/internal/app/services/spotify"
	server "github.com/angristan/spotify-search-proxy/internal/infra/http"
	spotifyHandler "github.com/angristan/spotify-search-proxy/internal/infra/http/handlers/spotify"
	redisCache "github.com/angristan/spotify-search-proxy/internal/infra/repository/cache/redis"
	spotifyClient "github.com/angristan/spotify-search-proxy/internal/infra/repository/spotify"
	"github.com/redis/go-redis/extra/redisotel/v9"
	goRedis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	err := LoadEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load environment variables")
	}

	config := GetEnv()

	ctx := context.Background()

	var tracerProvider *sdktrace.TracerProvider
	var tracer trace.Tracer

	if config.TracingEnabled && config.OTLPEndpoint != "" {
		spanExporter, err := newSpanExporter(ctx, config.OTLPEndpoint)
		if err != nil {
			logrus.Fatalf("failed to initialize exporter: %v", err)
		}

		tracerProvider, err = newTracerProvider(spanExporter)
		if err != nil {
			logrus.Fatalf("failed to create trace provider: %v", err)
		}

		defer func() { _ = tracerProvider.Shutdown(ctx) }()

		otel.SetTracerProvider(tracerProvider)
		tracer = tracerProvider.Tracer("spotify-search-proxy")
	} else {
		logrus.Info("Tracing disabled; no OTLP endpoint configured")
		tracer = otel.Tracer("spotify-search-proxy")
	}

	tracedHTTPClient := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			})),
	}

	redisClient := goRedis.NewClient(&goRedis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		Username: config.RedisUsername,
	})

	err = redisotel.InstrumentTracing(redisClient)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to instrument Redis tracing")
	}

	cache := redisCache.New(tracer, redisClient)

	spotifyClientConfig := spotifyClient.NewSpotifyClientConfig(
		config.SpotifyClientID,
		config.SpotifyClientSecret,
		tracedHTTPClient,
		tracer,
	)

	spotifyClient, err := spotifyClient.New(ctx, spotifyClientConfig)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create Spotify client")
	}

	spotifyService := spotifyService.New(tracer, spotifyClient, cache)

	spotifyHandler := spotifyHandler.New(tracer, spotifyService)

	serverConfig := server.NewConfig(config.Port, false)

	httpServer, err := server.New(serverConfig, spotifyHandler)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create HTTP server")
	}

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logrus.WithError(err).Fatal("HTTP server failed")
	}
}
