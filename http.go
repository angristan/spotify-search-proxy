package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/zmb3/spotify/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelTrace "go.opentelemetry.io/otel/trace"
)

func handleSearch(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "handleSearch")
	defer span.End()

	qType := c.Param("type")
	if qType == "" {
		log.Error("Type is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	query := c.Param("query")
	if query == "" {
		log.Error("Query is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	span.SetAttributes(
		attribute.String("query", query),
		attribute.String("type", qType),
	)

	var spotifyQueryType spotify.SearchType
	switch qType {
	case "artist":
		spotifyQueryType = spotify.SearchTypeArtist
	case "album":
		spotifyQueryType = spotify.SearchTypeAlbum
	case "track":
		spotifyQueryType = spotify.SearchTypeTrack
	default:
		err := fmt.Errorf("Invalid type: %s", qType)
		log.Error(err)
		span.SetStatus(codes.Error, "Invalid type: "+qType)
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of artist, album, track"})
		return
	}

	// Check if the result is cached
	ctx, redisSpan := tracer.Start(ctx, "redisGet")
	key := "spotify:" + qType + ":" + query
	val, err := redisClient.Get(ctx, key).Result()
	if err == nil {
		redisSpan.AddEvent("Cache hit", otelTrace.WithAttributes(attribute.String("key", key)))
		var cachedResult interface{}
		err = json.Unmarshal([]byte(val), &cachedResult)
		redisSpan.End()
		if err == nil {
			c.JSON(http.StatusOK, cachedResult)
			return
		} else {
			log.Error(err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		}
	} else {
		if err != redis.Nil {
			log.Error(err)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		redisSpan.AddEvent("Cache miss")
	}
	redisSpan.End()

	// The Spotify SDK will re-encode it, so we need to decode it first
	decodedQuery, err := url.QueryUnescape(query)
	if err != nil {
		log.Error(err)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Search for the query
	ctx, spotifySpan := tracer.Start(ctx, "spotfiySearch")
	spotifySpan.SetAttributes(
		attribute.String("spotify-query", decodedQuery),
		attribute.String("spotify-type", qType),
	)
	spotifySpan.AddEvent("Acquiring lock")
	APIClientLock.RLock()
	spotifySpan.AddEvent("Lock acquired")
	results, err := APIClient.Search(ctx, decodedQuery, spotifyQueryType)
	if err != nil {
		log.Error(err)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	spotifySpan.AddEvent("Releasing lock")
	APIClientLock.RUnlock()
	spotifySpan.AddEvent("Lock released")
	spotifySpan.End()

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
		log.Error("No results found")
		span.SetStatus(codes.Error, "No results found")
		c.JSON(http.StatusNotFound, gin.H{"error": "No results found"})
		return
	}

	// Cache the result
	ctx, redisSpan = tracer.Start(ctx, "Caching result")
	marshaledResult, err := json.Marshal(result)
	err = redisClient.Set(ctx, key, marshaledResult, time.Hour*24).Err()
	if err != nil {
		log.Error(err)
		span.SetStatus(codes.Error, err.Error())
	}
	redisSpan.End()

	c.JSON(http.StatusOK, result)
	return
}
