package identity

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type pgUserRepo struct{ pool *pgxpool.Pool }

func NewUserRepository(pool *pgxpool.Pool) UserRepository { return &pgUserRepo{pool: pool} }

func (r *pgUserRepo) Create(ctx context.Context, user *User) error               { panic("not implemented") }
func (r *pgUserRepo) GetByID(ctx context.Context, id [16]byte) (*User, error)    { panic("not implemented") }
func (r *pgUserRepo) GetByEmail(ctx context.Context, email string) (*User, error) { panic("not implemented") }
