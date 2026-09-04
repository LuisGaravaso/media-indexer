package server

import (
	"github.com/LuisGaravaso/media-indexer/internal/config"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures Gin engine with middleware, health checks, and API routes.
func SetupRouter(cfg *config.Config, db Pinger) *gin.Engine {
	if cfg.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health probes
	router.GET("/health", HealthHandler(db))
	router.GET("/healthz", HealthHandler(db))
	router.GET("/ping", PingHandler())

	return router
}
