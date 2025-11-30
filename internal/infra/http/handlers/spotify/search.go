package spotify

import (
	"errors"
	"net/http"
	"strings"

	appspotify "github.com/angristan/spotify-search-proxy/internal/app/services/spotify"
	"github.com/gin-gonic/gin"
)

func (h *SpotifyHandler) Search(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "SpotifyHandler.Search")
	defer span.End()

	qType := c.Param("type")
	if qType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
		return
	}

	query := strings.TrimPrefix(c.Param("query"), "/")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	result, err := h.spotifySearchService.Search(ctx, query, qType)
	if err != nil {
		status := http.StatusInternalServerError
		message := "internal server error"

		switch {
		case errors.Is(err, appspotify.InvalidQueryTypeErr):
			status = http.StatusBadRequest
			message = "invalid search type"
		case errors.Is(err, appspotify.NoResultsFoundErr):
			status = http.StatusNotFound
			message = "no results found"
		case errors.Is(err, appspotify.SpotifyClientErr):
			status = http.StatusBadGateway
			message = "spotify client error"
		}

		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, result)
}
