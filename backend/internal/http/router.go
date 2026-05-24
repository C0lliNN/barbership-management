package http

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the API HTTP handler (a Gin engine) with base middleware and
// operational endpoints wired up.
func NewRouter(logger *zap.Logger) *gin.Engine {
	engine := gin.New()
	// Order: recover (outermost) -> request ID -> request logger.
	engine.Use(recoverer(logger), requestID(), requestLogger(logger))

	engine.GET("/health", handleHealth)
	engine.GET("/ready", handleReady)

	return engine
}
