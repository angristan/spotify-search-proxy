package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

func (s SpotifySearchService) Search(ctx context.Context, query string, searchType string) (any, error) {
	ctx, span := s.tracer.Start(ctx, "SpotifySearchService.Search")
	defer span.End()

	// Check if the query type is valid
	switch searchType {
	case "artist", "album", "track":
	default:
		return nil, fmt.Errorf("%w: %s", InvalidQueryTypeErr, searchType)
	}

	// Check if the result is cached
	key := "spotify:" + searchType + ":" + query
	val, err := s.cache.Get(ctx, key)
	if err == nil && val != "" {
		var cachedResult any
		err = json.Unmarshal([]byte(val), &cachedResult)
		if err == nil {
			return cachedResult, nil
		}
	}

	// The Spotify SDK will re-encode it, so we need to decode it first
	// TODO move?
	decodedQuery, err := url.QueryUnescape(query)
	if err != nil {
		return nil, err // TODO err
	}

	// Search for the query
	result, err := s.spotifyClient.Search(ctx, decodedQuery, searchType)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", SpotifyClientErr, err.Error())
	}
	if result == nil {
		return nil, NoResultsFoundErr
	}

	// Cache the result
	marshaledResult, err := json.Marshal(result)
	if err != nil {
		return nil, err // TODO err
	}
	err = s.cache.Set(ctx, key, marshaledResult, time.Hour*24)
	if err != nil {
		span.RecordError(err)
	}

	return result, nil
}
