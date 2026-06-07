package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gcollin65/barbershop/internal/identity"
)

const middlewareTestSecret = "middleware-test-secret"

func newAuthRequiredRouter() *gin.Engine {
	r := gin.New()
	protected := r.Group("/protected", AuthRequired(middlewareTestSecret))
	protected.GET("", func(c *gin.Context) {
		claims := claimsFromContext(c)
		c.JSON(http.StatusOK, gin.H{"sub": claims.Subject, "email": claims.Email})
	})
	return r
}

func authRequest(router *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAuthRequiredPassesValidToken(t *testing.T) {
	router := newAuthRequiredRouter()
	token, err := identity.SignToken(middlewareTestSecret, time.Hour, identity.User{ID: "user-1", Email: "joao@test.com"}, "shop-1", identity.RoleOwner)
	require.NoError(t, err)

	w := authRequest(router, "Bearer "+token)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sub":"user-1"`)
	assert.Contains(t, w.Body.String(), `"email":"joao@test.com"`)
}

func TestAuthRequiredAbortsNoHeader(t *testing.T) {
	router := newAuthRequiredRouter()

	w := authRequest(router, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing or malformed token")
}

func TestAuthRequiredAbortsMissingBearerPrefix(t *testing.T) {
	router := newAuthRequiredRouter()

	w := authRequest(router, "sometoken")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing or malformed token")
}

func TestAuthRequiredAbortsMalformedToken(t *testing.T) {
	router := newAuthRequiredRouter()

	w := authRequest(router, "Bearer not-a-jwt")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAuthRequiredAbortsExpiredToken(t *testing.T) {
	router := newAuthRequiredRouter()
	token, err := identity.SignToken(middlewareTestSecret, -time.Hour, identity.User{ID: "user-1", Email: "joao@test.com"}, "shop-1", identity.RoleOwner)
	require.NoError(t, err)

	w := authRequest(router, "Bearer "+token)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}

func TestAuthRequiredAbortsWrongSecret(t *testing.T) {
	router := newAuthRequiredRouter()
	token, err := identity.SignToken("a-different-secret", time.Hour, identity.User{ID: "user-1", Email: "joao@test.com"}, "shop-1", identity.RoleOwner)
	require.NoError(t, err)

	w := authRequest(router, "Bearer "+token)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired token")
}
