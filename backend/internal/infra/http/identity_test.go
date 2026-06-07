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

func init() {
	gin.SetMode(gin.TestMode)
}

const testJWTSecret = "test-secret"

type SignUpHandlerSuite struct {
	suite.Suite
	signer *mocks.MockSigner
	router *gin.Engine
}

func TestSignUpHandlerSuite(t *testing.T) {
	suite.Run(t, new(SignUpHandlerSuite))
}

func (s *SignUpHandlerSuite) SetupTest() {
	s.signer = mocks.NewMockSigner(s.T())
	s.router = gin.New()
	apihttp.RegisterIdentityRoutes(s.router.Group("/v1"), s.signer, testJWTSecret)
}

func (s *SignUpHandlerSuite) post(body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/signup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *SignUpHandlerSuite) TestHappyPath() {
	s.signer.EXPECT().SignUp(mock.Anything, mock.Anything).Return(identity.SignUpResponse{
		Shop:  identity.Shop{ID: "shop-1", Name: "Barbearia do João", Slug: "barbearia-do-joao"},
		Owner: identity.User{ID: "user-1", Email: "joao@test.com", FullName: "João Silva"},
	}, nil)

	w := s.post(`{"shop":{"name":"Barbearia do João"},"owner":{"email":"joao@test.com","password":"Secret123","full_name":"João Silva"}}`)

	s.Equal(http.StatusCreated, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))

	shopData := resp["shop"].(map[string]any)
	ownerData := resp["owner"].(map[string]any)
	s.Equal("barbearia-do-joao", shopData["slug"])
	s.Equal("joao@test.com", ownerData["email"])
	s.NotContains(ownerData, "password_hash")
}

