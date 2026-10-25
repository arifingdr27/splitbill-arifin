package port

import (
	"context"
	"encoding/json"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/google/uuid"
)

type ShareRepository interface {
	Create(ctx context.Context, createdBy uuid.UUID, code string, payload json.RawMessage, expiresAt *time.Time) (*domain.Share, error)
	GetByCode(ctx context.Context, code string) (*domain.Share, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Share, error)
}
