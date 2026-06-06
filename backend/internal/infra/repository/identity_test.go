//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gcollin65/barbershop/internal/database"
	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/infra/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testCounter uint64

func uniqueSuffix() string {
	n := atomic.AddUint64(&testCounter, 1)
	return fmt.Sprintf("%d", n)
}

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err, "open pool")
	require.NoError(t, database.RunMigrations(dsn, database.Migrations, zap.NewNop()), "migrate")
	t.Cleanup(pool.Close)
	return pool
}

func TestShopRepository_CreateAndGetByID(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewShopRepository(pool)

	input := identity.Shop{
		Name: "Test Shop Create",
		Slug: "test-shop-create-" + uniqueSuffix(),
	}
	created, err := repo.Create(context.Background(), input)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.NotZero(t, created.CreatedAt)

	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, input.Name, got.Name)
}

func TestUserRepository_CreateAndGetByEmail(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewUserRepository(pool)

	email := "pgtest+" + uniqueSuffix() + "@test.com"
	created, err := repo.Create(context.Background(), identity.User{
		Email:        email,
		PasswordHash: "$2a$12$placeholder",
		FullName:     "Test User",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	got, err := repo.GetByEmail(context.Background(), email)
	require.NoError(t, err)
	assert.Equal(t, email, got.Email)
}

func TestUserRepository_DuplicateEmail(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewUserRepository(pool)

	email := "dup+" + uniqueSuffix() + "@test.com"
	_, err := repo.Create(context.Background(), identity.User{
		Email: email, PasswordHash: "$2a$12$ph", FullName: "Dup User",
	})
	require.NoError(t, err, "first Create")

	_, err = repo.Create(context.Background(), identity.User{
		Email: email, PasswordHash: "$2a$12$ph", FullName: "Dup User 2",
	})
	assert.ErrorIs(t, err, identity.ErrEmailTaken)
}

func TestService_SignUp_EndToEnd(t *testing.T) {
	pool := openTestPool(t)
	svc := identity.NewService(repository.NewTxManager(pool), zap.NewNop())

	sfx := uniqueSuffix()
	req := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "E2E Barber " + sfx, State: "SP", City: "São Paulo"},
		Owner: identity.OwnerInput{Email: "e2e+" + sfx + "@test.com", Password: "SecretPass123", FullName: "E2E Owner"},
	}

	resp, err := svc.SignUp(context.Background(), req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Shop.ID)
	assert.NotEmpty(t, resp.Owner.ID)
	assert.NotZero(t, resp.Shop.CreatedAt)

	var count int
	row := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM membership m
		 JOIN shop s ON s.id = m.shop_id
		 JOIN "user" u ON u.id = m.user_id
		 WHERE s.id = $1::uuid AND u.id = $2::uuid AND m.role = 'owner'`,
		resp.Shop.ID, resp.Owner.ID)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 1, count)
}

func TestService_SignUp_DuplicateEmail_Rollback(t *testing.T) {
	pool := openTestPool(t)
	svc := identity.NewService(repository.NewTxManager(pool), zap.NewNop())

	email := "rollback+" + uniqueSuffix() + "@test.com"

	_, err := svc.SignUp(context.Background(), identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "First Shop " + uniqueSuffix()},
		Owner: identity.OwnerInput{Email: email, Password: "SecretPass123", FullName: "Owner"},
	})
	require.NoError(t, err, "first SignUp")

	secondShopName := "Second Shop " + uniqueSuffix()
	_, err = svc.SignUp(context.Background(), identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: secondShopName},
		Owner: identity.OwnerInput{Email: email, Password: "AnotherPass123", FullName: "Owner2"},
	})
	assert.ErrorIs(t, err, identity.ErrEmailTaken)

	var count int
	row := pool.QueryRow(context.Background(), `SELECT count(*) FROM shop WHERE name = $1`, secondShopName)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 0, count, "rolled-back shop should not exist")
}
