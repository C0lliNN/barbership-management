package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Signer is implemented by Service and can be mocked in handler tests.
type Signer interface {
	SignUp(ctx context.Context, req SignUpRequest) (SignUpResponse, error)
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
}

// ShopManager is implemented by Service and can be mocked in handler tests.
// Membership is verified by middleware before these run, so both methods
// trust the given shopID and operate as plain tenant-scoped lookups/updates.
type ShopManager interface {
	GetShop(ctx context.Context, shopID string) (Shop, error)
	UpdateShop(ctx context.Context, shopID string, input ShopUpdateInput) (Shop, error)
}

// ShopUpdateInput carries the contact/location fields an Owner can edit.
// name/slug are intentionally excluded — see Item 010 design notes.
type ShopUpdateInput struct {
	Phone   string `json:"phone"`
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state" binding:"omitempty,len=2"`
}

// Service orchestrates identity operations.
type Service struct {
	shops       ShopRepository
	users       UserRepository
	memberships MembershipRepository
	bcryptCost  int
	jwtSecret   string
	jwtExpiry   time.Duration
}

// Option configures a Service.
type Option func(*Service)

// WithBcryptCost overrides the bcrypt work factor (default 12).
// Pass bcrypt.MinCost in tests to keep them fast.
func WithBcryptCost(cost int) Option {
	return func(s *Service) { s.bcryptCost = cost }
}

// WithJWTSecret sets the secret used to sign and verify access tokens.
func WithJWTSecret(secret string) Option {
	return func(s *Service) { s.jwtSecret = secret }
}

// WithJWTExpiry overrides the access token lifetime (default 24h).
func WithJWTExpiry(d time.Duration) Option {
	return func(s *Service) { s.jwtExpiry = d }
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
		jwtExpiry:   24 * time.Hour,
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

// LoginRequest carries email/password credentials for authentication.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is returned on successful authentication.
type LoginResponse struct {
	Token  string `json:"token"`
	User   User   `json:"user"`
	ShopID string `json:"shop_id"`
	Role   Role   `json:"role"`
}

// Login verifies the given credentials and, on success, returns a signed JWT
// along with the user's profile and primary shop membership (the first
// membership by created_at; empty if the user has no memberships).
//
// Returns ErrInvalidCredentials for both unknown emails and wrong passwords —
// the identical response prevents account enumeration.
func (s *Service) Login(ctx context.Context, req LoginRequest) (LoginResponse, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return LoginResponse{}, ErrInvalidCredentials
		}
		return LoginResponse{}, fmt.Errorf("login lookup: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return LoginResponse{}, ErrInvalidCredentials
	}

	var shopID string
	var role Role
	memberships, err := s.memberships.ListByUser(ctx, user.ID)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("login memberships: %w", err)
	}
	if len(memberships) > 0 {
		shopID = memberships[0].ShopID
		role = memberships[0].Role
	}

	token, err := SignToken(s.jwtSecret, s.jwtExpiry, user, shopID, role)
	if err != nil {
		return LoginResponse{}, fmt.Errorf("login sign: %w", err)
	}

	return LoginResponse{Token: token, User: user, ShopID: shopID, Role: role}, nil
}

// GetShop returns the shop's profile. Membership is verified by middleware
// before this runs, so it is a thin pass-through to the repository.
func (s *Service) GetShop(ctx context.Context, shopID string) (Shop, error) {
	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Shop{}, ErrNotFound
		}
		return Shop{}, fmt.Errorf("get shop: %w", err)
	}
	return shop, nil
}

// UpdateShop updates the shop's contact/location fields. Membership and role
// are verified by middleware before this runs.
func (s *Service) UpdateShop(ctx context.Context, shopID string, input ShopUpdateInput) (Shop, error) {
	shop, err := s.shops.Update(ctx, Shop{
		ID:      shopID,
		Phone:   input.Phone,
		Address: input.Address,
		City:    input.City,
		State:   input.State,
	})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Shop{}, ErrNotFound
		}
		return Shop{}, fmt.Errorf("update shop: %w", err)
	}
	return shop, nil
}
