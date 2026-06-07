//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gcollin65/barbershop/internal/identity"
	"github.com/gcollin65/barbershop/internal/infra/repository"
)

func TestShopRepository_Update(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewShopRepository(pool)

	sfx := uniqueSuffix()
	created, err := repo.Create(context.Background(), identity.Shop{
		Name: "Update Shop " + sfx,
		Slug: "update-shop-" + sfx,
	})
	require.NoError(t, err)

	updated, err := repo.Update(context.Background(), identity.Shop{
		ID:      created.ID,
		Phone:   "+5511988887777",
		Address: "Rua Augusta, 123",
		City:    "São Paulo",
		State:   "SP",
	})
	require.NoError(t, err)
	assert.Equal(t, "+5511988887777", updated.Phone)
	assert.Equal(t, "Rua Augusta, 123", updated.Address)
	assert.Equal(t, "São Paulo", updated.City)
	assert.Equal(t, "SP", updated.State)
	assert.Equal(t, created.Name, updated.Name, "name must be untouched")
	assert.Equal(t, created.Slug, updated.Slug, "slug must be untouched")
	assert.GreaterOrEqual(t, updated.UpdatedAt, created.UpdatedAt)

	got, err := repo.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "+5511988887777", got.Phone)
	assert.Equal(t, "São Paulo", got.City)
}

func TestShopRepository_Update_NotFound(t *testing.T) {
	pool := openTestPool(t)
	repo := repository.NewShopRepository(pool)

	_, err := repo.Update(context.Background(), identity.Shop{
		ID: "00000000-0000-0000-0000-000000000000", City: "Nowhere",
	})
	assert.ErrorIs(t, err, identity.ErrNotFound)
}

// TestCrossTenantIsolation_MembershipLookup is the headline Stage 1 proof:
// seed two shops with distinct owners, then verify that the very query
// RequireShopMembership relies on (shop_id + user_id) cannot return a
// membership across tenant boundaries — isolation is enforced at the SQL
// level, not just by application logic.
func TestCrossTenantIsolation_MembershipLookup(t *testing.T) {
	pool := openTestPool(t)
	svc := newIdentityService(pool)

	sfxA, sfxB := uniqueSuffix(), uniqueSuffix()
	var ownerA, ownerB identity.SignUpResponse

	require.NoError(t, runInTx(t, pool, func(ctx context.Context) error {
		var err error
		ownerA, err = svc.SignUp(ctx, identity.SignUpRequest{
			Shop:  identity.ShopInput{Name: "Isolation Shop A " + sfxA, City: "São Paulo", State: "SP"},
			Owner: identity.OwnerInput{Email: "iso-a+" + sfxA + "@test.com", Password: "SecretPass123", FullName: "Owner A"},
		})
		return err
	}))
	require.NoError(t, runInTx(t, pool, func(ctx context.Context) error {
		var err error
		ownerB, err = svc.SignUp(ctx, identity.SignUpRequest{
			Shop:  identity.ShopInput{Name: "Isolation Shop B " + sfxB, City: "Rio de Janeiro", State: "RJ"},
			Owner: identity.OwnerInput{Email: "iso-b+" + sfxB + "@test.com", Password: "SecretPass123", FullName: "Owner B"},
		})
		return err
	}))

	memberships := repository.NewMembershipRepository(pool)

	// Owner A has a membership in Shop A...
	m, err := memberships.GetByShopAndUser(context.Background(), ownerA.Shop.ID, ownerA.Owner.ID)
	require.NoError(t, err)
	assert.Equal(t, identity.RoleOwner, m.Role)

	// ...but looking up Owner A against Shop B finds nothing — the
	// `WHERE shop_id = $1 AND user_id = $2` predicate cannot leak across shops.
	_, err = memberships.GetByShopAndUser(context.Background(), ownerB.Shop.ID, ownerA.Owner.ID)
	assert.ErrorIs(t, err, identity.ErrNotFound)

	// ...and the symmetric case: Owner B has no membership in Shop A.
	_, err = memberships.GetByShopAndUser(context.Background(), ownerA.Shop.ID, ownerB.Owner.ID)
	assert.ErrorIs(t, err, identity.ErrNotFound)
}
