package identity

import "context"

type contextKey struct{}

// WithTenant returns a new context carrying the active shop (tenant) ID.
func WithTenant(ctx context.Context, shopID [16]byte) context.Context {
	return context.WithValue(ctx, contextKey{}, shopID)
}

// TenantFromCtx retrieves the tenant ID from ctx.
// Returns the zero UUID and false if not set.
func TenantFromCtx(ctx context.Context) ([16]byte, bool) {
	v, ok := ctx.Value(contextKey{}).([16]byte)
	return v, ok
}
