package main

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/storage/redis"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

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

	// Setup middleware
	redisStore := redis.New(redis.Config{
		URL:   "redis://" + config.RedisAddr,
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
		return echo.NewHTTPError(http.StatusBadRequest, "type must be one of artist, album, track")
	}

	// The Spotify SDK will re-encode it, so we need to decode it first
	decodedQuery, err := url.QueryUnescape(query)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Search for the query
	results, err := APIClient.Search(context.Background(), decodedQuery, spotifyQueryType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

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
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}

	return c.JSON(result)
}
