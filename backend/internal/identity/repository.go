package identity

import "context"

// ShopRepository is the persistence boundary for tenant records.
type ShopRepository interface {
	Create(ctx context.Context, shop Shop) (Shop, error)
	GetByID(ctx context.Context, id string) (Shop, error)
	GetBySlug(ctx context.Context, slug string) (Shop, error)
}

// UserRepository handles global user account records.
type UserRepository interface {
	Create(ctx context.Context, user User) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}

// MembershipRepository links users to shops. shopID is an explicit parameter
// on every method so tenant scope cannot be accidentally omitted.
type MembershipRepository interface {
	Create(ctx context.Context, m Membership) (Membership, error)
	GetByShopAndUser(ctx context.Context, shopID, userID string) (Membership, error)
	ListByShop(ctx context.Context, shopID string) ([]Membership, error)
	// ListByUser returns all memberships for a given user across all shops.
	ListByUser(ctx context.Context, userID string) ([]Membership, error)
	Delete(ctx context.Context, shopID, userID string) error
}
