package postgres

import (
	"context"
	"fmt"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsageRepo struct {
	pool *pgxpool.Pool
}

func NewUsageRepo(pool *pgxpool.Pool) *UsageRepo {
	return &UsageRepo{pool: pool}
}

func (r *UsageRepo) GetOrCreateForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, periodYM string, freeLimit int) (*domain.UsageMonthly, error) {
	const upsert = `
INSERT INTO usage_monthly (user_id, period_ym, free_used, free_limit)
VALUES ($1, $2, 0, $3)
ON CONFLICT (user_id, period_ym) DO NOTHING`

	if _, err := tx.Exec(ctx, upsert, userID, periodYM, freeLimit); err != nil {
		return nil, fmt.Errorf("ensure usage_monthly: %w", err)
	}

	const q = `
SELECT id, user_id, period_ym, free_used, free_limit
FROM usage_monthly
WHERE user_id = $1 AND period_ym = $2
FOR UPDATE`

	var u domain.UsageMonthly
	err := tx.QueryRow(ctx, q, userID, periodYM).Scan(&u.ID, &u.UserID, &u.PeriodYM, &u.FreeUsed, &u.FreeLimit)
	if err != nil {
		return nil, fmt.Errorf("lock usage_monthly: %w", err)
	}
	return &u, nil
}

func (r *UsageRepo) IncrementFreeUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE usage_monthly SET free_used = free_used + 1 WHERE id = $1`, id)
	return err
}

func (r *UsageRepo) InsertOCREvent(ctx context.Context, tx pgx.Tx, userID uuid.UUID, source, status string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO ocr_usage_events (user_id, source, status) VALUES ($1, $2, $3)`,
		userID, source, status,
	)
	return err
}

func (r *UsageRepo) InsertCreditLedger(ctx context.Context, tx pgx.Tx, userID uuid.UUID, delta int, reason, refID string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO credit_ledger (user_id, delta, reason, ref_id) VALUES ($1, $2, $3, $4)`,
		userID, delta, reason, refID,
	)
	return err
}
