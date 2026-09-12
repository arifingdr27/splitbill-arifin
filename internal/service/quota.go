package service

import (
	"context"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

var wib = time.FixedZone("WIB", 7*3600)

type QuotaService struct {
	pool      *pgxpool.Pool
	users     port.UserRepository
	usage     port.UsageRepository
	freeLimit int
	log       *logrus.Logger
}

func NewQuotaService(pool *pgxpool.Pool, users port.UserRepository, usage port.UsageRepository, freeLimit int, log *logrus.Logger) *QuotaService {
	if freeLimit < 1 {
		freeLimit = 5
	}
	return &QuotaService{pool: pool, users: users, usage: usage, freeLimit: freeLimit, log: log}
}

func CurrentPeriodYM(now time.Time) string {
	return now.In(wib).Format("2006-01")
}

func (s *QuotaService) Snapshot(ctx context.Context, userID uuid.UUID) (*domain.QuotaSnapshot, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	user, err := s.users.GetByIDForUpdate(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	period := CurrentPeriodYM(time.Now())
	usage, err := s.usage.GetOrCreateForUpdate(ctx, tx, userID, period, s.freeLimit)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return buildSnapshot(period, usage, user.CreditBalance), nil
}

func (s *QuotaService) AssertAvailable(ctx context.Context, userID uuid.UUID) (*domain.QuotaSnapshot, error) {
	snap, err := s.Snapshot(ctx, userID)
	if err != nil {
		return nil, err
	}
	if snap.TotalRemaining <= 0 {
		return snap, domain.ErrQuotaExceeded
	}
	return snap, nil
}

// DebitOnSuccess subtracts 1 credit first, else increments free_used. Call only after OCR 200.
func (s *QuotaService) DebitOnSuccess(ctx context.Context, userID uuid.UUID) (*domain.QuotaSnapshot, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	user, err := s.users.GetByIDForUpdate(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	period := CurrentPeriodYM(time.Now())
	usage, err := s.usage.GetOrCreateForUpdate(ctx, tx, userID, period, s.freeLimit)
	if err != nil {
		return nil, err
	}

	freeRemaining := usage.FreeLimit - usage.FreeUsed
	if freeRemaining < 0 {
		freeRemaining = 0
	}
	total := user.CreditBalance + freeRemaining
	if total <= 0 {
		return buildSnapshot(period, usage, user.CreditBalance), domain.ErrQuotaExceeded
	}

	source := "free"
	if user.CreditBalance > 0 {
		source = "credit"
		if err := s.users.UpdateCreditBalance(ctx, tx, userID, user.CreditBalance-1); err != nil {
			return nil, err
		}
		if err := s.usage.InsertCreditLedger(ctx, tx, userID, -1, "ocr", ""); err != nil {
			return nil, err
		}
		user.CreditBalance--
	} else {
		if err := s.usage.IncrementFreeUsed(ctx, tx, usage.ID); err != nil {
			return nil, err
		}
		usage.FreeUsed++
	}
	if err := s.usage.InsertOCREvent(ctx, tx, userID, source, "success"); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	snap := buildSnapshot(period, usage, user.CreditBalance)
	s.log.WithFields(logrus.Fields{
		"user_id":         userID.String(),
		"debit_source":    source,
		"quota_remaining": snap.TotalRemaining,
	}).Info("ocr quota debited")
	return snap, nil
}

func buildSnapshot(period string, usage *domain.UsageMonthly, credit int) *domain.QuotaSnapshot {
	freeRemaining := usage.FreeLimit - usage.FreeUsed
	if freeRemaining < 0 {
		freeRemaining = 0
	}
	return &domain.QuotaSnapshot{
		Period:         period,
		FreeLimit:      usage.FreeLimit,
		FreeUsed:       usage.FreeUsed,
		FreeRemaining:  freeRemaining,
		CreditBalance:  credit,
		TotalRemaining: credit + freeRemaining,
	}
}
