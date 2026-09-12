package jwtauth

import (
	"fmt"
	"time"

	"github.com/arifin2018/splitbill-arifin.git/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttlDays int) *Issuer {
	if ttlDays < 1 {
		ttlDays = 30
	}
	return &Issuer{
		secret: []byte(secret),
		ttl:    time.Duration(ttlDays) * 24 * time.Hour,
	}
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (i *Issuer) Issue(user *domain.User) (string, error) {
	now := time.Now()
	c := claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(i.secret)
}

func (i *Issuer) Parse(token string) (uuid.UUID, string, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return i.secret, nil
	})
	if err != nil || !parsed.Valid {
		return uuid.Nil, "", domain.ErrUnauthorized
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || c.Subject == "" {
		return uuid.Nil, "", domain.ErrUnauthorized
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return uuid.Nil, "", domain.ErrUnauthorized
	}
	return id, c.Email, nil
}