func (s *SignUpHandlerSuite) TestMissingShopName() {
	w := s.post(`{"shop":{},"owner":{"email":"e@test.com","password":"Secret123","full_name":"Name"}}`)

	s.Equal(http.StatusUnprocessableEntity, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("validation failed", resp["error"])
}

func (s *SignUpHandlerSuite) TestInvalidEmail() {
	w := s.post(`{"shop":{"name":"Valid Shop"},"owner":{"email":"not-email","password":"Secret123","full_name":"Name"}}`)
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *SignUpHandlerSuite) TestShortPassword() {
	w := s.post(`{"shop":{"name":"Valid Shop"},"owner":{"email":"e@test.com","password":"short","full_name":"Name"}}`)
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *SignUpHandlerSuite) TestDuplicateEmail() {
	s.signer.EXPECT().SignUp(mock.Anything, mock.Anything).Return(identity.SignUpResponse{}, identity.ErrEmailTaken)

	w := s.post(`{"shop":{"name":"Valid Shop"},"owner":{"email":"e@test.com","password":"Secret123","full_name":"Name"}}`)

	s.Equal(http.StatusConflict, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	s.Equal("email already registered", resp["error"])
}

func (s *SignUpHandlerSuite) TestSlugExhausted() {
	s.signer.EXPECT().SignUp(mock.Anything, mock.Anything).Return(identity.SignUpResponse{}, identity.ErrSlugTaken)

	w := s.post(`{"shop":{"name":"Valid Shop"},"owner":{"email":"e@test.com","password":"Secret123","full_name":"Name"}}`)

	s.Equal(http.StatusConflict, w.Code)
}

type LoginHandlerSuite struct {
	suite.Suite
	signer *mocks.MockSigner
	router *gin.Engine
}

func TestLoginHandlerSuite(t *testing.T) {
	suite.Run(t, new(LoginHandlerSuite))
}

func (s *LoginHandlerSuite) SetupTest() {
	s.signer = mocks.NewMockSigner(s.T())
	s.router = gin.New()
	apihttp.RegisterIdentityRoutes(s.router.Group("/v1"), s.signer, testJWTSecret)
}

func (s *LoginHandlerSuite) do(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	var reader *bytes.Buffer
	if body != "" {
		reader = bytes.NewBufferString(body)
	} else {
		reader = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	return w
}

func (s *LoginHandlerSuite) bearerToken(shopID string, role identity.Role, expiry time.Duration) string {
	token, err := identity.SignToken(testJWTSecret, expiry, identity.User{ID: "user-1", Email: "joao@test.com"}, shopID, role)
	s.Require().NoError(err)
	return token
}

func (s *LoginHandlerSuite) TestLoginHappyPath() {
	s.signer.EXPECT().Login(mock.Anything, mock.Anything).Return(identity.LoginResponse{
		Token:  "signed.jwt.token",
		User:   identity.User{ID: "user-1", Email: "joao@test.com", FullName: "João Silva"},
		ShopID: "shop-1",
		Role:   identity.RoleOwner,
	}, nil)

	w := s.do(http.MethodPost, "/v1/login", `{"email":"joao@test.com","password":"Secret123"}`, nil)
	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("signed.jwt.token", resp["token"])
	s.Equal("shop-1", resp["shop_id"])
	s.Equal("owner", resp["role"])
	user := resp["user"].(map[string]any)
	s.Equal("joao@test.com", user["email"])
}

func (s *LoginHandlerSuite) TestLoginWrongPassword() {
	s.signer.EXPECT().Login(mock.Anything, mock.Anything).Return(identity.LoginResponse{}, identity.ErrInvalidCredentials)

	w := s.do(http.MethodPost, "/v1/login", `{"email":"joao@test.com","password":"wrongpass"}`, nil)
	s.Equal(http.StatusUnauthorized, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("invalid email or password", resp["error"])
}

func (s *LoginHandlerSuite) TestLoginUnknownEmail() {
	s.signer.EXPECT().Login(mock.Anything, mock.Anything).Return(identity.LoginResponse{}, identity.ErrInvalidCredentials)

	w := s.do(http.MethodPost, "/v1/login", `{"email":"ghost@test.com","password":"Secret123"}`, nil)
	s.Equal(http.StatusUnauthorized, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("invalid email or password", resp["error"])
}

func (s *LoginHandlerSuite) TestLoginMissingEmail() {
	w := s.do(http.MethodPost, "/v1/login", `{"password":"Secret123"}`, nil)
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *LoginHandlerSuite) TestLoginMissingPassword() {
	w := s.do(http.MethodPost, "/v1/login", `{"email":"joao@test.com"}`, nil)
	s.Equal(http.StatusUnprocessableEntity, w.Code)
}

func (s *LoginHandlerSuite) TestMeWithValidToken() {
	token := s.bearerToken("shop-1", identity.RoleOwner, time.Hour)

	w := s.do(http.MethodGet, "/v1/me", "", map[string]string{"Authorization": "Bearer " + token})
	s.Equal(http.StatusOK, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("user-1", resp["user_id"])
	s.Equal("joao@test.com", resp["email"])
	s.Equal("shop-1", resp["shop_id"])
	s.Equal("owner", resp["role"])
}

func (s *LoginHandlerSuite) TestMeNoHeader() {
	w := s.do(http.MethodGet, "/v1/me", "", nil)
	s.Equal(http.StatusUnauthorized, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("missing or malformed token", resp["error"])
}

func (s *LoginHandlerSuite) TestMeMalformedToken() {
	w := s.do(http.MethodGet, "/v1/me", "", map[string]string{"Authorization": "Bearer garbage"})
	s.Equal(http.StatusUnauthorized, w.Code)

	var resp map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &resp))
	s.Equal("invalid or expired token", resp["error"])
}

func (s *LoginHandlerSuite) TestMeExpiredToken() {
	token := s.bearerToken("shop-1", identity.RoleOwner, -time.Hour)

	w := s.do(http.MethodGet, "/v1/me", "", map[string]string{"Authorization": "Bearer " + token})
	s.Equal(http.StatusUnauthorized, w.Code)
}

func (s *LoginHandlerSuite) TestLogout() {
	w := s.do(http.MethodPost, "/v1/logout", "", nil)
	s.Equal(http.StatusOK, w.Code)
	s.JSONEq(`{}`, w.Body.String())
}

func (s *LoginHandlerSuite) TestLogoutWithToken() {
	token := s.bearerToken("shop-1", identity.RoleOwner, time.Hour)
	w := s.do(http.MethodPost, "/v1/logout", "", map[string]string{"Authorization": "Bearer " + token})
	s.Equal(http.StatusOK, w.Code)
	s.JSONEq(`{}`, w.Body.String())
}
