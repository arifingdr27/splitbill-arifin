package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) UpsertGoogleUser(ctx context.Context, identity domain.GoogleIdentity) (*domain.User, error) {
	const q = `
INSERT INTO users (email, name, google_sub)
VALUES ($1, $2, $3)
ON CONFLICT (google_sub) DO UPDATE
SET email = EXCLUDED.email,
    name = EXCLUDED.name,
    updated_at = NOW()
RETURNING id, email, name, COALESCE(google_sub, ''), credit_balance, created_at, updated_at`

	var u domain.User
	err := r.pool.QueryRow(ctx, q, identity.Email, identity.Name, identity.Sub).Scan(
		&u.ID, &u.Email, &u.Name, &u.GoogleSub, &u.CreditBalance, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert google user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
SELECT id, email, name, COALESCE(google_sub, ''), credit_balance, created_at, updated_at
FROM users WHERE id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.GoogleSub, &u.CreditBalance, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.User, error) {
	const q = `
SELECT id, email, name, COALESCE(google_sub, ''), credit_balance, created_at, updated_at
FROM users WHERE id = $1 FOR UPDATE`
	var u domain.User
	err := tx.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.Name, &u.GoogleSub, &u.CreditBalance, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) UpdateCreditBalance(ctx context.Context, tx pgx.Tx, id uuid.UUID, balance int) error {
	_, err := tx.Exec(ctx, `UPDATE users SET credit_balance = $2, updated_at = $3 WHERE id = $1`, id, balance, time.Now().UTC())
	return err
}
