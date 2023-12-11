package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Server struct {
	*http.Server
}

func New(cfg Config, sh SpotifyHandler) *Server {
	engine := gin.New()

	httpPort, err := strconv.Atoi(cfg.Port)
	if err != nil {
		panic(err) //TODO
	}

	if !cfg.disableMiddleware {
		engine.Use(gin.Recovery())
		engine.Use(gin.Logger())
		engine.Use(otelgin.Middleware("spotify-search-proxy"))
	}

	engine.GET("/search/:type/:query", sh.Search)

	internalServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", httpPort),
		Handler: engine,
	}

	return &Server{internalServer}
}
