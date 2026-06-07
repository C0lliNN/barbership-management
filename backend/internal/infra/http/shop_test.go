package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	apihttp "github.com/gcollin65/barbershop/internal/infra/http"
	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/identity/mocks"
)

const shopTestJWTSecret = "shop-test-secret"

type ShopHandlerSuite struct {
	suite.Suite
	shops   *mocks.MockShopManager
	members *mocks.MockMembershipRepository
	router  *gin.Engine
}

func TestShopHandlerSuite(t *testing.T) {
	suite.Run(t, new(ShopHandlerSuite))
}

func (s *ShopHandlerSuite) SetupTest() {
	s.shops = mocks.NewMockShopManager(s.T())
	s.members = mocks.NewMockMembershipRepository(s.T())
	s.router = gin.New()
	apihttp.RegisterShopRoutes(s.router.Group("/v1"), s.shops, s.members, shopTestJWTSecret)
}

func (s *ShopHandlerSuite) token(userID string) string {
	tok, err := identity.SignToken(shopTestJWTSecret, time.Hour, identity.User{ID: userID, Email: "u@test.com"}, "", "")
	s.Require().NoError(err)
	return tok
}

func (s *ShopHandlerSuite) do(method, path, token, body string) *httptest.ResponseRecorder {
	var reader *bytes.Buffer
	if body != "" {
		reader = bytes.NewBufferString(body)
	} else {
		reader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *ShopHandlerSuite) expectMembership(shopID, userID string, role identity.Role) {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, shopID, userID).
		Return(identity.Membership{ID: "mem-1", ShopID: shopID, UserID: userID, Role: role}, nil)
}

func (s *ShopHandlerSuite) testGetHappyPathForRole(role identity.Role) {
	s.expectMembership("shop-a", "user-1", role)
	s.shops.EXPECT().GetShop(mock.Anything, "shop-a").
		Return(identity.Shop{ID: "shop-a", Name: "Barbearia A", Slug: "barbearia-a"}, nil)

	w := s.do(http.MethodGet, "/v1/shops/shop-a", s.token("user-1"), "")
	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("shop-a", resp["id"])
	s.Equal("barbearia-a", resp["slug"])
}

func (s *ShopHandlerSuite) TestGetHappyPathOwner() { s.testGetHappyPathForRole(identity.RoleOwner) }

func (s *ShopHandlerSuite) TestGetHappyPathBarber() { s.testGetHappyPathForRole(identity.RoleBarber) }

func (s *ShopHandlerSuite) TestGetHappyPathCustomer() {
	s.testGetHappyPathForRole(identity.RoleCustomer)
}

func (s *ShopHandlerSuite) TestGetNonMemberReturns404() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-b", "user-1").
		Return(identity.Membership{}, identity.ErrNotFound)

	w := s.do(http.MethodGet, "/v1/shops/shop-b", s.token("user-1"), "")
	s.Equal(http.StatusNotFound, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("not found", resp["error"])
}

func (s *ShopHandlerSuite) TestGetServiceNotFoundMapsTo404() {
	s.expectMembership("shop-a", "user-1", identity.RoleOwner)
	s.shops.EXPECT().GetShop(mock.Anything, "shop-a").Return(identity.Shop{}, identity.ErrNotFound)

	w := s.do(http.MethodGet, "/v1/shops/shop-a", s.token("user-1"), "")
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *ShopHandlerSuite) TestPatchHappyPathOwner() {
	s.expectMembership("shop-a", "user-1", identity.RoleOwner)
	s.shops.EXPECT().UpdateShop(mock.Anything, "shop-a", mock.MatchedBy(func(in identity.ShopUpdateInput) bool {
		return in.Phone == "+5511988887777" && in.City == "São Paulo" && in.State == "SP"
	})).Return(identity.Shop{ID: "shop-a", Name: "Barbearia A", Slug: "barbearia-a", Phone: "+5511988887777", City: "São Paulo", State: "SP"}, nil)

	w := s.do(http.MethodPatch, "/v1/shops/shop-a", s.token("user-1"),
		`{"phone":"+5511988887777","address":"Rua Augusta, 123","city":"São Paulo","state":"SP"}`)

	s.Equal(http.StatusOK, w.Code)
	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("+5511988887777", resp["phone"])
	s.Equal("SP", resp["state"])
}

func (s *ShopHandlerSuite) TestPatchNonOwnerReturns403() {
	s.expectMembership("shop-a", "user-1", identity.RoleBarber)

	w := s.do(http.MethodPatch, "/v1/shops/shop-a", s.token("user-1"), `{"city":"São Paulo"}`)
	s.Equal(http.StatusForbidden, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("insufficient role", resp["error"])
}

func (s *ShopHandlerSuite) TestPatchNonMemberReturns404() {
	s.members.EXPECT().GetByShopAndUser(mock.Anything, "shop-b", "user-1").
		Return(identity.Membership{}, identity.ErrNotFound)

	w := s.do(http.MethodPatch, "/v1/shops/shop-b", s.token("user-1"), `{"city":"Hacked"}`)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *ShopHandlerSuite) TestPatchInvalidStateReturns422() {
	s.expectMembership("shop-a", "user-1", identity.RoleOwner)

	w := s.do(http.MethodPatch, "/v1/shops/shop-a", s.token("user-1"), `{"state":"São Paulo"}`)
	s.Equal(http.StatusUnprocessableEntity, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("validation failed", resp["error"])
}

func (s *ShopHandlerSuite) TestPatchServiceNotFoundMapsTo404() {
	s.expectMembership("shop-a", "user-1", identity.RoleOwner)
	s.shops.EXPECT().UpdateShop(mock.Anything, "shop-a", mock.Anything).Return(identity.Shop{}, identity.ErrNotFound)

	w := s.do(http.MethodPatch, "/v1/shops/shop-a", s.token("user-1"), `{"city":"São Paulo"}`)
	s.Equal(http.StatusNotFound, w.Code)
}

func (s *ShopHandlerSuite) TestUnauthenticatedReturns401() {
	w := s.do(http.MethodGet, "/v1/shops/shop-a", "", "")
	s.Equal(http.StatusUnauthorized, w.Code)
}
