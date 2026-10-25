package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Share struct {
	ID        uuid.UUID
	Code      string
	CreatedBy *uuid.UUID
	Payload   json.RawMessage
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type CreateShareResult struct {
	ID string `json:"id"`
}

type ShareView struct {
	ID        string          `json:"id"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt *time.Time      `json:"expires_at,omitempty"`
}
