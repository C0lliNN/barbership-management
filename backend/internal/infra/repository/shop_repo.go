package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gcollin65/barbershop/internal/database"
	"github.com/gcollin65/barbershop/internal/identity"
)

// ShopRepository is the PostgreSQL implementation of identity.ShopRepository.
type ShopRepository struct{ pool *pgxpool.Pool }

func NewShopRepository(pool *pgxpool.Pool) *ShopRepository {
	return &ShopRepository{pool: pool}
}

func (r *ShopRepository) Create(ctx context.Context, shop identity.Shop) (identity.Shop, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`INSERT INTO shop (name, slug, phone, address, city, state)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id::text, created_at, updated_at`,
		shop.Name, shop.Slug,
		nullText(shop.Phone), nullText(shop.Address), nullText(shop.City), nullText(shop.State),
	).Scan(&shop.ID, &createdAt, &updatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if pgErr.ConstraintName == "shop_slug_unique" {
				return identity.Shop{}, identity.ErrSlugTaken
			}
		}
		return identity.Shop{}, fmt.Errorf("shop create: %w", err)
	}
	shop.CreatedAt = createdAt.Unix()
	shop.UpdatedAt = updatedAt.Unix()
	return shop, nil
}

func (r *ShopRepository) GetByID(ctx context.Context, id string) (identity.Shop, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var shop identity.Shop
	var phone, address, city, state pgtype.Text
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id::text, name, slug, phone, address, city, state, created_at, updated_at
		 FROM shop WHERE id = $1::uuid`, id,
	).Scan(&shop.ID, &shop.Name, &shop.Slug, &phone, &address, &city, &state,
		&createdAt, &updatedAt)
	if err != nil {
		if isNotFound(err) {
			return identity.Shop{}, identity.ErrNotFound
		}
		return identity.Shop{}, fmt.Errorf("shop get by id: %w", err)
	}
	shop.Phone = phone.String
	shop.Address = address.String
	shop.City = city.String
	shop.State = state.String
	shop.CreatedAt = createdAt.Unix()
	shop.UpdatedAt = updatedAt.Unix()
	return shop, nil
}

func (r *ShopRepository) Update(ctx context.Context, shop identity.Shop) (identity.Shop, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var phone, address, city, state pgtype.Text
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`UPDATE shop SET phone = $2, address = $3, city = $4, state = $5, updated_at = now()
		 WHERE id = $1::uuid
		 RETURNING id::text, name, slug, phone, address, city, state, created_at, updated_at`,
		shop.ID,
		nullText(shop.Phone), nullText(shop.Address), nullText(shop.City), nullText(shop.State),
	).Scan(&shop.ID, &shop.Name, &shop.Slug, &phone, &address, &city, &state,
		&createdAt, &updatedAt)
	if err != nil {
		if isNotFound(err) {
			return identity.Shop{}, identity.ErrNotFound
		}
		return identity.Shop{}, fmt.Errorf("shop update: %w", err)
	}
	shop.Phone = phone.String
	shop.Address = address.String
	shop.City = city.String
	shop.State = state.String
	shop.CreatedAt = createdAt.Unix()
	shop.UpdatedAt = updatedAt.Unix()
	return shop, nil
}

func (r *ShopRepository) GetBySlug(ctx context.Context, slug string) (identity.Shop, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var shop identity.Shop
	var phone, address, city, state pgtype.Text
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id::text, name, slug, phone, address, city, state, created_at, updated_at
		 FROM shop WHERE slug = $1`, slug,
	).Scan(&shop.ID, &shop.Name, &shop.Slug, &phone, &address, &city, &state,
		&createdAt, &updatedAt)
	if err != nil {
		if isNotFound(err) {
			return identity.Shop{}, identity.ErrNotFound
		}
		return identity.Shop{}, fmt.Errorf("shop get by slug: %w", err)
	}
	shop.Phone = phone.String
	shop.Address = address.String
	shop.City = city.String
	shop.State = state.String
	shop.CreatedAt = createdAt.Unix()
	shop.UpdatedAt = updatedAt.Unix()
	return shop, nil
}
