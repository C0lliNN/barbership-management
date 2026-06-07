package http

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewRouter builds the API HTTP handler (a Gin engine) with base middleware and
// operational endpoints. pinger is used by GET /ready to probe DB connectivity;
// pass nil to skip the DB check (useful in tests or early bootstrap).
// allowedOrigins lists the browser origins permitted to call the API cross-origin.
func NewRouter(logger *zap.Logger, pinger DBPinger, allowedOrigins []string) *gin.Engine {
	engine := gin.New()
	// Order: recover (outermost) → CORS (short-circuits preflight, tags every
	// response incl. errors) → request ID → enrich logger → request logger.
	engine.Use(recoverer(logger), cors(allowedOrigins), requestID(), Logger(logger), requestLogger())

	engine.GET("/health", handleHealth)
	engine.GET("/ready", handleReady(pinger))

	return engine
}
