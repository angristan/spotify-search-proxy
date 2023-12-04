package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/angristan/spotify-search-proxy/internal/infra/http/handlers"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Config struct {
	Port              string
	Handlers          *Handlers
	disableMiddleware bool
}

type Handlers struct {
	SpotifySearchHandler handlers.SpotifyHandler
}

func NewHandlers(
	SpotifySearchHandler *handlers.SpotifyHandler,
) *Handlers {
	return &Handlers{
		SpotifySearchHandler: *SpotifySearchHandler,
	}
}

// NewConfig returns a new config.
func NewConfig(
	port string,
	handlers *Handlers,
	disableMiddleware bool,
) Config {
	return Config{
		Port:              port,
		Handlers:          handlers,
		disableMiddleware: false,
	}
}

type Server struct {
	*http.Server
}

func NewServer(cfg Config) *Server {
	engine := gin.New()

	httpPort, err := strconv.Atoi(cfg.Port)
	if err != nil {
		panic(err) //TODO
	}

	internalServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", httpPort),
		Handler: routeBuilder(engine, cfg),
	}

	return &Server{internalServer}
}

func routeBuilder(engine *gin.Engine, cfg Config) *gin.Engine {
	r := configureMiddleware(engine, cfg)

	r.GET("/search/:type/:query", cfg.Handlers.SpotifySearchHandler.Search)

	return r
}

func configureMiddleware(engine *gin.Engine, cfg Config) *gin.Engine {
	if !cfg.disableMiddleware {
		engine.Use(gin.Recovery())
		engine.Use(gin.Logger())
		engine.Use(otelgin.Middleware("spotify-search-proxy"))
	}

	return engine
}
