package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	apihttp.RegisterIdentityRoutes(s.router.Group("/v1"), s.signer)
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
