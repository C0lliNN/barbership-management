package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleHealth reports that the process is alive.
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReady reports readiness to serve traffic.
//
// For now this returns 200 unconditionally; Item 003 will extend it to check
// dependencies such as database connectivity.
func handleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
