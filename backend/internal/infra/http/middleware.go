package http

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/database"
	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/logger"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
	tenantHeader    = "X-Tenant"
	claimsKey       = "claims"
	membershipKey   = "membership"
)

// requestID attaches a request ID (from the inbound header or freshly generated)
// to the context and echoes it on the response.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		c.Set(requestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// Logger enriches the per-request zap.Logger with request_id and tenant fields
// and stores it in the request context. Must run after requestID().
func Logger(base *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		fields := []zap.Field{zap.String("request_id", c.GetString(requestIDKey))}
		if tenant := c.GetHeader(tenantHeader); tenant != "" {
			fields = append(fields, zap.String("tenant", tenant))
		}
		c.Request = c.Request.WithContext(
			logger.WithLogger(c.Request.Context(), base.With(fields...)),
		)
		c.Next()
	}
}

// requestLogger logs each request's method, path, status, and duration.
// request_id and tenant are already present on the context logger as fields.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.FromContext(c.Request.Context()).Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

// AuthRequired validates the JWT in the Authorization header and stores the
// parsed claims in the Gin context under the key "claims" (see claimsFromContext).
func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or malformed token"})
			return
		}
		tokenStr := header[len(prefix):]
		claims, err := identity.ParseToken(jwtSecret, tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

// claimsFromContext returns the JWT claims stored by AuthRequired.
// Must only be called on routes protected by AuthRequired.
func claimsFromContext(c *gin.Context) *identity.Claims {
	return c.MustGet(claimsKey).(*identity.Claims)
}

// membershipFromContext returns the membership stored by RequireShopMembership.
// Must only be called on routes protected by RequireShopMembership.
func membershipFromContext(c *gin.Context) identity.Membership {
	return c.MustGet(membershipKey).(identity.Membership)
}

// RequireShopMembership resolves the :shopID URL parameter and the caller's
// user ID (from AuthRequired claims) to a membership, verified directly
// against the repository — never against the JWT's embedded shop_id/role,
// which reflect the user's primary shop at login time and can go stale.
//
// On success it stores the membership in the Gin context and the verified
// shop ID via identity.WithTenant, so handlers and downstream layers use only
// the ID this middleware confirmed the caller belongs to.
//
// Returns 404 — not 403 — both when the shop doesn't exist and when the
// caller has no membership in it, so shop existence is never leaked.
func RequireShopMembership(memberships identity.MembershipRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		shopID := c.Param("shopID")
		claims := claimsFromContext(c)

		m, err := memberships.GetByShopAndUser(c.Request.Context(), shopID, claims.Subject)
		if err != nil {
			if errors.Is(err, identity.ErrNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			logger.FromContext(c.Request.Context()).Error("membership lookup failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Set(membershipKey, m)
		c.Request = c.Request.WithContext(identity.WithTenant(c.Request.Context(), shopID))
		c.Next()
	}
}

// RequireRole aborts with 403 unless the caller's membership (stored by
// RequireShopMembership) has one of the allowed roles. Must run after
// RequireShopMembership.
func RequireRole(roles ...identity.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		m := membershipFromContext(c)
		for _, role := range roles {
			if m.Role == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	}
}

// recoverer converts panics into a 500 response and logs them.
// Uses the context logger when available (enriched with request_id/tenant);
// falls back to base if Logger middleware has not yet run.
func recoverer(base *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		log := logger.FromContext(c.Request.Context())
		if log == zap.NewNop() {
			log = base
		}
		log.Error("panic recovered", zap.Any("panic", err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	})
}

// Transaction begins a database transaction before the handler runs and
// commits it on 2xx responses or rolls it back on 4xx/5xx responses and panics.
//
// Note: Gin writes the response into its buffer during the handler, so if
// Commit fails after a successful handler the client already received 2xx —
// the commit error is logged but cannot be surfaced to the caller.
func Transaction(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.FromContext(c.Request.Context())
		tx, err := pool.Begin(c.Request.Context())
		if err != nil {
			log.Error("failed to begin transaction", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		ctx := database.WithTx(c.Request.Context(), tx)
		c.Request = c.Request.WithContext(ctx)

		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback(ctx)
				panic(r) // let recoverer middleware handle it
			}
		}()

		c.Next()

		if c.Writer.Status() >= http.StatusBadRequest {
			_ = tx.Rollback(ctx)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			_ = tx.Rollback(ctx)
			log.Error("failed to commit transaction", zap.Error(err))
		}
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
