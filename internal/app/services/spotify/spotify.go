package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/angristan/spotify-search-proxy/internal/infra/repository/cache"
	internalSpotify "github.com/angristan/spotify-search-proxy/internal/infra/repository/spotify"
	"go.opentelemetry.io/otel/trace"
)

type SpotifyClient interface {
	Search(ctx context.Context, query string, searchType internalSpotify.SearchType) (interface{}, error)
}

type SpotifySearchService struct {
	tracer        trace.Tracer
	spotifyClient SpotifyClient
	cache         cache.Cache
}

func NewSpotifySearchService(
	tracer trace.Tracer,
	spotifyClient SpotifyClient,
	cache cache.Cache,
) SpotifySearchService {
	return SpotifySearchService{
		tracer:        tracer,
		spotifyClient: spotifyClient,
		cache:         cache,
	}
}

var (
	InvalidQueryTypeErr = fmt.Errorf("Invalid type")
	NoResultsFoundErr   = fmt.Errorf("No results found")
	SpotifyClientErr    = fmt.Errorf("Spotify client error")
)

func (s SpotifySearchService) Search(ctx context.Context, query string, searchType string) (interface{}, error) {
	ctx, span := s.tracer.Start(ctx, "SpotifySearchService.Search")
	defer span.End()

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
	val, err := s.cache.Get(ctx, key)
	if err == nil && val != "" {
		var cachedResult interface{}
		err = json.Unmarshal([]byte(val), &cachedResult)
		if err == nil {
			return cachedResult, nil
		}
	}

	// The Spotify SDK will re-encode it, so we need to decode it first
	// TODO move?
	decodedQuery, err := url.QueryUnescape(query)
	if err != nil {
		return nil, err //TODO err
	}

	// Search for the query
	result, err := s.spotifyClient.Search(ctx, decodedQuery, spotifyQueryType)
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
	err = s.cache.Set(ctx, key, marshaledResult, time.Hour*24)
	if err != nil {
		return nil, err //TODO err
	}

	return result, nil
}
