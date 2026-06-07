package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gcollin65/barbershop/internal/database"
	"github.com/gcollin65/barbershop/internal/identity"
)

// MembershipRepository is the PostgreSQL implementation of identity.MembershipRepository.
type MembershipRepository struct{ pool *pgxpool.Pool }

func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{pool: pool}
}

func (r *MembershipRepository) Create(ctx context.Context, m identity.Membership) (identity.Membership, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var createdAt time.Time
	err := q.QueryRow(ctx,
		`INSERT INTO membership (shop_id, user_id, role)
		 VALUES ($1::uuid, $2::uuid, $3)
		 RETURNING id::text, created_at`,
		m.ShopID, m.UserID, string(m.Role),
	).Scan(&m.ID, &createdAt)
	if err != nil {
		return identity.Membership{}, fmt.Errorf("membership create: %w", err)
	}
	m.CreatedAt = createdAt.Unix()
	return m, nil
}

func (r *MembershipRepository) GetByShopAndUser(ctx context.Context, shopID, userID string) (identity.Membership, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var m identity.Membership
	var role string
	var createdAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id::text, shop_id::text, user_id::text, role, created_at
		 FROM membership WHERE shop_id = $1::uuid AND user_id = $2::uuid`, shopID, userID,
	).Scan(&m.ID, &m.ShopID, &m.UserID, &role, &createdAt)
	if err != nil {
		if isNotFound(err) {
			return identity.Membership{}, identity.ErrNotFound
		}
		return identity.Membership{}, fmt.Errorf("membership get: %w", err)
	}
	m.Role = identity.Role(role)
	m.CreatedAt = createdAt.Unix()
	return m, nil
}

func (r *MembershipRepository) ListByShop(ctx context.Context, shopID string) ([]identity.Membership, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT id::text, shop_id::text, user_id::text, role, created_at
		 FROM membership WHERE shop_id = $1::uuid`, shopID,
	)
	if err != nil {
		return nil, fmt.Errorf("membership list: %w", err)
	}
	defer rows.Close()

	var memberships []identity.Membership
	for rows.Next() {
		var m identity.Membership
		var role string
		var createdAt time.Time
		if err := rows.Scan(&m.ID, &m.ShopID, &m.UserID, &role, &createdAt); err != nil {
			return nil, fmt.Errorf("membership list scan: %w", err)
		}
		m.Role = identity.Role(role)
		m.CreatedAt = createdAt.Unix()
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membership list rows: %w", err)
	}
	return memberships, nil
}

func (r *MembershipRepository) ListByUser(ctx context.Context, userID string) ([]identity.Membership, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT id::text, shop_id::text, user_id::text, role, created_at
		 FROM membership WHERE user_id = $1::uuid ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("membership list by user: %w", err)
	}
	defer rows.Close()

	var memberships []identity.Membership
	for rows.Next() {
		var m identity.Membership
		var role string
		var createdAt time.Time
		if err := rows.Scan(&m.ID, &m.ShopID, &m.UserID, &role, &createdAt); err != nil {
			return nil, fmt.Errorf("membership list by user scan: %w", err)
		}
		m.Role = identity.Role(role)
		m.CreatedAt = createdAt.Unix()
		memberships = append(memberships, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membership list by user rows: %w", err)
	}
	return memberships, nil
}

func (r *MembershipRepository) Delete(ctx context.Context, shopID, userID string) error {
	q := database.QuerierFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`DELETE FROM membership WHERE shop_id = $1::uuid AND user_id = $2::uuid`,
		shopID, userID,
	)
	if err != nil {
		return fmt.Errorf("membership delete: %w", err)
	}
	return nil
}
