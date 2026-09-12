package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            uuid.UUID
	Email         string
	Name          string
	GoogleSub     string
	CreditBalance int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UsageMonthly struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	PeriodYM  string
	FreeUsed  int
	FreeLimit int
}

type QuotaSnapshot struct {
	Period         string `json:"period"`
	FreeLimit      int    `json:"free_limit"`
	FreeUsed       int    `json:"free_used"`
	FreeRemaining  int    `json:"free_remaining"`
	CreditBalance  int    `json:"credit_balance"`
	TotalRemaining int    `json:"total_remaining"`
}

type AuthResult struct {
	Token string `json:"token"`
	User  AuthUser `json:"user"`
}

type AuthUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GoogleIdentity struct {
	Sub   string
	Email string
	Name  string
}
