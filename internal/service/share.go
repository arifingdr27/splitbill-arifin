package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/arifin2018/splitbill-arifin.git/internal/port"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	maxSharePayloadBytes = 256 * 1024
	defaultShareTTLDays  = 30
	shareCodeAlphabet    = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	shareCodeLength      = 8
	shareCreateAttempts  = 5
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

func generateShareCode() (string, error) {
	buf := make([]byte, shareCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	var b strings.Builder
	b.Grow(shareCodeLength)
	for _, v := range buf {
		b.WriteByte(shareCodeAlphabet[int(v)%len(shareCodeAlphabet)])
	}
	return b.String(), nil
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

	var lastErr error
	for attempt := 0; attempt < shareCreateAttempts; attempt++ {
		code, err := generateShareCode()
		if err != nil {
			return nil, err
		}
		share, err := s.shares.Create(ctx, userID, code, payload, &expiresAt)
		if err == nil {
			return &domain.CreateShareResult{ID: share.Code}, nil
		}
		lastErr = err
		if !errors.Is(err, domain.ErrShareCodeConflict) {
			s.log.WithError(err).Error("create share failed")
			return nil, err
		}
	}

	s.log.WithError(lastErr).Error("create share exhausted code retries")
	return nil, lastErr
}

func (s *ShareService) Get(ctx context.Context, publicID string) (*domain.ShareView, error) {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return nil, domain.ErrShareNotFound
	}

	var (
		share *domain.Share
		err   error
	)
	if id, parseErr := uuid.Parse(publicID); parseErr == nil {
		share, err = s.shares.GetByID(ctx, id)
	} else {
		share, err = s.shares.GetByCode(ctx, publicID)
	}
	if err != nil {
		return nil, err
	}
	if share.ExpiresAt != nil && time.Now().UTC().After(share.ExpiresAt.UTC()) {
		return nil, domain.ErrShareNotFound
	}

	viewID := share.Code
	if viewID == "" {
		viewID = share.ID.String()
	}
	return &domain.ShareView{
		ID:        viewID,
		Payload:   share.Payload,
		CreatedAt: share.CreatedAt,
		ExpiresAt: share.ExpiresAt,
	}, nil
}
