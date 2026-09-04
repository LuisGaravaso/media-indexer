package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger defines the interface for database ping probes.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler returns a Gin handler for readiness probes (/health).
func HealthHandler(db Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
				"service": "media-indexer",
				"db":     "disabled",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "degraded",
				"service": "media-indexer",
				"db":      "down",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "media-indexer",
			"db":      "up",
		})
	}
}

// PingHandler returns a fast liveness probe (/ping) without downstream checks.
func PingHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "media-indexer",
		})
	}
}
