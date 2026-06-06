//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MustCreateShop(t *testing.T, pool *pgxpool.Pool, slug string) identity.Shop {
	t.Helper()
	shop := identity.Shop{
		Name: "Test Shop " + slug,
		Slug: slug,
	}
	var createdAt, updatedAt time.Time
	row := pool.QueryRow(context.Background(),
		`INSERT INTO shop (name, slug) VALUES ($1, $2)
		 RETURNING id::text, created_at, updated_at`,
		shop.Name, shop.Slug)
	if err := row.Scan(&shop.ID, &createdAt, &updatedAt); err != nil {
		t.Fatalf("MustCreateShop: %v", err)
	}
	shop.CreatedAt = createdAt.Unix()
	shop.UpdatedAt = updatedAt.Unix()
	return shop
}

func MustCreateUser(t *testing.T, pool *pgxpool.Pool, email string) identity.User {
	t.Helper()
	u := identity.User{
		Email:        email,
		PasswordHash: "$2a$10$placeholder_hash_for_testing_only",
		FullName:     "Test User " + fmt.Sprint(time.Now().UnixNano()),
	}
	var createdAt, updatedAt time.Time
	row := pool.QueryRow(context.Background(),
		`INSERT INTO "user" (email, password_hash, full_name) VALUES ($1, $2, $3)
		 RETURNING id::text, created_at, updated_at`,
		u.Email, u.PasswordHash, u.FullName)
	if err := row.Scan(&u.ID, &createdAt, &updatedAt); err != nil {
		t.Fatalf("MustCreateUser: %v", err)
	}
	u.CreatedAt = createdAt.Unix()
	u.UpdatedAt = updatedAt.Unix()
	return u
}

func MustCreateMembership(t *testing.T, pool *pgxpool.Pool, shopID, userID string, role identity.Role) identity.Membership {
	t.Helper()
	m := identity.Membership{ShopID: shopID, UserID: userID, Role: role}
	var createdAt time.Time
	row := pool.QueryRow(context.Background(),
		`INSERT INTO membership (shop_id, user_id, role) VALUES ($1::uuid, $2::uuid, $3)
		 RETURNING id::text, created_at`,
		m.ShopID, m.UserID, string(m.Role))
	if err := row.Scan(&m.ID, &createdAt); err != nil {
		t.Fatalf("MustCreateMembership: %v", err)
	}
	m.CreatedAt = createdAt.Unix()
	return m
}
