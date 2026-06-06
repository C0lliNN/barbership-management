package identity_test

import (
	"context"
	"testing"

	"github.com/gcollin65/barbershop/internal/identity"
)

func TestWithTenant_roundTrip(t *testing.T) {
	var id [16]byte
	id[15] = 42
	ctx := identity.WithTenant(context.Background(), id)
	got, ok := identity.TenantFromCtx(ctx)
	if !ok || got != id {
		t.Fatalf("got %v ok=%v, want %v ok=true", got, ok, id)
	}
}

func TestTenantFromCtx_missing(t *testing.T) {
	_, ok := identity.TenantFromCtx(context.Background())
	if ok {
		t.Fatal("expected ok=false for context without tenant")
	}
}
