package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/identity/mocks"
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

// --- RequireShopMembership / RequireRole ---

type ShopMembershipMiddlewareSuite struct {
	suite.Suite
	members *mocks.MockMembershipRepository
	router  *gin.Engine
}

func TestShopMembershipMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(ShopMembershipMiddlewareSuite))
}

func (s *ShopMembershipMiddlewareSuite) SetupTest() {
	s.members = mocks.NewMockMembershipRepository(s.T())

	s.router = gin.New()
	scoped := s.router.Group("/shops/:shopID",
		AuthRequired(middlewareTestSecret),
		RequireShopMembership(s.members),
	)
	scoped.GET("", func(c *gin.Context) {
		shopID, ok := identity.TenantFromCtx(c.Request.Context())
		m := membershipFromContext(c)
		c.JSON(http.StatusOK, gin.H{"tenant": shopID, "tenant_set": ok, "role": string(m.Role)})
	})
	scoped.PATCH("", RequireRole(identity.RoleOwner), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func (s *ShopMembershipMiddlewareSuite) token(userID string) string {
	tok, err := identity.SignToken(middlewareTestSecret, time.Hour, identity.User{ID: userID, Email: "u@test.com"}, "", "")
	s.Require().NoError(err)
	return tok
}

func (s *ShopMembershipMiddlewareSuite) request(method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *ShopMembershipMiddlewareSuite) TestMemberPasses() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-a", "user-1").
		Return(identity.Membership{ID: "mem-1", ShopID: "shop-a", UserID: "user-1", Role: identity.RoleBarber}, nil)

	w := s.request(http.MethodGet, "/shops/shop-a", s.token("user-1"))

	s.Equal(http.StatusOK, w.Code)
	s.Contains(w.Body.String(), `"tenant":"shop-a"`)
	s.Contains(w.Body.String(), `"tenant_set":true`)
	s.Contains(w.Body.String(), `"role":"barber"`)
}

func (s *ShopMembershipMiddlewareSuite) TestNonMemberReturns404() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-a", "user-1").
		Return(identity.Membership{}, identity.ErrNotFound)

	w := s.request(http.MethodGet, "/shops/shop-a", s.token("user-1"))

	s.Equal(http.StatusNotFound, w.Code)
	s.JSONEq(`{"error":"not found"}`, w.Body.String())
}

func (s *ShopMembershipMiddlewareSuite) TestNonExistentShopReturns404() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-ghost", "user-1").
		Return(identity.Membership{}, identity.ErrNotFound)

	w := s.request(http.MethodGet, "/shops/shop-ghost", s.token("user-1"))

	s.Equal(http.StatusNotFound, w.Code)
	s.JSONEq(`{"error":"not found"}`, w.Body.String())
}

func (s *ShopMembershipMiddlewareSuite) TestRepositoryErrorReturns500() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-a", "user-1").
		Return(identity.Membership{}, errors.New("db exploded"))

	w := s.request(http.MethodGet, "/shops/shop-a", s.token("user-1"))

	s.Equal(http.StatusInternalServerError, w.Code)
}

func (s *ShopMembershipMiddlewareSuite) TestOwnerPassesRequireRole() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-a", "user-1").
		Return(identity.Membership{ID: "mem-1", ShopID: "shop-a", UserID: "user-1", Role: identity.RoleOwner}, nil)

	w := s.request(http.MethodPatch, "/shops/shop-a", s.token("user-1"))

	s.Equal(http.StatusOK, w.Code)
}

func (s *ShopMembershipMiddlewareSuite) TestBarberBlockedByRequireRole() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-a", "user-1").
		Return(identity.Membership{ID: "mem-1", ShopID: "shop-a", UserID: "user-1", Role: identity.RoleBarber}, nil)

	w := s.request(http.MethodPatch, "/shops/shop-a", s.token("user-1"))

	s.Equal(http.StatusForbidden, w.Code)
	s.JSONEq(`{"error":"insufficient role"}`, w.Body.String())
}
