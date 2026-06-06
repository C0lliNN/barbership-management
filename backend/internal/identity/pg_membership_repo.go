package identity

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgMembershipRepo struct{ pool *pgxpool.Pool }

func NewMembershipRepository(pool *pgxpool.Pool) MembershipRepository {
	return &pgMembershipRepo{pool: pool}
}

func (r *pgMembershipRepo) Create(ctx context.Context, m *Membership) error { panic("not implemented") }
func (r *pgMembershipRepo) GetByShopAndUser(ctx context.Context, shopID, userID [16]byte) (*Membership, error) {
	panic("not implemented")
}
func (r *pgMembershipRepo) ListByShop(ctx context.Context, shopID [16]byte) ([]Membership, error) {
	panic("not implemented")
}
func (r *pgMembershipRepo) Delete(ctx context.Context, shopID, userID [16]byte) error {
	panic("not implemented")
}
