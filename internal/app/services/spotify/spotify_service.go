package spotify

import (
	"fmt"

	"go.opentelemetry.io/otel/trace"
)

type SpotifySearchService struct {
	tracer        trace.Tracer
	spotifyClient SpotifyClient
	cache         Cache
}

func New(
	tracer trace.Tracer,
	spotifyClient SpotifyClient,
	cache Cache,
) SpotifySearchService {
	return SpotifySearchService{
		tracer:        tracer,
		spotifyClient: spotifyClient,
		cache:         cache,
	}
}

var (
	ErrInvalidQueryType = fmt.Errorf("invalid query type")
	ErrNoResultsFound   = fmt.Errorf("no results found")
	ErrSpotifyClient    = fmt.Errorf("spotify client error")
)
