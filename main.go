package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	spotifyService "github.com/angristan/spotify-search-proxy/internal/app/services/spotify"
	server "github.com/angristan/spotify-search-proxy/internal/infra/http"
	"github.com/angristan/spotify-search-proxy/internal/infra/http/handlers"
	redisCache "github.com/angristan/spotify-search-proxy/internal/infra/repository/cache/redis"
	spotifyClient "github.com/angristan/spotify-search-proxy/internal/infra/repository/spotify"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/extra/redisotel/v9"
	goRedis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var APIClientLock sync.RWMutex
var APIClient *spotify.Client

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

	tracer := tracerProvider.Tracer("spotify-search-proxy")

	tracedHTTPClient := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			})),
	}

	// TODO: renew token
	// go renewToken(ctx)

	redisClient := goRedis.NewClient(&goRedis.Options{
		Addr: config.RedisURL,
	})

	err = redisotel.InstrumentTracing(redisClient)
	if err != nil {
		panic(err)
	}

	cache := redisCache.NewCache(tracer, redisClient, 1*time.Minute)

	spotifyClientConfig := spotifyClient.NewSpotifyClientConfig(
		config.SpotifyClientID,
		config.SpotifyClientSecret,
		tracedHTTPClient,
		tracer,
	)

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
