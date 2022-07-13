package main

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/storage/redis"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

var APIClientLock sync.RWMutex
var APIClient *spotify.Client

func main() {
	err := LoadEnv()
	if err != nil {
		logrus.WithError(err).Warn("Failed to load environment variables")
	}

	config := GetEnv()

	// Create a new Spotify client
	ctx := context.Background()
	spotifyConfig := &clientcredentials.Config{
		ClientID:     config.SpotifyClientID,
		ClientSecret: config.SpotifyClientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}
	token, err := spotifyConfig.Token(ctx)
	if err != nil {
		panic(err)
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	APIClient = spotify.New(httpClient)

	go renewToken()

	// Setup middleware
	redisStore := redis.New(redis.Config{
		URL:   config.RedisURL,
		Reset: false,
	})

	app := fiber.New()
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(etag.New())
	app.Use(cache.New(cache.Config{
		Expiration: time.Hour * 24,
		Storage:    redisStore,
	}))

	// Setup routes
	app.Get("/search/:type/:query", handleSearch)

	// Start server
	logrus.Fatal(app.Listen(":" + config.Port))
}

func handleSearch(c *fiber.Ctx) error {
	qType := c.Params("type")
	if qType == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "type is required",
		})
	}

	query := c.Params("query")
	if query == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "query is required",
		})
	}

	var spotifyQueryType spotify.SearchType
	switch qType {
	case "artist":
		spotifyQueryType = spotify.SearchTypeArtist
	case "album":
		spotifyQueryType = spotify.SearchTypeAlbum
	case "track":
		spotifyQueryType = spotify.SearchTypeTrack
	default:
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "type must be one of artist, album, track",
		})
	}

	// The Spotify SDK will re-encode it, so we need to decode it first
	decodedQuery, err := url.QueryUnescape(query)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Search for the query
	APIClientLock.RLock()
	results, err := APIClient.Search(context.Background(), decodedQuery, spotifyQueryType)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	APIClientLock.RUnlock()

	var result interface{}

	switch spotifyQueryType {
	case spotify.SearchTypeArtist:
		if results.Artists != nil {
			if len(results.Artists.Artists) > 0 {
				result = results.Artists.Artists[0]
			}
		}
	case spotify.SearchTypeAlbum:
		if results.Albums != nil {
			if len(results.Albums.Albums) > 0 {
				result = results.Albums.Albums[0]
			}
		}
	case spotify.SearchTypeTrack:
		if results.Tracks != nil {
			if len(results.Tracks.Tracks) > 0 {
				result = results.Tracks.Tracks[0]
			}
		}
	}

	if result == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "not found",
		})
	}

	return c.JSON(result)
}

// Check if the token expires soon, and if so recreates an API client with a new token
func renewToken() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		logrus.Info("Checking if Spotify token needs to be renewed")

		spotifyToken, err := APIClient.Token()
		if err != nil {
			logrus.WithError(err).Error("Failed to refresh token")
		}
		if time.Until(spotifyToken.Expiry) > time.Minute*5 {
			logrus.Info("Token is still valid, no need to refresh")
			return
		}

		ctx := context.Background()
		spotifyConfig := &clientcredentials.Config{
			ClientID:     GetEnv().SpotifyClientID,
			ClientSecret: GetEnv().SpotifyClientSecret,
			TokenURL:     spotifyauth.TokenURL,
		}
		token, err := spotifyConfig.Token(ctx)
		if err != nil {
			logrus.WithError(err).Error("Failed to refresh token")
			return
		}

		httpClient := spotifyauth.New().Client(ctx, token)
		APIClientLock.Lock()
		APIClient = spotify.New(httpClient)
		APIClientLock.Unlock()

		logrus.Info("Token refreshed")
	}
}
