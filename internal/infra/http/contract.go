package server

import (
	"github.com/gin-gonic/gin"
)

type SpotifyHandler interface {
	Search(ctx *gin.Context)
}
