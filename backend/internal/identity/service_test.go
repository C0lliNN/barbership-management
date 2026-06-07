package identity_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/identity/mocks"
)

type SignUpSuite struct {
	suite.Suite
	shops   *mocks.MockShopRepository
	users   *mocks.MockUserRepository
	members *mocks.MockMembershipRepository
	svc     *identity.Service
}

func TestSignUpSuite(t *testing.T) {
	suite.Run(t, new(SignUpSuite))
}

func (s *SignUpSuite) SetupTest() {
	s.shops = mocks.NewMockShopRepository(s.T())
	s.users = mocks.NewMockUserRepository(s.T())
	s.members = mocks.NewMockMembershipRepository(s.T())
	s.svc = identity.NewService(s.shops, s.users, s.members, identity.WithBcryptCost(bcrypt.MinCost))
}

func (s *SignUpSuite) TestHappyPath() {
	s.shops.EXPECT().Create(mock.Anything, mock.MatchedBy(func(shop identity.Shop) bool {
		return shop.Name == "Barbearia do João"
	})).Return(identity.Shop{ID: "shop-1", Slug: "barbearia-do-joao"}, nil)

	s.users.EXPECT().Create(mock.Anything, mock.MatchedBy(func(u identity.User) bool {
		return u.Email == "joao@test.com"
	})).Return(identity.User{ID: "user-1", Email: "joao@test.com"}, nil)

	s.members.EXPECT().Create(mock.Anything, mock.MatchedBy(func(m identity.Membership) bool {
		return m.Role == identity.RoleOwner && m.ShopID == "shop-1" && m.UserID == "user-1"
	})).Return(identity.Membership{ID: "mem-1"}, nil)

	req := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "Barbearia do João"},
		Owner: identity.OwnerInput{Email: "joao@test.com", Password: "Secret123", FullName: "João Silva"},
	}

	resp, err := s.svc.SignUp(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("barbearia-do-joao", resp.Shop.Slug)
	s.Equal("joao@test.com", resp.Owner.Email)
}

func (s *SignUpSuite) TestSlugCollisionRetry() {
	s.shops.EXPECT().Create(mock.Anything, mock.MatchedBy(func(shop identity.Shop) bool {
		return shop.Slug == "test-shop"
	})).Return(identity.Shop{}, identity.ErrSlugTaken).Once()
	s.shops.EXPECT().Create(mock.Anything, mock.MatchedBy(func(shop identity.Shop) bool {
		return shop.Slug == "test-shop-2"
	})).Return(identity.Shop{}, identity.ErrSlugTaken).Once()
	s.shops.EXPECT().Create(mock.Anything, mock.MatchedBy(func(shop identity.Shop) bool {
		return shop.Slug == "test-shop-3"
	})).Return(identity.Shop{ID: "shop-1", Slug: "test-shop-3"}, nil).Once()

	s.users.EXPECT().Create(mock.Anything, mock.Anything).Return(identity.User{ID: "user-1"}, nil)
	s.members.EXPECT().Create(mock.Anything, mock.Anything).Return(identity.Membership{ID: "mem-1"}, nil)

	req := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "Test Shop"},
		Owner: identity.OwnerInput{Email: "owner@test.com", Password: "Secret123", FullName: "Owner"},
	}

	resp, err := s.svc.SignUp(context.Background(), req)
	s.Require().NoError(err)
	s.Equal("test-shop-3", resp.Shop.Slug)
}

func (s *SignUpSuite) TestSlugExhausted() {
	s.shops.EXPECT().Create(mock.Anything, mock.Anything).
		Return(identity.Shop{}, identity.ErrSlugTaken).Times(6)

	req := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "Test Shop"},
		Owner: identity.OwnerInput{Email: "owner@test.com", Password: "Secret123", FullName: "Owner"},
	}

	_, err := s.svc.SignUp(context.Background(), req)
	s.ErrorIs(err, identity.ErrSlugTaken)
}

