package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/angristan/spotify-search-proxy/internal/infra/repository/cache"
	internalSpotify "github.com/angristan/spotify-search-proxy/internal/infra/repository/spotify"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type SpotifySearchService struct {
	Tracer        otelTrace.Tracer
	SpotifyClient *internalSpotify.SpotifyClient
	Cache         *cache.Cache
}

func NewSpotifySearchService(
	tracer otelTrace.Tracer,
	spotifyClient *internalSpotify.SpotifyClient,
	cache *cache.Cache,
) SpotifySearchService {
	return SpotifySearchService{
		Tracer:        tracer,
		SpotifyClient: spotifyClient,
		Cache:         cache,
	}
}

var (
	InvalidQueryTypeErr = fmt.Errorf("Invalid type")
	NoResultsFoundErr   = fmt.Errorf("No results found")
	SpotifyClientErr    = fmt.Errorf("Spotify client error")
)

func (service SpotifySearchService) Search(ctx context.Context, query string, searchType string) (interface{}, error) {
	var spotifyQueryType internalSpotify.SearchType
	switch searchType {
	case "artist":
		spotifyQueryType = internalSpotify.SearchTypeArtist
	case "album":
		spotifyQueryType = internalSpotify.SearchTypeAlbum
	case "track":
		spotifyQueryType = internalSpotify.SearchTypeTrack
	default:
		return nil, fmt.Errorf("%w: %s", InvalidQueryTypeErr, searchType)
	}

	// Check if the result is cached
	key := "spotify:" + searchType + ":" + query
	val, err := service.Cache.Get(ctx, key)
	if err == nil && val != "" {
		var cachedResult interface{}
		err = json.Unmarshal([]byte(val), &cachedResult)
		if err == nil {
			return cachedResult, nil
		} else {
			return nil, err //TODO err
		}
	}

	fmt.Printf("cc\n")

	// The Spotify SDK will re-encode it, so we need to decode it first
	// TODO move?
	decodedQuery, err := url.QueryUnescape(query)
	if err != nil {
		return nil, err //TODO err
	}

	// Search for the query
	result, err := service.SpotifyClient.Search(ctx, decodedQuery, spotifyQueryType)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", SpotifyClientErr, err.Error())
	}
	if result == nil {
		return nil, NoResultsFoundErr
	}

	fmt.Printf("result: %+v\n", result)

	// Cache the result
	marshaledResult, err := json.Marshal(result)
	if err != nil {
		return nil, err //TODO err
	}
	err = service.Cache.Set(ctx, key, marshaledResult, time.Hour*24)
	if err != nil {
		return nil, err //TODO err
	}

	return result, nil
}
