package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShareRepo struct {
	pool *pgxpool.Pool
}

func NewShareRepo(pool *pgxpool.Pool) *ShareRepo {
	return &ShareRepo{pool: pool}
}

func (r *ShareRepo) Create(ctx context.Context, createdBy uuid.UUID, payload json.RawMessage, expiresAt *time.Time) (*domain.Share, error) {
	const q = `
INSERT INTO shares (created_by, payload, expires_at)
VALUES ($1, $2, $3)
RETURNING id, created_by, payload, created_at, expires_at`

	var s domain.Share
	err := r.pool.QueryRow(ctx, q, createdBy, payload, expiresAt).Scan(
		&s.ID, &s.CreatedBy, &s.Payload, &s.CreatedAt, &s.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}
	return &s, nil
}

func (r *ShareRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Share, error) {
	const q = `
SELECT id, created_by, payload, created_at, expires_at
FROM shares
WHERE id = $1`

	var s domain.Share
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&s.ID, &s.CreatedBy, &s.Payload, &s.CreatedAt, &s.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrShareNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get share: %w", err)
	}
	return &s, nil
}
