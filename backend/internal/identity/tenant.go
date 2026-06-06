package identity

import "context"

type contextKey struct{}

// WithTenant returns a new context carrying the active shop (tenant) ID.
func WithTenant(ctx context.Context, shopID string) context.Context {
	return context.WithValue(ctx, contextKey{}, shopID)
}

// TenantFromCtx retrieves the tenant ID from ctx.
// Returns an empty string and false if not set.
func TenantFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(contextKey{}).(string)
	return v, ok
}
