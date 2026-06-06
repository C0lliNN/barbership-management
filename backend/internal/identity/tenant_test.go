package identity_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gcollin65/barbershop/internal/identity"
)

func TestWithTenant_roundTrip(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	ctx := identity.WithTenant(context.Background(), id)
	got, ok := identity.TenantFromCtx(ctx)
	require.True(t, ok)
	assert.Equal(t, id, got)
}

func TestTenantFromCtx_missing(t *testing.T) {
	_, ok := identity.TenantFromCtx(context.Background())
	assert.False(t, ok)
}
