package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	otelTrace "go.opentelemetry.io/otel/trace"
)

type SpotifySearchService interface {
	Search(ctx context.Context, query string, searchType string) (interface{}, error)
}

type SpotifyHandler struct {
	Tracer               otelTrace.Tracer
	SpotifySearchService SpotifySearchService
}

func NewSpotifyHandler(
	tracer otelTrace.Tracer,
	spotifySearchService SpotifySearchService,
) *SpotifyHandler {
	return &SpotifyHandler{
		Tracer:               tracer,
		SpotifySearchService: spotifySearchService,
	}
}

func (handler *SpotifyHandler) Search(c *gin.Context) {
	ctx, span := handler.Tracer.Start(c.Request.Context(), "handleSearch")
	defer span.End()

	qType := c.Param("type")
	if qType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	query := c.Param("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	result, err := handler.SpotifySearchService.Search(ctx, query, qType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
