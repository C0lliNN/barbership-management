//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

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

// newIdentityService builds an identity.Service backed by the real Postgres
// repositories, matching how it's wired in cmd/api/main.go.
func newIdentityService(pool *pgxpool.Pool) *identity.Service {
	return identity.NewService(
		repository.NewShopRepository(pool),
		repository.NewUserRepository(pool),
		repository.NewMembershipRepository(pool),
		identity.WithBcryptCost(bcrypt.MinCost),
		identity.WithJWTSecret("integration-test-secret"),
		identity.WithJWTExpiry(time.Hour),
	)
}

// runInTx mirrors the production Transaction middleware: it commits fn's
// writes on success and rolls them back if fn returns an error, so direct
// service-level integration tests see the same atomicity guarantees as the
// HTTP handlers do.
func runInTx(t *testing.T, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "begin tx")

	txCtx := database.WithTx(ctx, tx)
	fnErr := fn(txCtx)
	if fnErr != nil {
		require.NoError(t, tx.Rollback(ctx), "rollback tx")
		return fnErr
	}
	require.NoError(t, tx.Commit(ctx), "commit tx")
	return nil
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
	svc := newIdentityService(pool)

	sfx := uniqueSuffix()
	req := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "E2E Barber " + sfx, State: "SP", City: "São Paulo"},
		Owner: identity.OwnerInput{Email: "e2e+" + sfx + "@test.com", Password: "SecretPass123", FullName: "E2E Owner"},
	}

	var resp identity.SignUpResponse
	err := runInTx(t, pool, func(ctx context.Context) error {
		var err error
		resp, err = svc.SignUp(ctx, req)
		return err
	})
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
	svc := newIdentityService(pool)

	email := "rollback+" + uniqueSuffix() + "@test.com"

	err := runInTx(t, pool, func(ctx context.Context) error {
		_, err := svc.SignUp(ctx, identity.SignUpRequest{
			Shop:  identity.ShopInput{Name: "First Shop " + uniqueSuffix()},
			Owner: identity.OwnerInput{Email: email, Password: "SecretPass123", FullName: "Owner"},
		})
		return err
	})
	require.NoError(t, err, "first SignUp")

	secondShopName := "Second Shop " + uniqueSuffix()
	err = runInTx(t, pool, func(ctx context.Context) error {
		_, err := svc.SignUp(ctx, identity.SignUpRequest{
			Shop:  identity.ShopInput{Name: secondShopName},
			Owner: identity.OwnerInput{Email: email, Password: "AnotherPass123", FullName: "Owner2"},
		})
		return err
	})
	assert.ErrorIs(t, err, identity.ErrEmailTaken)

	var count int
	row := pool.QueryRow(context.Background(), `SELECT count(*) FROM shop WHERE name = $1`, secondShopName)
	require.NoError(t, row.Scan(&count))
	assert.Equal(t, 0, count, "rolled-back shop should not exist")
}

func TestMembershipRepository_ListByUser(t *testing.T) {
	pool := openTestPool(t)
	shops := repository.NewShopRepository(pool)
	users := repository.NewUserRepository(pool)
	memberships := repository.NewMembershipRepository(pool)

	sfx := uniqueSuffix()
	shop, err := shops.Create(context.Background(), identity.Shop{
		Name: "ListByUser Shop " + sfx,
		Slug: "list-by-user-shop-" + sfx,
	})
	require.NoError(t, err)

	user, err := users.Create(context.Background(), identity.User{
		Email:        "listbyuser+" + sfx + "@test.com",
		PasswordHash: "$2a$04$placeholder",
		FullName:     "List By User",
	})
	require.NoError(t, err)

	created, err := memberships.Create(context.Background(), identity.Membership{
		ShopID: shop.ID, UserID: user.ID, Role: identity.RoleOwner,
	})
	require.NoError(t, err)

	got, err := memberships.ListByUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, created.ID, got[0].ID)
	assert.Equal(t, shop.ID, got[0].ShopID)
	assert.Equal(t, identity.RoleOwner, got[0].Role)
}

func TestMembershipRepository_ListByUser_Empty(t *testing.T) {
	pool := openTestPool(t)
	users := repository.NewUserRepository(pool)
	memberships := repository.NewMembershipRepository(pool)

	sfx := uniqueSuffix()
	user, err := users.Create(context.Background(), identity.User{
		Email:        "nomembership+" + sfx + "@test.com",
		PasswordHash: "$2a$04$placeholder",
		FullName:     "No Membership",
	})
	require.NoError(t, err)

	got, err := memberships.ListByUser(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestService_Login_EndToEnd(t *testing.T) {
	pool := openTestPool(t)
	svc := newIdentityService(pool)

	sfx := uniqueSuffix()
	signUpReq := identity.SignUpRequest{
		Shop:  identity.ShopInput{Name: "Login E2E Barber " + sfx, State: "SP", City: "São Paulo"},
		Owner: identity.OwnerInput{Email: "login-e2e+" + sfx + "@test.com", Password: "SecretPass123", FullName: "Login Owner"},
	}

	var signUpResp identity.SignUpResponse
	require.NoError(t, runInTx(t, pool, func(ctx context.Context) error {
		var err error
		signUpResp, err = svc.SignUp(ctx, signUpReq)
		return err
	}))

	// Happy path: correct credentials return a token plus the owner's primary shop/role.
	loginResp, err := svc.Login(context.Background(), identity.LoginRequest{
		Email:    signUpReq.Owner.Email,
		Password: signUpReq.Owner.Password,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, loginResp.Token)
	assert.Equal(t, signUpResp.Shop.ID, loginResp.ShopID)
	assert.Equal(t, identity.RoleOwner, loginResp.Role)
	assert.Equal(t, signUpReq.Owner.Email, loginResp.User.Email)

	claims, err := identity.ParseToken("integration-test-secret", loginResp.Token)
	require.NoError(t, err)
	assert.Equal(t, signUpResp.Owner.ID, claims.Subject)
	assert.Equal(t, signUpResp.Shop.ID, claims.ShopID)
	assert.Equal(t, string(identity.RoleOwner), claims.Role)

	// Wrong password: same generic error as an unknown email (no enumeration).
	_, err = svc.Login(context.Background(), identity.LoginRequest{
		Email:    signUpReq.Owner.Email,
		Password: "WrongPassword123",
	})
	assert.ErrorIs(t, err, identity.ErrInvalidCredentials)
}
