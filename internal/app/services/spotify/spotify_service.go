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
	InvalidQueryTypeErr = fmt.Errorf("Invalid query type")
	NoResultsFoundErr   = fmt.Errorf("No results found")
	SpotifyClientErr    = fmt.Errorf("Spotify client error")
)
