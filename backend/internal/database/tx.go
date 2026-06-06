package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

// WithTx stores tx in ctx so repository methods can retrieve it.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromCtx retrieves the active transaction from ctx, if any.
func TxFromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}

// QuerierFromCtx returns the active transaction when one is present in ctx,
// falling back to pool for non-transactional operations.
func QuerierFromCtx(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := TxFromCtx(ctx); ok {
		return tx
	}
	return pool
}
