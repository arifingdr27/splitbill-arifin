package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	maxSharePayloadBytes = 256 * 1024
	defaultShareTTLDays  = 30
)

type ShareService struct {
	shares  port.ShareRepository
	ttlDays int
	log     *logrus.Logger
}

func NewShareService(shares port.ShareRepository, ttlDays int, log *logrus.Logger) *ShareService {
	if ttlDays < 1 {
		ttlDays = defaultShareTTLDays
	}
	return &ShareService{shares: shares, ttlDays: ttlDays, log: log}
}

func (s *ShareService) Create(ctx context.Context, userID uuid.UUID, payload json.RawMessage) (*domain.CreateShareResult, error) {
	if len(payload) == 0 || !json.Valid(payload) {
		return nil, domain.ErrSharePayloadInvalid
	}
	if len(payload) > maxSharePayloadBytes {
		return nil, domain.ErrPayloadTooLarge
	}

	var probe any
	if err := json.Unmarshal(payload, &probe); err != nil {
		return nil, domain.ErrSharePayloadInvalid
	}
	obj, ok := probe.(map[string]any)
	if !ok || len(obj) == 0 {
		return nil, domain.ErrSharePayloadInvalid
	}

	expiresAt := time.Now().UTC().Add(time.Duration(s.ttlDays) * 24 * time.Hour)
	share, err := s.shares.Create(ctx, userID, payload, &expiresAt)
	if err != nil {
		s.log.WithError(err).Error("create share failed")
		return nil, err
	}
	return &domain.CreateShareResult{ID: share.ID.String()}, nil
}

func (s *ShareService) Get(ctx context.Context, id uuid.UUID) (*domain.ShareView, error) {
	share, err := s.shares.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if share.ExpiresAt != nil && time.Now().UTC().After(share.ExpiresAt.UTC()) {
		return nil, domain.ErrShareNotFound
	}
	return &domain.ShareView{
		ID:        share.ID.String(),
		Payload:   share.Payload,
		CreatedAt: share.CreatedAt,
		ExpiresAt: share.ExpiresAt,
	}, nil
}