func (s *SignUpSuite) TestDuplicateEmail() {
	s.shops.EXPECT().Create(mock.Anything, mock.Anything).
		Return(identity.Shop{ID: "shop-1", Slug: "test-shop"}, nil)
	s.users.EXPECT().Create(mock.Anything, mock.Anything).
		Return(identity.User{}, identity.ErrEmailTaken)

	req := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "Test Shop"},
		Owner: identity.OwnerInput{Email: "dup@test.com", Password: "Secret123", FullName: "Dup"},
	}

	_, err := s.svc.SignUp(context.Background(), req)
	s.ErrorIs(err, identity.ErrEmailTaken)
}

type LoginSuite struct {
	suite.Suite
	shops   *mocks.MockShopRepository
	users   *mocks.MockUserRepository
	members *mocks.MockMembershipRepository
	svc     *identity.Service
}

func TestLoginSuite(t *testing.T) {
	suite.Run(t, new(LoginSuite))
}

func (s *LoginSuite) SetupTest() {
	s.shops = mocks.NewMockShopRepository(s.T())
	s.users = mocks.NewMockUserRepository(s.T())
	s.members = mocks.NewMockMembershipRepository(s.T())
	s.svc = identity.NewService(s.shops, s.users, s.members,
		identity.WithBcryptCost(bcrypt.MinCost),
		identity.WithJWTSecret("test-secret"),
		identity.WithJWTExpiry(time.Hour),
	)
}

func hashPassword(s *suite.Suite, password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	s.Require().NoError(err)
	return string(hash)
}

func (s *LoginSuite) TestHappyPathWithMembership() {
	hash := hashPassword(&s.Suite, "Secret123")
	s.users.EXPECT().GetByEmail(mock.Anything, "joao@test.com").
		Return(identity.User{ID: "user-1", Email: "joao@test.com", PasswordHash: hash}, nil)
	s.members.EXPECT().ListByUser(mock.Anything, "user-1").
		Return([]identity.Membership{{ID: "mem-1", ShopID: "shop-1", UserID: "user-1", Role: identity.RoleOwner}}, nil)

	resp, err := s.svc.Login(context.Background(), identity.LoginRequest{Email: "joao@test.com", Password: "Secret123"})
	s.Require().NoError(err)
	s.NotEmpty(resp.Token)
	s.Equal("shop-1", resp.ShopID)
	s.Equal(identity.RoleOwner, resp.Role)
	s.Equal("joao@test.com", resp.User.Email)
}

func (s *LoginSuite) TestHappyPathNoMembership() {
	hash := hashPassword(&s.Suite, "Secret123")
	s.users.EXPECT().GetByEmail(mock.Anything, "joao@test.com").
		Return(identity.User{ID: "user-1", Email: "joao@test.com", PasswordHash: hash}, nil)
	s.members.EXPECT().ListByUser(mock.Anything, "user-1").Return(nil, nil)

	resp, err := s.svc.Login(context.Background(), identity.LoginRequest{Email: "joao@test.com", Password: "Secret123"})
	s.Require().NoError(err)
	s.NotEmpty(resp.Token)
	s.Empty(resp.ShopID)
	s.Empty(resp.Role)
}

func (s *LoginSuite) TestUnknownEmail() {
	s.users.EXPECT().GetByEmail(mock.Anything, "ghost@test.com").
		Return(identity.User{}, identity.ErrNotFound)

	_, err := s.svc.Login(context.Background(), identity.LoginRequest{Email: "ghost@test.com", Password: "whatever1"})
	s.ErrorIs(err, identity.ErrInvalidCredentials)
}

func (s *LoginSuite) TestWrongPassword() {
	hash := hashPassword(&s.Suite, "Secret123")
	s.users.EXPECT().GetByEmail(mock.Anything, "joao@test.com").
		Return(identity.User{ID: "user-1", Email: "joao@test.com", PasswordHash: hash}, nil)

	_, err := s.svc.Login(context.Background(), identity.LoginRequest{Email: "joao@test.com", Password: "WrongPass1"})
	s.ErrorIs(err, identity.ErrInvalidCredentials)
}
