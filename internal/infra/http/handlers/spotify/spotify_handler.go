package spotify

import (
	"go.opentelemetry.io/otel/trace"
)

type SpotifyHandler struct {
	tracer               trace.Tracer
	spotifySearchService SpotifyService
}

func New(
	tracer trace.Tracer,
	spotifySearchService SpotifyService,
) *SpotifyHandler {
	return &SpotifyHandler{
		tracer:               tracer,
		spotifySearchService: spotifySearchService,
	}
}
