package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DBPinger is satisfied by any type with a Ping method — notably *pgxpool.Pool.
// Keeping the interface here avoids an import cycle while remaining testable with mocks.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// handleHealth reports that the process is alive. It never checks dependencies.
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReady returns a closure that checks DB connectivity before reporting readiness.// If pinger is nil the handler always returns 200 (useful in tests / early bootstrap).
func handleReady(pinger DBPinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if pinger != nil {
			if err := pinger.Ping(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "not_ready",
					"error":  "database unreachable",
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
