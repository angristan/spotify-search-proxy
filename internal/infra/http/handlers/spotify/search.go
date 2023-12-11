package spotify

import (
	"net/http"

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

	query := c.Param("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	result, err := h.spotifySearchService.Search(ctx, query, qType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
