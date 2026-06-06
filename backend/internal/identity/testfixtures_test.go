//go:build integration

package identity_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MustCreateShop inserts a shop row and returns it. Fails the test on error.
func MustCreateShop(t *testing.T, pool *pgxpool.Pool, slug string) *identity.Shop {
	t.Helper()
	shop := &identity.Shop{
		Name:     "Test Shop " + slug,
		Slug:     slug,
		Timezone: "America/Sao_Paulo",
	}
	row := pool.QueryRow(context.Background(),
		`INSERT INTO shop (name, slug, timezone) VALUES ($1, $2, $3)
         RETURNING id, created_at, updated_at`,
		shop.Name, shop.Slug, shop.Timezone)
	if err := row.Scan(&shop.ID, &shop.CreatedAt, &shop.UpdatedAt); err != nil {
		t.Fatalf("MustCreateShop: %v", err)
	}
	return shop
}

// MustCreateUser inserts a user row and returns it. Fails the test on error.
func MustCreateUser(t *testing.T, pool *pgxpool.Pool, email string) *identity.User {
	t.Helper()
	u := &identity.User{
		Email:        email,
		PasswordHash: "$2a$10$placeholder_hash_for_testing_only",
		FullName:     "Test User " + fmt.Sprint(time.Now().UnixNano()),
	}
	row := pool.QueryRow(context.Background(),
		`INSERT INTO "user" (email, password_hash, full_name) VALUES ($1, $2, $3)
         RETURNING id, created_at, updated_at`,
		u.Email, u.PasswordHash, u.FullName)
	if err := row.Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		t.Fatalf("MustCreateUser: %v", err)
	}
	return u
}

// MustCreateMembership inserts a membership row and returns it. Fails the test on error.
func MustCreateMembership(t *testing.T, pool *pgxpool.Pool, shopID, userID [16]byte, role identity.Role) *identity.Membership {
	t.Helper()
	m := &identity.Membership{ShopID: shopID, UserID: userID, Role: role}
	row := pool.QueryRow(context.Background(),
		`INSERT INTO membership (shop_id, user_id, role) VALUES ($1, $2, $3)
         RETURNING id, created_at`,
		m.ShopID, m.UserID, string(m.Role))
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		t.Fatalf("MustCreateMembership: %v", err)
	}
	return m
}
