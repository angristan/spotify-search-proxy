package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	spotifyService "github.com/angristan/spotify-search-proxy/internal/app/services/spotify" // TODO
	server "github.com/angristan/spotify-search-proxy/internal/infra/http"
	"github.com/angristan/spotify-search-proxy/internal/infra/http/handlers"
	"github.com/angristan/spotify-search-proxy/internal/infra/repository/cache"
	spotifyClient "github.com/angristan/spotify-search-proxy/internal/infra/repository/spotify" // TODO
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otelTrace "go.opentelemetry.io/otel/trace"
)

var APIClientLock sync.RWMutex
var APIClient *spotify.Client
var tracer otelTrace.Tracer
var redisClient *redis.Client

func main() {
	err := LoadEnv()
	if err != nil {
		logrus.WithError(err).Warn("Failed to load environment variables")
	}

	config := GetEnv()

	ctx := context.Background()

	spanExporter, err := newSpanExporter(ctx)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	tracerProvider, err := newTracerProvider(spanExporter)
	if err != nil {
		log.Fatalf("failed to create trace provider: %v", err)
	}

	defer func() { _ = tracerProvider.Shutdown(ctx) }()

	otel.SetTracerProvider(tracerProvider)

	tracer = tracerProvider.Tracer("spotify-search-proxy")

	tracedHTTPClient := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			})),
	}

	// go renewToken(ctx)

	redisClient = redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		panic(err)
	}

	cache := cache.NewCache(redisClient, 24*time.Hour)

	spotifyClientConfig := spotifyClient.NewSpotifyClientConfig(config.SpotifyClientID, config.SpotifyClientSecret, tracedHTTPClient)

	spotifyClient := spotifyClient.NewSpotifyClient(ctx, spotifyClientConfig)

	spotifyService := spotifyService.NewSpotifySearchService(tracer, spotifyClient, cache)

	spotifyHandler := handlers.NewSpotifyHandler(tracer, spotifyService)

	handlers := server.NewHandlers(spotifyHandler)

	serverConfig := server.NewConfig(config.Port, handlers, false)

	httpServer := server.NewServer(serverConfig)

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
