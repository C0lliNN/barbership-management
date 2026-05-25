package http

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the API HTTP handler (a Gin engine) with base middleware and
// operational endpoints. pinger is used by GET /ready to probe DB connectivity;
// pass nil to skip the DB check (useful in tests or early bootstrap).
func NewRouter(logger *zap.Logger, pinger DBPinger) *gin.Engine {
	engine := gin.New()
	// Order: recover (outermost) -> request ID -> request logger.
	engine.Use(recoverer(logger), requestID(), requestLogger(logger))

	engine.GET("/health", handleHealth)
	engine.GET("/ready", handleReady(pinger))

	return engine
}
