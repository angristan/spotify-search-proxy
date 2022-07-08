package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

var APIClient *spotify.Client
var redisClient *redis.Client

func main() {
	err := LoadEnv()
	if err != nil {
		logrus.WithError(err).Warn("Failed to load environment variables")
	}

	config := GetEnv()

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

	redisClient = redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})

	e := echo.New()

	e.GET("/search/:type/:query", handleSearch)

	e.Use(middleware.Logger())

	e.Logger.Fatal(e.Start(":" + config.Port))
}

func handleSearch(c echo.Context) error {
	qType := c.Param("type")
	if qType == "" {
		return c.JSON(400, "type is required")
	}

	query := c.Param("query")
	if query == "" {
		return c.JSON(400, "query is required")
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

	// Check if we have a cached result for this query
	val, err := redisClient.Get(fmt.Sprintf("%s:%s", qType, query)).Result()
	if err == nil {
		logrus.WithField("query", query).Info("Found cached result")
		var result interface{}
		err = json.Unmarshal([]byte(val), &result)
		if err != nil {
			logrus.WithError(err).Error("Failed to unmarshal cached result")
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to unmarshal cached result")
		}
		return c.JSON(200, result)
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

	// Cache the result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	err = redisClient.Set(fmt.Sprintf("%s:%s", qType, query), resultBytes, time.Hour*24*7).Err()
	if err != nil {
		logrus.WithError(err).Error("failed to set redis key")
	}

	return c.JSON(http.StatusOK, result)
}
