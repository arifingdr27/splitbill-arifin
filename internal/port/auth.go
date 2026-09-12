package port

import (
	"context"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepository interface {
	UpsertGoogleUser(ctx context.Context, identity domain.GoogleIdentity) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.User, error)
	UpdateCreditBalance(ctx context.Context, tx pgx.Tx, id uuid.UUID, balance int) error
}

type UsageRepository interface {
	GetOrCreateForUpdate(ctx context.Context, tx pgx.Tx, userID uuid.UUID, periodYM string, freeLimit int) (*domain.UsageMonthly, error)
	IncrementFreeUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	InsertOCREvent(ctx context.Context, tx pgx.Tx, userID uuid.UUID, source, status string) error
	InsertCreditLedger(ctx context.Context, tx pgx.Tx, userID uuid.UUID, delta int, reason, refID string) error
}

type GoogleTokenVerifier interface {
	Verify(ctx context.Context, idToken string) (*domain.GoogleIdentity, error)
}

type TokenIssuer interface {
	Issue(user *domain.User) (string, error)
	Parse(token string) (userID uuid.UUID, email string, err error)
}
