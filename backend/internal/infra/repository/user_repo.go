package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gcollin65/barbershop/internal/database"
	"github.com/gcollin65/barbershop/internal/identity"
)

// UserRepository is the PostgreSQL implementation of identity.UserRepository.
type UserRepository struct{ pool *pgxpool.Pool }

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user identity.User) (identity.User, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`INSERT INTO "user" (email, password_hash, full_name, phone)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id::text, created_at, updated_at`,
		user.Email, user.PasswordHash, user.FullName, nullText(user.Phone),
	).Scan(&user.ID, &createdAt, &updatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if pgErr.ConstraintName == "user_email_unique" {
				return identity.User{}, identity.ErrEmailTaken
			}
		}
		return identity.User{}, fmt.Errorf("user create: %w", err)
	}
	user.CreatedAt = createdAt.Unix()
	user.UpdatedAt = updatedAt.Unix()
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (identity.User, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var user identity.User
	var phone pgtype.Text
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id::text, email, password_hash, full_name, phone, created_at, updated_at
		 FROM "user" WHERE id = $1::uuid`, id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &phone,
		&createdAt, &updatedAt)
	if err != nil {
		if isNotFound(err) {
			return identity.User{}, identity.ErrNotFound
		}
		return identity.User{}, fmt.Errorf("user get by id: %w", err)
	}
	user.Phone = phone.String
	user.CreatedAt = createdAt.Unix()
	user.UpdatedAt = updatedAt.Unix()
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (identity.User, error) {
	q := database.QuerierFromCtx(ctx, r.pool)
	var user identity.User
	var phone pgtype.Text
	var createdAt, updatedAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id::text, email, password_hash, full_name, phone, created_at, updated_at
		 FROM "user" WHERE email = $1`, email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &phone,
		&createdAt, &updatedAt)
	if err != nil {
		if isNotFound(err) {
			return identity.User{}, identity.ErrNotFound
		}
		return identity.User{}, fmt.Errorf("user get by email: %w", err)
	}
	user.Phone = phone.String
	user.CreatedAt = createdAt.Unix()
	user.UpdatedAt = updatedAt.Unix()
	return user, nil
}
