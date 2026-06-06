package identity

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgShopRepo struct{ pool *pgxpool.Pool }

func NewShopRepository(pool *pgxpool.Pool) ShopRepository { return &pgShopRepo{pool: pool} }

func (r *pgShopRepo) Create(ctx context.Context, shop *Shop) error              { panic("not implemented") }
func (r *pgShopRepo) GetByID(ctx context.Context, id [16]byte) (*Shop, error)   { panic("not implemented") }
func (r *pgShopRepo) GetBySlug(ctx context.Context, slug string) (*Shop, error) { panic("not implemented") }
