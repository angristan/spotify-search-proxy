package main

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"os"
	"sync"

	"github.com/gofiber/fiber/v2/log"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otelTrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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

	// Create a new Spotify client
	spotifyConfig := &clientcredentials.Config{
		ClientID:     config.SpotifyClientID,
		ClientSecret: config.SpotifyClientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := spotifyConfig.Token(ctx)
	if err != nil {
		panic(err)
	}

	tracedHTTPClient := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			})),
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, tracedHTTPClient)

	httpClient := spotifyauth.New().Client(ctx, token)
	APIClient = spotify.New(httpClient)

	go renewToken(ctx)

	redisClient = redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	if err := redisotel.InstrumentTracing(redisClient); err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.Use(otelmux.Middleware("spotify-search-proxy"))
	r.HandleFunc("/search/{type}/{query}", handleSearch).Methods("GET")
	http.Handle("/", r)
	wrappedRouter := handlers.CombinedLoggingHandler(os.Stdout, r)
	wrappedRouter = handlers.CompressHandler(wrappedRouter)
	wrappedRouter = handlers.RecoveryHandler()(wrappedRouter)
	wrappedRouter = handlers.ProxyHeaders(wrappedRouter)
	err = http.ListenAndServe(":"+config.Port, wrappedRouter)
	if err != nil {
		panic(err)
	}
}
