package identity

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Signer is implemented by Service and can be mocked in handler tests.
type Signer interface {
	SignUp(ctx context.Context, req SignUpRequest) (SignUpResponse, error)
}

// Service orchestrates identity operations.
type Service struct {
	shops       ShopRepository
	users       UserRepository
	memberships MembershipRepository
	bcryptCost  int
}

// Option configures a Service.
type Option func(*Service)

// WithBcryptCost overrides the bcrypt work factor (default 12).
// Pass bcrypt.MinCost in tests to keep them fast.
func WithBcryptCost(cost int) Option {
	return func(s *Service) { s.bcryptCost = cost }
}

func NewService(
	shops ShopRepository,
	users UserRepository,
	memberships MembershipRepository,
	opts ...Option,
) *Service {
	svc := &Service{
		shops:       shops,
		users:       users,
		memberships: memberships,
		bcryptCost:  12,
	}
	for _, o := range opts {
		o(svc)
	}
	return svc
}

// SignUpRequest carries the inputs for creating a new shop and its first owner.
type SignUpRequest struct {
	Shop  ShopInput  `json:"shop"`
	Owner OwnerInput `json:"owner"`
}

type ShopInput struct {
	Name     string `json:"name"     binding:"required,min=2,max=100"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"    binding:"omitempty,len=2"`
}

type OwnerInput struct {
	Email    string `json:"email"     binding:"required,email"`
	Password string `json:"password"  binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required,min=2,max=100"`
	Phone    string `json:"phone"`
}

// SignUpResponse is returned on successful signup.
type SignUpResponse struct {
	Shop  Shop `json:"shop"`
	Owner User `json:"owner"`
}

// SignUp creates a shop, its first owner user, and their membership.
// Atomicity is guaranteed by the transaction middleware on the HTTP handler.
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (SignUpResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Owner.Password), s.bcryptCost)
	if err != nil {
		return SignUpResponse{}, fmt.Errorf("hash password: %w", err)
	}
	return s.signUp(ctx, req, string(hash))
}

func (s *Service) signUp(ctx context.Context, req SignUpRequest, passwordHash string) (SignUpResponse, error) {
	shopInput := Shop{
		Name:    req.Shop.Name,
		Phone:   req.Shop.Phone,
		Address: req.Shop.Address,
		City:    req.Shop.City,
		State:   req.Shop.State,
	}

	baseSlug := slugify(req.Shop.Name)
	shopInput.Slug = baseSlug
	var shop Shop
	var slugErr error
	for attempt := 0; attempt < 6; attempt++ {
		if attempt > 0 {
			shopInput.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
		}
		shop, slugErr = s.shops.Create(ctx, shopInput)
		if slugErr == nil {
			break
		}
		if slugErr != ErrSlugTaken {
			return SignUpResponse{}, fmt.Errorf("create shop: %w", slugErr)
		}
	}
	if slugErr != nil {
		return SignUpResponse{}, ErrSlugTaken
	}

	user, err := s.users.Create(ctx, User{
		Email:        req.Owner.Email,
		PasswordHash: passwordHash,
		FullName:     req.Owner.FullName,
		Phone:        req.Owner.Phone,
	})
	if err != nil {
		return SignUpResponse{}, fmt.Errorf("create user: %w", err)
	}

	if _, err := s.memberships.Create(ctx, Membership{ShopID: shop.ID, UserID: user.ID, Role: RoleOwner}); err != nil {
		return SignUpResponse{}, fmt.Errorf("create membership: %w", err)
	}

	return SignUpResponse{Shop: shop, Owner: user}, nil
}
