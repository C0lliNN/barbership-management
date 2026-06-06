package identity

import "context"

// ShopRepository is the persistence boundary for tenant records.
type ShopRepository interface {
	Create(ctx context.Context, shop *Shop) error
	GetByID(ctx context.Context, id [16]byte) (*Shop, error)
	GetBySlug(ctx context.Context, slug string) (*Shop, error)
}

// UserRepository handles global user account records.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id [16]byte) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// MembershipRepository links users to shops. shopID is an explicit parameter
// on every method so tenant scope cannot be accidentally omitted.
type MembershipRepository interface {
	Create(ctx context.Context, m *Membership) error
	GetByShopAndUser(ctx context.Context, shopID, userID [16]byte) (*Membership, error)
	ListByShop(ctx context.Context, shopID [16]byte) ([]Membership, error)
	Delete(ctx context.Context, shopID, userID [16]byte) error
}
